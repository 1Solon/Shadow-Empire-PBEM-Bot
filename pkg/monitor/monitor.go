package monitor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/1Solon/shadow-empire-pbem-bot/pkg/types"
	"github.com/1Solon/shadow-empire-pbem-bot/pkg/userparser"
	"github.com/1Solon/shadow-empire-pbem-bot/pkg/webhook"
)

// FileTrackingInfo stores information about when a file was first seen
type FileTrackingInfo struct {
	FirstSeen int64
	Processed bool
	LastSize  int64
}

// TurnInfo stores information about the current player's turn
type TurnInfo struct {
	StartedAt      time.Time
	Username       string
	DiscordID      string
	NextUsername   string
	TurnNumber     int
	LastRemindedAt time.Time
}

// parseIgnorePatterns parses comma-separated ignore patterns from environment variable
// helper to mask a Discord ID in logs
func maskID(id string) string {
	if len(id) <= 4 {
		return "****"
	}
	return "****" + id[len(id)-4:]
}

// shouldIgnoreFile checks if a filename contains any of the ignore patterns
func shouldIgnoreFile(filename string, ignorePatterns []string) bool {
	if len(ignorePatterns) == 0 {
		return false
	}

	lowerFilename := strings.ToLower(filename)
	for _, pattern := range ignorePatterns {
		if strings.Contains(lowerFilename, pattern) {
			return true
		}
	}
	return false
}

// hasAllowedExtension checks if the filename has one of the allowed extensions
func hasAllowedExtension(filename string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	lower := strings.ToLower(filename)
	for _, e := range exts {
		e = strings.TrimPrefix(e, ".")
		if strings.HasSuffix(lower, "."+e) {
			return true
		}
	}
	return false
}

// MonitorDirectory monitors a directory for new save files and notifies the next player
func MonitorDirectory(ctx context.Context, cfg types.Config) {
	dirPath := cfg.WatchDirectory

	// Get username to Discord ID mappings from config
	userMappings, err := userparser.ParseUsersFromString(cfg.UserMappingsRaw)
	if err != nil {
		log.Fatalf("❌ Failed to parse USER_MAPPINGS: %v. Please check the format (e.g., '1 User1 ID1,2 User2 ID2').", err)
	}

	// Parse ignore patterns from cfg
	ignorePatterns := cfg.IgnorePatterns
	if len(ignorePatterns) > 0 {
		fmt.Printf("🚫 Loaded %d ignore patterns\n", len(ignorePatterns))
	}

	// Log the parsed user mappings
	log.Printf("👥 Loaded %d user mappings:\n", len(userMappings))
	for _, mapping := range userMappings {
		log.Printf("  - Order: %d, User: %s, ID: %s\n", mapping.Order, mapping.Username, maskID(mapping.DiscordID))
	}

	// File tracking map with timestamps to implement debouncing
	fileTracker := make(map[string]*FileTrackingInfo)

	// Current turn tracking
	currentTurn := 1

	// Current player turn tracking
	var currentTurnInfo *TurnInfo = nil

	// Reminder interval from cfg
	reminderIntervalMinutes := cfg.ReminderIntervalMinutes

	// Reminder interval printed in main; skip repeating here

	// File debounce from cfg
	fileDebounceMs := cfg.FileDebounceMs
	fmt.Printf("⏱️ File debounce time set to %d seconds\n", fileDebounceMs/1000)

	// Initialize tracker with existing files as already processed
	files, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("❌ Error reading directory: %v\n", err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			lowerFilename := strings.ToLower(file.Name())
			fileTracker[lowerFilename] = &FileTrackingInfo{
				FirstSeen: time.Now().UnixMilli(),
				Processed: true,
			}
		}
	}
	log.Printf("📋 Initialized with %d existing files\n", len(fileTracker))

	// Set up polling interval
	pollInterval := time.Duration(cfg.PollIntervalSec) * time.Second

	log.Printf("👁️ Started monitoring directory: %s (polling every %v)\n", dirPath, pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	reminderIntervalDuration := time.Duration(reminderIntervalMinutes) * time.Minute

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 Shutting down monitor...")
			return
		case <-ticker.C:
			// Process directory for new files
			currentTurn, currentTurnInfo = processDirectory(dirPath, fileTracker, userMappings, fileDebounceMs, ignorePatterns, cfg, currentTurn, currentTurnInfo)

			// Check if we should send a reminder
			if currentTurnInfo != nil {
				timeSinceTurnStart := time.Since(currentTurnInfo.StartedAt)
				timeSinceLastReminder := time.Since(currentTurnInfo.LastRemindedAt)

				// If this is the first reminder or enough time has passed since the last reminder
				if (currentTurnInfo.LastRemindedAt.IsZero() && timeSinceTurnStart >= reminderIntervalDuration) ||
					(!currentTurnInfo.LastRemindedAt.IsZero() && timeSinceLastReminder >= reminderIntervalDuration) {

					// Send reminder
					minutesElapsed := int(timeSinceTurnStart.Minutes())
					hours := minutesElapsed / 60
					minutes := minutesElapsed % 60

					if hours > 0 && minutes > 0 {
						log.Printf("⏰ Sending turn reminder to %s (%s) - %d hours and %d minutes elapsed since turn start\n",
							currentTurnInfo.Username, maskID(currentTurnInfo.DiscordID), hours, minutes)
					} else if hours > 0 {
						log.Printf("⏰ Sending turn reminder to %s (%s) - %d hours elapsed since turn start\n",
							currentTurnInfo.Username, maskID(currentTurnInfo.DiscordID), hours)
					} else {
						log.Printf("⏰ Sending turn reminder to %s (%s) - %d minutes elapsed since turn start\n",
							currentTurnInfo.Username, maskID(currentTurnInfo.DiscordID), minutes)
					}

					err := webhook.SendReminderWebHook(
						currentTurnInfo.Username,
						currentTurnInfo.DiscordID,
						currentTurnInfo.NextUsername,
						currentTurnInfo.TurnNumber,
						minutesElapsed,
						cfg,
					)

					if err == nil {
						// Update the last reminded time
						currentTurnInfo.LastRemindedAt = time.Now()
					} else {
						fmt.Printf("❌ Failed to send reminder: %v\n", err)
					}
				}
			}
		}
	}
}

