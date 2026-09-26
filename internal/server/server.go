package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"e5renewx/internal/config"
	"e5renewx/internal/graph"
	"e5renewx/internal/model"
	"e5renewx/internal/scheduler"
	"e5renewx/internal/store"
)

type Server struct {
	cfg   config.Config
	store *store.Store
	sched *scheduler.Scheduler
	mux   *http.ServeMux
	sess  sync.Map
}

func New(cfg config.Config, st *store.Store, sched *scheduler.Scheduler) *Server {
	s := &Server{cfg: cfg, store: st, sched: sched, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.withSecurity(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/logout", s.handleLogout)
	s.mux.HandleFunc("/api/me", s.auth(s.handleMe))
	s.mux.HandleFunc("/api/overview", s.auth(s.handleOverview))
	s.mux.HandleFunc("/api/accounts", s.auth(s.handleAccounts))
	s.mux.HandleFunc("/api/accounts/", s.auth(s.handleAccountItem))
	s.mux.HandleFunc("/api/logs", s.auth(s.handleLogs))
	s.mux.HandleFunc("/api/settings", s.auth(s.handleSettings))
	s.mux.HandleFunc("/api/catalog", s.auth(s.handleCatalog))
	s.mux.HandleFunc("/api/pardon", s.auth(s.handlePardon))
	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	f, err := os.Open("web" + path)
	if err != nil {
		http.ServeFile(w, r, "web/index.html")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.ServeFile(w, r, "web/index.html")
		return
	}
	http.ServeContent(w, r, path, stat.ModTime(), f)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now()})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.Password != s.store.Settings().AdminPassword {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "密码错误"})
		return
	}
	token := newToken()
	s.sess.Store(token, time.Now().Add(24*time.Hour))
	http.SetCookie(w, &http.Cookie{
		Name:     "e5_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("e5_session"); err == nil {
		s.sess.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "e5_session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"name": "admin",
		"site": s.store.Settings().SiteName,
	})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	accs := s.store.Accounts()
	running, paused, errors := 0, 0, 0
	success, fail := 0, 0
	for _, a := range accs {
		success += a.SuccessCount
		fail += a.FailCount
		switch a.Status {
		case model.StatusRunning:
			running++
		case model.StatusPaused:
			paused++
		default:
			errors++
		}
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts":     len(accs),
		"running":      running,
		"paused":       paused,
		"errors":       errors,
		"success":      success,
		"fail":         fail,
		"goroutines":   runtime.NumGoroutine(),
		"allocMB":      float64(ms.Alloc) / 1024 / 1024,
		"sysMB":        float64(ms.Sys) / 1024 / 1024,
		"goVersion":    runtime.Version(),
		"lastPardon":   s.store.LastPardon(),
		"lastReport":   s.store.LastReport(),
		"notice":       s.store.Settings().Notice,
		"siteName":     s.store.Settings().SiteName,
		"icpText":      s.store.Settings().ICPText,
		"icpLink":      s.store.Settings().ICPLink,
		"catalogCount": len(graph.Catalog()),
	})
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		writeJSON(w, http.StatusOK, graph.Catalog())
		return
	}
	writeJSON(w, http.StatusOK, graph.FilterByMode(mode))
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.Accounts())
	case http.MethodPost:
		s.createAccount(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		UPN         string   `json:"upn"`
		ClientID    string   `json:"clientId"`
		Secret      string   `json:"secret"`
		Tenant      string   `json:"tenant"`
		Mode        string   `json:"mode"`
		APIList     []string `json:"apiList"`
		NotifyEmail string   `json:"notifyEmail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.UPN = strings.TrimSpace(req.UPN)
	req.ClientID = strings.TrimSpace(req.ClientID)
	if req.UPN == "" || req.ClientID == "" || req.Secret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "账号、客户端 ID 和密钥必填"})
		return
	}
	mode := model.AuthMode(req.Mode)
	if mode != model.ModeApp {
		mode = model.ModeLogin
	}
	if req.Name == "" {
		req.Name = req.UPN
	}
	if len(req.APIList) == 0 {
		req.APIList = graph.DefaultIDs(string(mode))
	}
	now := time.Now()
	next := now.Add(15 * time.Second)
	acc := &model.Account{
		ID:          newToken()[:16],
		Name:        req.Name,
		UPN:         req.UPN,
		ClientID:    req.ClientID,
		Secret:      req.Secret,
		Tenant:      strings.TrimSpace(req.Tenant),
		Mode:        mode,
		APIList:     req.APIList,
		Status:      model.StatusRunning,
		NotifyEmail: req.NotifyEmail,
		CreatedAt:   now,
		UpdatedAt:   now,
		NextRunAt:   &next,
	}
	if err := s.store.PutAccount(acc); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.sched.RunNow(acc.ID)
	out, _ := s.store.Account(acc.ID)
	if out != nil {
		out.Secret = ""
		out.AccessToken = ""
		out.RefreshToken = ""
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAccountItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	id = strings.Trim(id, "/")
	parts := strings.Split(id, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	accID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch {
	case action == "" && r.Method == http.MethodGet:
		acc, ok := s.store.Account(accID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		acc.Secret = ""
		acc.AccessToken = ""
		acc.RefreshToken = ""
		writeJSON(w, http.StatusOK, acc)
	case action == "" && r.Method == http.MethodPut:
		s.updateAccount(w, r, accID)
	case action == "" && r.Method == http.MethodDelete:
		if err := s.store.DeleteAccount(accID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case action == "run" && r.Method == http.MethodPost:
		s.sched.RunNow(accID)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case action == "pause" && r.Method == http.MethodPost:
		now := time.Now()
		_ = s.store.MutateAccount(accID, func(a *model.Account) {
			a.Status = model.StatusPaused
			a.PausedAt = &now
			a.UpdatedAt = now
		})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case action == "resume" && r.Method == http.MethodPost:
		now := time.Now()
		next := now.Add(5 * time.Second)
		_ = s.store.MutateAccount(accID, func(a *model.Account) {
			a.Status = model.StatusRunning
			a.ConsecutiveFail = 0
			a.LastError = ""
			a.PausedAt = nil
			a.NextRunAt = &next
			a.UpdatedAt = now
		})
		s.sched.RunNow(accID)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Name        string   `json:"name"`
		UPN         string   `json:"upn"`
		ClientID    string   `json:"clientId"`
		Secret      string   `json:"secret"`
		Tenant      string   `json:"tenant"`
		Mode        string   `json:"mode"`
		APIList     []string `json:"apiList"`
		NotifyEmail string   `json:"notifyEmail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	err := s.store.MutateAccount(id, func(a *model.Account) {
		if req.Name != "" {
			a.Name = req.Name
		}
		if req.UPN != "" {
			a.UPN = strings.TrimSpace(req.UPN)
		}
		if req.ClientID != "" {
			a.ClientID = strings.TrimSpace(req.ClientID)
		}
		if req.Secret != "" {
			a.Secret = req.Secret
			a.AccessToken = ""
			a.RefreshToken = ""
			a.TokenExpiry = time.Time{}
		}
		a.Tenant = strings.TrimSpace(req.Tenant)
		if req.Mode == string(model.ModeApp) || req.Mode == string(model.ModeLogin) {
			a.Mode = model.AuthMode(req.Mode)
		}
		if req.APIList != nil {
			a.APIList = req.APIList
		}
		a.NotifyEmail = req.NotifyEmail
		a.UpdatedAt = time.Now()
	})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	acc, _ := s.store.Account(id)
	if acc != nil {
		acc.Secret = ""
		acc.AccessToken = ""
		acc.RefreshToken = ""
	}
	writeJSON(w, http.StatusOK, acc)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("accountId")
	writeJSON(w, http.StatusOK, s.store.Logs(accountID, 200))
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st := s.store.Settings()
		st.AdminPassword = ""
		st.SMTPPassword = ""
		writeJSON(w, http.StatusOK, st)
	case http.MethodPut:
		var req model.Settings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		err := s.store.UpdateSettings(func(st *model.Settings) {
			if req.AdminPassword != "" {
				st.AdminPassword = req.AdminPassword
			}
			st.NotifyEmail = req.NotifyEmail
			if req.SMTPHost != "" {
				st.SMTPHost = req.SMTPHost
			}
			if req.SMTPPort > 0 {
				st.SMTPPort = req.SMTPPort
			}
			if req.SMTPUser != "" {
				st.SMTPUser = req.SMTPUser
			}
			if req.SMTPPassword != "" {
				st.SMTPPassword = req.SMTPPassword
			}
			st.SMTPFrom = req.SMTPFrom
			if req.MinIntervalSec > 0 {
				st.MinIntervalSec = req.MinIntervalSec
			}
			if req.MaxIntervalSec > 0 {
				st.MaxIntervalSec = req.MaxIntervalSec
			}
			if req.MaxAPIsPerRound > 0 {
				st.MaxAPIsPerRound = req.MaxAPIsPerRound
			}
			if req.FailPauseThreshold > 0 {
				st.FailPauseThreshold = req.FailPauseThreshold
			}
			if req.AutoResumeHours > 0 {
				st.AutoResumeHours = req.AutoResumeHours
			}
			if req.PardonIntervalDays > 0 {
				st.PardonIntervalDays = req.PardonIntervalDays
			}
			if req.DailyReportHour >= 0 && req.DailyReportHour <= 23 {
				st.DailyReportHour = req.DailyReportHour
			}
			st.ICPText = req.ICPText
			if req.ICPLink != "" {
				st.ICPLink = req.ICPLink
			}
			st.Notice = req.Notice
			if req.SiteName != "" {
				st.SiteName = req.SiteName
			}
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		st := s.store.Settings()
		st.AdminPassword = ""
		st.SMTPPassword = ""
		writeJSON(w, http.StatusOK, st)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handlePardon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	_ = s.store.ResumeAll()
	_ = s.store.SetLastPardon(time.Now())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("e5_session")
		if err != nil || c.Value == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		v, ok := s.sess.Load(c.Value)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		exp, _ := v.(time.Time)
		if time.Now().After(exp) {
			s.sess.Delete(c.Value)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (s *Server) withSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func newToken() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		h := hmac.New(sha256.New, []byte(time.Now().String()))
		return hex.EncodeToString(h.Sum(nil))
	}
	return hex.EncodeToString(b)
}
