package telegram

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Token        string
	AllowedUsers map[int64]bool
}

// LoadConfig returns nil if Telegram is not configured.
func LoadConfig() *Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil
	}

	c := &Config{
		Token:        token,
		AllowedUsers: parseIDs(os.Getenv("TELEGRAM_ALLOWED_USER_IDS")),
	}

	if len(c.AllowedUsers) == 0 {
		return nil
	}

	return c
}

func (c *Config) IsAllowed(uid int64) bool {
	return c.AllowedUsers[uid]
}

func parseIDs(s string) map[int64]bool {
	m := make(map[int64]bool)
	for _, p := range strings.Split(s, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
			m[id] = true
		}
	}
	return m
}