// extractTurnNumber attempts to extract turn number from a filename
func extractTurnNumber(filename string) int {
	// Support: PBEM1_turn1_Player, PBEM1_Player_turn1, and end-of-string variations
	lower := strings.ToLower(filename)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:^|_)turn(\d+)(?:_|$)`),         // _turn1_ or _turn1 end
		regexp.MustCompile(`(?:^|_)player_?turn(\d+)(?:_|$)`), // _player_turn1
	}
	for _, rx := range patterns {
		if m := rx.FindStringSubmatch(lower); len(m) > 1 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				return n
			}
		}
	}
	return 0
}

// processDirectory handles a single directory scan iteration
// Returns the current turn number and turn info (possibly updated)
func processDirectory(dirPath string, fileTracker map[string]*FileTrackingInfo,
	userMappings []userparser.UserMapping,
	fileDebounceMs int, ignorePatterns []string, cfg types.Config, currentTurn int, currentTurnInfo *TurnInfo) (int, *TurnInfo) {

	now := time.Now().UnixMilli()

	// Get the configured game name
	gameName := strings.ToLower(cfg.GameName)

	// Track current files to detect deleted ones
	currentFiles := make(map[string]bool)

	// Read all files in directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("❌ Error reading directory: %v\n", err)
		return currentTurn, currentTurnInfo
	}

	// Process each file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := strings.ToLower(file.Name())
		// Only process allowed extensions
		if !hasAllowedExtension(filename, cfg.AllowedExtensions) {
			continue
		}
		currentFiles[filename] = true

		// Try to extract turn number from filename
		if turnNumber := extractTurnNumber(filename); turnNumber > currentTurn {
			currentTurn = turnNumber
			fmt.Printf("🔢 Updated current turn to %d based on filename: %s\n", currentTurn, filename)
		}

		// get file size for debounce improvement
		var size int64 = 0
		if fi, err := os.Stat(filepath.Join(dirPath, file.Name())); err == nil {
			size = fi.Size()
		}

		if info, exists := fileTracker[filename]; !exists {
			// New file detected
			fmt.Printf("📄 New save file detected: %s, starting debounce period\n", filename)
			fileTracker[filename] = &FileTrackingInfo{
				FirstSeen: now,
				Processed: false,
				LastSize:  size,
			}
		} else if !info.Processed && (now-info.FirstSeen) >= int64(fileDebounceMs) {
			// File has been stable for debounce period
			// Ensure the file size hasn't changed since first seen
			if size != 0 && info.LastSize != 0 && size != info.LastSize {
				// Update size and extend debounce window
				info.LastSize = size
				info.FirstSeen = now
				fmt.Printf("⏳ File %s size changed, extending debounce window\n", filename)
				continue
			}
			fmt.Printf("⏱️ File %s stable for %ds, processing now\n", filename, fileDebounceMs/1000)

			// Check if the file should be ignored
			if shouldIgnoreFile(filename, ignorePatterns) {
				fmt.Printf("🚫 Ignoring file %s based on ignore patterns\n", filename)
				info.Processed = true
				continue
			}

			// Check if the game name in the filename matches the configured game name
			if !strings.HasPrefix(filename, gameName) {
				fmt.Printf("⚠️ File %s doesn't match configured game name '%s'\n", filename, gameName)

				// Try to find which user *might* have saved this based on filename content
				var foundUserIndex = -1 // Index in the userMappings slice
				for i, mapping := range userMappings {
					if strings.Contains(filename, strings.ToLower(mapping.Username)) {
						foundUserIndex = i
						break
					}
				}

				// Find the previous user who should be notified about the naming issue
				if foundUserIndex != -1 {
					// Determine the index of the user who *should* have saved (previous user in order)
					previousUserIndex := (foundUserIndex - 1 + len(userMappings)) % len(userMappings)
					previousUserMapping := userMappings[previousUserIndex]

					fmt.Printf("🔔 Sending rename notification to previous user %s (%s) for incorrectly named file %s\n",
						previousUserMapping.Username, maskID(previousUserMapping.DiscordID), filename)
					webhook.SendRenameWebHook(previousUserMapping.Username, previousUserMapping.DiscordID, filename, currentTurn, cfg)

				} else {
					fmt.Printf("❓ Cannot identify any user for incorrectly named file: %s. Cannot determine who to notify.\n", filename)
				}

				info.Processed = true
				continue
			}

			// Find username in filename to identify the player whose turn it *is*
			var currentPlayerIndex = -1 // Index in the userMappings slice
			for i, mapping := range userMappings {
				// Check if the filename contains the *current* player's username (case-insensitive)
				if strings.Contains(filename, strings.ToLower(mapping.Username)) {
					currentPlayerIndex = i
					break
				}
			}

			if currentPlayerIndex != -1 {
				// The user found in the filename is the *current* player
				currentUserMapping := userMappings[currentPlayerIndex]

				// Determine the index of the *next* player in the order
				nextPlayerIndex := (currentPlayerIndex + 1) % len(userMappings)
				nextUserMapping := userMappings[nextPlayerIndex]

				// Determine the index of the player who just finished (previous player)
				previousPlayerIndex := (currentPlayerIndex - 1 + len(userMappings)) % len(userMappings)
				previousUserMapping := userMappings[previousPlayerIndex]

				// Determine the turn number for the *next* save file instruction
				saveInstructionTurnNumber := currentTurn
				// Check if the *current* player (whose file we are processing) is the last in the order.
				// If so, the save instruction should be for the *next* turn.
				if currentPlayerIndex == len(userMappings)-1 {
					saveInstructionTurnNumber = currentTurn + 1
					fmt.Printf("🔄 Last player (%s) finished turn %d, next save will start turn %d\n", currentUserMapping.Username, currentTurn, saveInstructionTurnNumber)
					// Update the main turn counter *after* processing this file and determining the instruction number
					currentTurn = saveInstructionTurnNumber
				}

				fmt.Printf("🔄 Turn %d: It's %s's turn (save from %s). Next up: %s (for turn %d)\n", currentTurn, currentUserMapping.Username, previousUserMapping.Username, nextUserMapping.Username, saveInstructionTurnNumber)

				// Send webhook to the *current* player, instructing them to save for the *next* player, using the correct turn number for the save instruction
				err := webhook.SendWebHook(currentUserMapping.Username, currentUserMapping.DiscordID, nextUserMapping.Username, saveInstructionTurnNumber, cfg)

				if err == nil {
					// Update the current turn info for reminder tracking
					currentTurnInfo = &TurnInfo{
						StartedAt:      time.Now(),
						Username:       currentUserMapping.Username,
						DiscordID:      currentUserMapping.DiscordID,
						NextUsername:   nextUserMapping.Username,
						TurnNumber:     saveInstructionTurnNumber,
						LastRemindedAt: time.Time{}, // Zero time indicates no reminders sent yet
					}

					fmt.Printf("✅ Started tracking turn for %s (reminders will be sent if needed)\n", currentUserMapping.Username)
				}

				info.Processed = true
			} else {
				fmt.Printf("❓ Cannot match any user to save file: %s\n", filename)
				info.Processed = true
			}
		}
	}

	// Clean up tracking for deleted files
	for filename := range fileTracker {
		if !currentFiles[filename] {
			delete(fileTracker, filename)
			fmt.Printf("🗑️ Removed tracking for deleted file: %s\n", filename)
		}
	}

	return currentTurn, currentTurnInfo
}
