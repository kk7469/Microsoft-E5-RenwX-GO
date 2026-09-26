package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DataPath      string
	AdminPassword string
	SessionSecret string
}

func Load() Config {
	c := Config{
		Port:          env("PORT", "1066"),
		DataPath:      env("DATA_PATH", "data/store.json"),
		AdminPassword: env("ADMIN_PASSWORD", "123456"),
		SessionSecret: env("SESSION_SECRET", "e5renewx-go-session-secret"),
	}
	return c
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func EnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
