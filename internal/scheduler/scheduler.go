package scheduler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"e5renewx/internal/graph"
	"e5renewx/internal/mailer"
	"e5renewx/internal/model"
	"e5renewx/internal/store"
)

type Scheduler struct {
	store    *store.Store
	graph    *graph.Client
	stop     chan struct{}
	wg       sync.WaitGroup
	inflight sync.Map
}

func New(st *store.Store) *Scheduler {
	return &Scheduler{
		store: st,
		graph: graph.New(),
		stop:  make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *Scheduler) loop() {
	defer s.wg.Done()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	s.tickOnce()
	for {
		select {
		case <-s.stop:
			return
		case <-tick.C:
			s.tickOnce()
		}
	}
}

func (s *Scheduler) tickOnce() {
	now := time.Now()
	settings := s.store.Settings()
	s.maybePardon(now, settings)
	s.maybeDailyReport(now, settings)
	s.maybeAutoResume(now, settings)

	for _, acc := range s.store.Accounts() {
		full, ok := s.store.Account(acc.ID)
		if !ok {
			continue
		}
		if full.Status != model.StatusRunning {
			continue
		}
		if full.NextRunAt != nil && now.Before(*full.NextRunAt) {
			continue
		}
		go s.runAccount(full.ID)
	}
}

func (s *Scheduler) maybePardon(now time.Time, settings model.Settings) {
	days := settings.PardonIntervalDays
	if days < 1 {
		days = 30
	}
	last := s.store.LastPardon()
	if last.IsZero() {
		_ = s.store.SetLastPardon(now)
		return
	}
	if now.Sub(last) >= time.Duration(days)*24*time.Hour {
		_ = s.store.ResumeAll()
		_ = s.store.SetLastPardon(now)
	}
}

func (s *Scheduler) maybeAutoResume(now time.Time, settings model.Settings) {
	hours := settings.AutoResumeHours
	if hours < 1 {
		hours = 24
	}
	for _, acc := range s.store.Accounts() {
		if acc.Status != model.StatusPaused && acc.Status != model.StatusError {
			continue
		}
		if acc.PausedAt == nil {
			continue
		}
		if now.Sub(*acc.PausedAt) >= time.Duration(hours)*time.Hour {
			_ = s.store.MutateRuntime(acc.ID, func(a *model.Account) {
				a.Status = model.StatusRunning
				a.ConsecutiveFail = 0
				a.LastError = ""
				a.PausedAt = nil
				a.UpdatedAt = now
			})
		}
	}
}

func (s *Scheduler) maybeDailyReport(now time.Time, settings model.Settings) {
	hour := settings.DailyReportHour
	if hour < 0 || hour > 23 {
		hour = 18
	}
	if now.Hour() != hour {
		return
	}
	last := s.store.LastReport()
	if !last.IsZero() && last.Year() == now.Year() && last.YearDay() == now.YearDay() {
		return
	}
	to := settings.NotifyEmail
	if to == "" {
		_ = s.store.SetLastReport(now)
		return
	}
	var b strings.Builder
	b.WriteString("E5 RenewX GO 每日运行报告\n\n")
	accs := s.store.Accounts()
	okCount := 0
	for _, a := range accs {
		line := fmt.Sprintf("- %s (%s) 状态=%s 成功=%d 失败=%d", a.Name, a.UPN, a.Status, a.SuccessCount, a.FailCount)
		if a.LastError != "" {
			line += " 错误=" + a.LastError
		}
		b.WriteString(line + "\n")
		if a.Status == model.StatusRunning {
			okCount++
		}
	}
	b.WriteString(fmt.Sprintf("\n运行中 %d / 共 %d\n", okCount, len(accs)))
	_ = mailer.Send(settings, to, "E5 RenewX GO 每日运行报告", b.String())
	_ = s.store.SetLastReport(now)
}

func (s *Scheduler) RunNow(id string) {
	go s.runAccount(id)
}

func (s *Scheduler) runAccount(id string) {
	if _, loaded := s.inflight.LoadOrStore(id, true); loaded {
		return
	}
	defer s.inflight.Delete(id)
	acc, ok := s.store.Account(id)
	if !ok {
		return
	}
	settings := s.store.Settings()
	now := time.Now()

	if err := s.graph.EnsureToken(acc); err != nil {
		s.recordFail(acc, "token", err.Error(), settings)
		return
	}
	_ = s.store.MutateRuntime(id, func(a *model.Account) {
		a.AccessToken = acc.AccessToken
		a.RefreshToken = acc.RefreshToken
		a.TokenExpiry = acc.TokenExpiry
	})

	apis := selectedAPIs(acc)
	if len(apis) == 0 {
		s.recordFail(acc, "config", "未选择任何 API", settings)
		return
	}
	n := 1 + randInt(settings.MaxAPIsPerRound)
	if n > len(apis) {
		n = len(apis)
	}
	picked := pickN(apis, n)
	success := 0
	fail := 0
	skipped := 0
	var lastErr string
	for _, api := range picked {
		res := s.graph.Call(acc, api)
		log := model.CallLog{
			ID:        newID(),
			AccountID: acc.ID,
			APIID:     api.ID,
			APIName:   api.Name,
			Method:    res.Method,
			Path:      res.Path,
			Status:    res.Status,
			OK:        res.OK,
			Skipped:   res.Skipped,
			Message:   res.Message,
			Duration:  res.Duration.Milliseconds(),
			CreatedAt: time.Now(),
		}
		s.store.AppendLog(log)
		if res.OK {
			success++
		} else if res.Skipped {
			skipped++
		} else {
			fail++
			lastErr = res.Message
		}
	}

	next := now.Add(randomInterval(settings.MinIntervalSec, settings.MaxIntervalSec))
	_ = s.store.MutateRuntime(id, func(a *model.Account) {
		a.LastRunAt = &now
		a.NextRunAt = &next
		a.UpdatedAt = time.Now()
		a.SuccessCount += success
		a.FailCount += fail
		if fail > 0 && success == 0 {
			a.ConsecutiveFail++
			a.LastError = lastErr
			if a.ConsecutiveFail >= settings.FailPauseThreshold {
				paused := time.Now()
				a.Status = model.StatusPaused
				a.PausedAt = &paused
			}
		} else {
			a.ConsecutiveFail = 0
			if success > 0 {
				a.LastError = ""
				a.Status = model.StatusRunning
			} else if skipped > 0 {
				a.LastError = "部分 API 因权限不足已跳过"
				a.Status = model.StatusRunning
			}
		}
	})

	if fail > 0 && success == 0 {
		updated, _ := s.store.Account(id)
		if updated != nil && updated.Status == model.StatusPaused {
			s.notify(updated, settings, "账号已暂停", fmt.Sprintf("账号 %s (%s) 连续失败 %d 次，已自动暂停。\n最后错误：%s", updated.Name, updated.UPN, updated.ConsecutiveFail, lastErr))
		}
	}
}

func (s *Scheduler) recordFail(acc *model.Account, kind, msg string, settings model.Settings) {
	now := time.Now()
	s.store.AppendLog(model.CallLog{
		ID:        newID(),
		AccountID: acc.ID,
		APIID:     kind,
		APIName:   kind,
		OK:        false,
		Message:   msg,
		CreatedAt: now,
	})
	next := now.Add(randomInterval(settings.MinIntervalSec, settings.MaxIntervalSec))
	_ = s.store.MutateRuntime(acc.ID, func(a *model.Account) {
		a.LastRunAt = &now
		a.NextRunAt = &next
		a.FailCount++
		a.ConsecutiveFail++
		a.LastError = msg
		a.UpdatedAt = now
		if a.ConsecutiveFail >= settings.FailPauseThreshold {
			paused := now
			a.Status = model.StatusPaused
			a.PausedAt = &paused
		}
	})
}

func (s *Scheduler) notify(acc *model.Account, settings model.Settings, subject, body string) {
	to := acc.NotifyEmail
	if to == "" {
		to = settings.NotifyEmail
	}
	if to == "" {
		return
	}
	_ = mailer.Send(settings, to, "E5 RenewX GO: "+subject, body)
}

func selectedAPIs(acc *model.Account) []model.APIDef {
	mode := string(acc.Mode)
	if len(acc.APIList) == 0 {
		out := make([]model.APIDef, 0)
		for _, id := range graph.DefaultIDs(mode) {
			if api, ok := graph.ByID(id); ok {
				out = append(out, api)
			}
		}
		return out
	}
	out := make([]model.APIDef, 0, len(acc.APIList))
	for _, id := range acc.APIList {
		api, ok := graph.ByID(id)
		if !ok {
			continue
		}
		for _, m := range api.Modes {
			if m == mode {
				out = append(out, api)
				break
			}
		}
	}
	return out
}

func pickN(apis []model.APIDef, n int) []model.APIDef {
	cp := append([]model.APIDef(nil), apis...)
	for i := len(cp) - 1; i > 0; i-- {
		j := randInt(i + 1)
		cp[i], cp[j] = cp[j], cp[i]
	}
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

func randomInterval(minSec, maxSec int) time.Duration {
	if minSec < 10 {
		minSec = 10
	}
	if maxSec < minSec {
		maxSec = minSec
	}
	return time.Duration(minSec+randInt(maxSec-minSec+1)) * time.Second
}

func randInt(n int) int {
	if n <= 1 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return int(time.Now().UnixNano() % int64(n))
	}
	return int(v.Int64())
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
