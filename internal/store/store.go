package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"e5renewx/internal/model"
)

// logLimit 是内存中保留的调用日志条数上限。日志不落盘，进程重启即清空。
const logLimit = 200

type Store struct {
	mu       sync.RWMutex
	path     string
	settings model.Settings
	accounts map[string]*model.Account
	logs     []model.CallLog
	pardon   time.Time
	report   time.Time
}

func New(path, adminPassword string) (*Store, error) {
	s := &Store{
		path:     path,
		accounts: map[string]*model.Account{},
		logs:     []model.CallLog{},
		settings: defaultSettings(adminPassword),
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if s.settings.AdminPassword == "" {
		s.settings.AdminPassword = adminPassword
	}
	return s, nil
}

func defaultSettings(adminPassword string) model.Settings {
	return model.Settings{
		AdminPassword:      adminPassword,
		SMTPHost:           "smtp.163.com",
		SMTPPort:           465,
		MinIntervalSec:     1000,
		MaxIntervalSec:     2000,
		MaxAPIsPerRound:    3,
		FailPauseThreshold: 5,
		AutoResumeHours:    24,
		PardonIntervalDays: 7,
		DailyReportHour:    18,
		ICPLink:            "https://beian.miit.gov.cn",
		SiteName:           "Microsoft 365 E5 RenewX GO",
		Notice:             "通过定时随机调用 Microsoft Graph API，保持 E5 开发者订阅活跃。",
	}
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.saveLocked()
		}
		return err
	}
	var snap model.Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	if snap.Settings.MinIntervalSec > 0 {
		s.settings = snap.Settings
	}
	for i := range snap.Accounts {
		acc := snap.Accounts[i]
		resetAccountRuntime(&acc)
		s.accounts[acc.ID] = &acc
	}
	s.logs = []model.CallLog{} // 日志仅存内存，不从文件恢复
	s.pardon = snap.LastPardon
	s.report = snap.LastReport
	// 一次性迁移：把历史快照中尚未随代码默认值更新的"默认字段"重置为当前代码内置默认值，
	// 保证新增的默认站点名、默认特赦间隔等设置对已有数据文件也生效。
	migrated := false
	if s.settings.SiteName != "Microsoft 365 E5 RenewX GO" {
		s.settings.SiteName = defaultSettings(s.settings.AdminPassword).SiteName
		migrated = true
	}
	if s.settings.PardonIntervalDays != 7 {
		s.settings.PardonIntervalDays = 7
		migrated = true
	}
	if migrated {
		if err := s.saveLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) saveLocked() error {
	accs := make([]model.Account, 0, len(s.accounts))
	for _, a := range s.accounts {
		accs = append(accs, *a)
	}
	snap := model.Snapshot{
		Settings:   s.settings,
		Accounts:   accs,
		LastPardon: s.pardon,
		LastReport: s.report,
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) Settings() model.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) UpdateSettings(fn func(*model.Settings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.settings)
	return s.saveLocked()
}

func (s *Store) Accounts() []model.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Account, 0, len(s.accounts))
	for _, a := range s.accounts {
		cp := *a
		cp.Secret = ""
		cp.AccessToken = ""
		cp.RefreshToken = ""
		out = append(out, cp)
	}
	return out
}

func (s *Store) Account(id string) (*model.Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.accounts[id]
	if !ok {
		return nil, false
	}
	cp := *a
	return &cp, true
}

func (s *Store) PutAccount(acc *model.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[acc.ID] = acc
	return s.saveLocked()
}

func (s *Store) DeleteAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.accounts, id)
	return s.saveLocked()
}

func (s *Store) MutateAccount(id string, fn func(*model.Account)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.accounts[id]
	if !ok {
		return os.ErrNotExist
	}
	fn(a)
	return s.saveLocked()
}

// MutateRuntime 只修改账号的运行时状态（状态、成功/失败计数、令牌、调度时间等）。
// 这些字段不落盘，因此无需写文件；账号配置变更请继续使用 MutateAccount。
func (s *Store) MutateRuntime(id string, fn func(*model.Account)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.accounts[id]
	if !ok {
		return os.ErrNotExist
	}
	fn(a)
	return nil
}

// resetAccountRuntime 把账号的运行时状态重置为初始值。
// 这些字段不会从文件恢复；其中 Status 必须显式设为 running，
// 否则调度器会因状态为空字符串而永不执行该账号。
func resetAccountRuntime(a *model.Account) {
	a.Status = model.StatusRunning
	a.UpdatedAt = time.Time{}
	a.LastRunAt = nil
	a.NextRunAt = nil
	a.PausedAt = nil
	a.LastError = ""
	a.SuccessCount = 0
	a.FailCount = 0
	a.ConsecutiveFail = 0
	a.AccessToken = ""
	a.RefreshToken = ""
	a.TokenExpiry = time.Time{}
}

func (s *Store) AppendLog(log model.CallLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, log)
	if len(s.logs) > logLimit {
		s.logs = s.logs[len(s.logs)-logLimit:]
	}
	// 日志只保留在内存中，不落盘，进程重启即清空
}

func (s *Store) Logs(accountID string, limit int) []model.CallLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CallLog, 0, limit)
	for i := len(s.logs) - 1; i >= 0 && len(out) < limit; i-- {
		if accountID == "" || s.logs[i].AccountID == accountID {
			out = append(out, s.logs[i])
		}
	}
	return out
}

func (s *Store) LastPardon() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pardon
}

func (s *Store) SetLastPardon(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pardon = t
	return s.saveLocked()
}

func (s *Store) LastReport() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.report
}

func (s *Store) SetLastReport(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.report = t
	return s.saveLocked()
}

func (s *Store) ResumeAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, a := range s.accounts {
		if a.Status == model.StatusPaused || a.Status == model.StatusError {
			a.Status = model.StatusRunning
			a.ConsecutiveFail = 0
			a.LastError = ""
			a.PausedAt = nil
			a.UpdatedAt = now
		}
	}
	return s.saveLocked()
}
