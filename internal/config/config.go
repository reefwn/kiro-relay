package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	WorkDir    string
	TrustTools string
}

func Load() *Config {
	godotenv.Load()

	c := &Config{
		WorkDir:    envOr("KIRO_WORK_DIR", os.ExpandEnv("$HOME")),
		TrustTools: os.Getenv("KIRO_TRUST_TOOLS"),
	}

	slog.Info("config loaded", "workdir", c.WorkDir, "trust_tools", c.TrustTools)
	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return expandHome(v)
	}
	return fallback
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, _ := os.UserHomeDir()
		return strings.Replace(path, "~", home, 1)
	}
	return path
}
