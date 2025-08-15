package types

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration values loaded from environment variables.
type Config struct {
	// Raw values
	UserMappingsRaw      string
	GameName             string
	WebhookURL           string
	WatchDirectory       string
	IgnorePatternsRaw    string
	AllowedExtensionsRaw string

	// Parsed values
	IgnorePatterns          []string
	AllowedExtensions       []string
	FileDebounceMs          int
	ReminderIntervalMinutes int
	PollIntervalSec         int
}

// LoadConfigFromEnv reads environment variables and returns a populated Config with defaults applied.
func LoadConfigFromEnv() Config {
	cfg := Config{}

	cfg.UserMappingsRaw = os.Getenv("USER_MAPPINGS")
	cfg.GameName = firstNonEmpty(os.Getenv("GAME_NAME"), "pbem1")
	cfg.WebhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
	cfg.WatchDirectory = firstNonEmpty(os.Getenv("WATCH_DIRECTORY"), "./data")
	cfg.IgnorePatternsRaw = os.Getenv("IGNORE_PATTERNS")
	cfg.AllowedExtensionsRaw = firstNonEmpty(os.Getenv("ALLOWED_EXTENSIONS"), "se1")

	// Parse lists
	cfg.IgnorePatterns = parseCSVLower(cfg.IgnorePatternsRaw)
	cfg.AllowedExtensions = parseCSVLower(cfg.AllowedExtensionsRaw)

	// Numbers with defaults
	cfg.FileDebounceMs = parseIntOrDefault(os.Getenv("FILE_DEBOUNCE_MS"), 30000)
	cfg.ReminderIntervalMinutes = parseIntOrDefault(os.Getenv("REMINDER_INTERVAL_MINUTES"), 720)
	cfg.PollIntervalSec = parseIntOrDefault(os.Getenv("POLL_INTERVAL_SEC"), 5)

	return cfg
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func parseCSVLower(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseIntOrDefault(s string, def int) int {
	if strings.TrimSpace(s) == "" {
		return def
	}
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}
