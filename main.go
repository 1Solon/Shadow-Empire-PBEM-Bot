package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/1Solon/shadow-empire-pbem-bot/pkg/monitor"
	"github.com/1Solon/shadow-empire-pbem-bot/pkg/types"
	"github.com/joho/godotenv"
)

func main() {
	// Set log output with timestamps
	log.SetFlags(log.LstdFlags)

	// Check if required environment variables exist
	if os.Getenv("USER_MAPPINGS") == "" || os.Getenv("GAME_NAME") == "" {
		// If not, try to load from .env file
		envPath := filepath.Join(".", ".env")
		if _, err := os.Stat(envPath); err == nil {
			fmt.Println("📝 Loading environment variables from .env file")
			err := godotenv.Load()
			if err != nil {
				log.Printf("⚠️ Error loading .env file: %v", err)
			}
		} else {
			fmt.Println("⚠️ No .env file found and required environment variables not set")
		}
	} else {
		fmt.Println("🔧 Using environment variables from system")
	}

	// Load config once
	cfg := types.LoadConfigFromEnv()

	// Check if specific environment variables are set after potential loading
	if cfg.UserMappingsRaw == "" {
		fmt.Println("⚠️ USER_MAPPINGS environment variable is not set, exiting")
		os.Exit(1)
	}
	if os.Getenv("GAME_NAME") == "" { // note: cfg has default applied, call out only when env was empty
		fmt.Println("ℹ️ GAME_NAME environment variable is not set, using default: pbem1")
	}
	if cfg.WebhookURL == "" {
		fmt.Println("⚠️ DISCORD_WEBHOOK_URL environment variable is not set, webhook notifications will fail")
	}

	// Check if WATCH_DIRECTORY is set
	if os.Getenv("WATCH_DIRECTORY") == "" {
		fmt.Println("⚠️ WATCH_DIRECTORY environment variable is not set, using default: ./data")
	}

	// Check if IGNORE_PATTERNS is set
	if cfg.IgnorePatternsRaw != "" {
		fmt.Printf("🔍 Will ignore files containing patterns: %s\n", cfg.IgnorePatternsRaw)
	}

	// Check if FILE_DEBOUNCE_MS is set
	if os.Getenv("FILE_DEBOUNCE_MS") == "" {
		fmt.Println("ℹ️ FILE_DEBOUNCE_MS environment variable is not set, using default: 30000 (30 seconds)")
	} else {
		// Convert ms to seconds for display
		if ms, err := strconv.Atoi(os.Getenv("FILE_DEBOUNCE_MS")); err == nil {
			fmt.Printf("⏱️ File debounce time set to %d seconds\n", ms/1000)
		}
	}

	// Check if REMINDER_INTERVAL_MINUTES is set
	if os.Getenv("REMINDER_INTERVAL_MINUTES") == "" {
		fmt.Println("ℹ️ REMINDER_INTERVAL_MINUTES environment variable is not set, using default: 720 minutes (12 hours)")
	} else {
		minutes, _ := strconv.Atoi(os.Getenv("REMINDER_INTERVAL_MINUTES"))
		hours := minutes / 60
		mins := minutes % 60

		if hours > 0 && mins > 0 {
			fmt.Printf("⏰ Reminder interval set to %d hours and %d minutes\n", hours, mins)
		} else if hours > 0 {
			fmt.Printf("⏰ Reminder interval set to %d hours\n", hours)
		} else {
			fmt.Printf("⏰ Reminder interval set to %d minutes\n", mins)
		}
	}

	// Start monitoring the directory
	fmt.Printf("👀 Monitoring directory: %s (poll every %ds)\n", cfg.WatchDirectory, cfg.PollIntervalSec)

	// Block and monitor directory with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n🛑 Received shutdown signal, stopping...")
		cancel()
	}()

	monitor.MonitorDirectory(ctx, cfg)
}
