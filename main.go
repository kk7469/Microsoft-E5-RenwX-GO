package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"e5renewx/internal/config"
	"e5renewx/internal/model"
	"e5renewx/internal/scheduler"
	"e5renewx/internal/server"
	"e5renewx/internal/store"
)

func main() {
	cfg := config.Load()
	st, err := store.New(cfg.DataPath, cfg.AdminPassword)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	applyEnvSettings(st)

	sched := scheduler.New(st)
	sched.Start()
	defer sched.Stop()

	srv := server.New(cfg, st, sched)
	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("E5 RenewX GO listening on :%s", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	_ = httpSrv.Close()
}

func applyEnvSettings(st *store.Store) {
	_ = st.UpdateSettings(func(s *model.Settings) {
		if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
			s.AdminPassword = v
		}
		if v := os.Getenv("NOTIFY_EMAIL"); v != "" {
			s.NotifyEmail = v
		}
		if v := os.Getenv("SMTP_HOST"); v != "" {
			s.SMTPHost = v
		}
		if v := os.Getenv("SMTP_USER"); v != "" {
			s.SMTPUser = v
		}
		if v := os.Getenv("SMTP_PASSWORD"); v != "" {
			s.SMTPPassword = v
		}
		if v := os.Getenv("SMTP_FROM"); v != "" {
			s.SMTPFrom = v
		}
		if v := os.Getenv("SMTP_PORT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				s.SMTPPort = n
			}
		}
	})
}
