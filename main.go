package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/1Solon/shadow-empire-pbem-bot/pkg/monitor"
	"github.com/joho/godotenv"
)

func main() {
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

	// Check if specific environment variables are set after potential loading
	if os.Getenv("USER_MAPPINGS") == "" {
		fmt.Println("⚠️ USER_MAPPINGS environment variable is not set, exiting")
		os.Exit(1)
	}
	if os.Getenv("GAME_NAME") == "" {
		fmt.Println("ℹ️ GAME_NAME environment variable is not set, using default: pbem1")
	}
	if os.Getenv("DISCORD_WEBHOOK_URL") == "" {
		fmt.Println("⚠️ DISCORD_WEBHOOK_URL environment variable is not set, webhook notifications will fail")
	}

	// Check if WATCH_DIRECTORY is set
	if os.Getenv("WATCH_DIRECTORY") == "" {
		fmt.Println("⚠️ WATCH_DIRECTORY environment variable is not set, using default: ./data")
	}

	// Check if IGNORE_PATTERNS is set
	if os.Getenv("IGNORE_PATTERNS") != "" {
		fmt.Printf("🔍 Will ignore files containing patterns: %s\n", os.Getenv("IGNORE_PATTERNS"))
	}

	// Check if FILE_DEBOUNCE_MS is set
	if os.Getenv("FILE_DEBOUNCE_MS") == "" {
		fmt.Println("ℹ️ FILE_DEBOUNCE_MS environment variable is not set, using default: 30000 (30 seconds)")
	} else {
		fmt.Printf("⏱️ File debounce time set to %s seconds\n", os.Getenv("FILE_DEBOUNCE_MS"))
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

	// Start monitoring the directory, default to "./data"
	directoryToWatch := os.Getenv("WATCH_DIRECTORY")
	if directoryToWatch == "" {
		directoryToWatch = "./data"
	}
	fmt.Printf("👀 Monitoring directory: %s\n", directoryToWatch)

	// Block and monitor directory
	monitor.MonitorDirectory(directoryToWatch)
}
