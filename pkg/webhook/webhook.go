package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/1Solon/shadow-empire-pbem-bot/pkg/types"
)

// prepareWebhookURL adds the wait=true parameter to the webhook URL
func prepareWebhookURL(webhookURL string) (string, error) {
	if webhookURL == "" {
		log.Println("❌ DISCORD_WEBHOOK_URL is not configured")
		return "", fmt.Errorf("webhook URL not set")
	}

	// Add wait=true query parameter to ensure webhook delivery confirmation
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return "", fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Add the wait=true parameter
	q := parsedURL.Query()
	q.Set("wait", "true")
	parsedURL.RawQuery = q.Encode()

	return parsedURL.String(), nil
}

// sendDiscordWebhook sends a webhook with retry logic and status code handling
func sendDiscordWebhook(payload *types.DiscordWebhook, username, discordID string, isRename bool, cfg types.Config) error {
	webhookURL, err := prepareWebhookURL(cfg.WebhookURL)
	if err != nil {
		return err
	}

	// Marshal JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	// HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Add retry logic
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Send request
		req, _ := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("❌ Attempt %d: Failed to send Discord notification: %v\n", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return fmt.Errorf("failed to send Discord notification after %d attempts: %w", maxRetries, err)
		}

		// Read response body for debugging
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Handle different status codes
		switch resp.StatusCode {
		case 204:
			msgType := "notification"
			if isRename {
				msgType = "rename notification"
			}
			log.Printf("ℹ️ Discord returned status 204 for %s to %s (%s)\n", msgType, username, maskID(discordID))
			log.Printf("ℹ️ This usually means the webhook was accepted but verify it appeared in Discord\n")
			return nil
		case 200:
			msgType := ""
			if isRename {
				msgType = "Rename "
			}
			log.Printf("✅ %snotification sent to %s (%s) successfully\n", msgType, username, maskID(discordID))
			return nil
		case 429:
			// Rate limit handling with Retry-After/X-RateLimit-Reset-After
			retryAfter := parseRetryAfter(resp.Header)
			log.Printf("⚠️ Attempt %d: Discord rate limit hit (429). Waiting %v before retry. Response: %s\n", attempt, retryAfter, string(body))
			if attempt < maxRetries {
				time.Sleep(retryAfter)
				continue
			}
			return fmt.Errorf("discord rate limit exceeded after %d attempts", maxRetries)
		default:
			log.Printf("❌ Attempt %d: Discord returned unexpected status %d. Response: %s\n",
				attempt, resp.StatusCode, string(body))
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return fmt.Errorf("discord returned status %d after %d attempts: %s",
				resp.StatusCode, maxRetries, string(body))
		}
	}

	return fmt.Errorf("failed to send Discord notification after %d attempts", maxRetries)
}

// SendWebHook sends a Discord webhook notification to the next player
// targetUsername/targetDiscordID: The player whose turn it is now (will be pinged)
// nextPlayerSaveName: The username of the player *after* the target player (used for save instructions)
func SendWebHook(targetUsername, targetDiscordID, nextPlayerSaveName string, turnNumber int, cfg types.Config) error {
	gameName := cfg.GameName

	// Create webhook payload
	payload := types.DiscordWebhook{
		Username:  "Shadow Empire Assistant",
		AvatarURL: "https://raw.githubusercontent.com/auricom/home-ops/main/docs/src/assets/logo.png",
		Content:   fmt.Sprintf("🎲 It's your turn, <@%s>!", targetDiscordID), // Ping the target player
		Embeds: []types.Embed{
			{
				Color: 0xFFA500,
				Thumbnail: types.Thumbnail{
					URL: "https://upload.wikimedia.org/wikipedia/en/4/4f/Shadow_Empire_cover.jpg",
				},
				Fields: []types.Field{
					{
						Name: "📋 Save File Instructions",
						// Instruct to save for the player *after* the current one
						Value: fmt.Sprintf("After completing your turn, please save the file as:\n```\n%s_turn%d_%s\n```", gameName, turnNumber, nextPlayerSaveName),
					},
				},
				Footer: types.Footer{
					Text: "Made with ❤️ by Solon",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	// Pass targetUsername for logging purposes in sendDiscordWebhook
	return sendDiscordWebhook(&payload, targetUsername, targetDiscordID, false, cfg)
}

// SendRenameWebHook sends a Discord webhook notification asking to rename a file
func SendRenameWebHook(username, discordID, filename string, turnNumber int, cfg types.Config) error {
	gameName := cfg.GameName

	// Create webhook payload
	payload := types.DiscordWebhook{
		Username:  "Shadow Empire Assistant",
		AvatarURL: "https://raw.githubusercontent.com/auricom/home-ops/main/docs/src/assets/logo.png",
		Content:   fmt.Sprintf("⚠️ File naming issue detected in your save, <@%s>!", discordID),
		Embeds: []types.Embed{
			{
				Color: 0xFF0000, // Red color for warning
				Thumbnail: types.Thumbnail{
					URL: "https://upload.wikimedia.org/wikipedia/en/4/4f/Shadow_Empire_cover.jpg",
				},
				Fields: []types.Field{
					{
						Name: "📋 File Rename Required",
						Value: fmt.Sprintf("The save file you created `%s` doesn't match the configured game name.\n\nPlease rename it to follow the format:\n```\n%s_turn%d_%s\n```\n*(Replace %s with the next player's name)*",
							filename, gameName, turnNumber, "[NextPlayerName]", "[NextPlayerName]"),
					},
				},
				Footer: types.Footer{
					Text: "Made with ❤️ by Solon",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return sendDiscordWebhook(&payload, username, discordID, true, cfg)
}

// SendReminderWebHook sends a Discord webhook notification reminding a player it's their turn
func SendReminderWebHook(username, discordID, nextPlayerSaveName string, turnNumber int, minutesElapsed int, cfg types.Config) error {
	gameName := cfg.GameName

	// Format elapsed time as hours and minutes for display
	hours := minutesElapsed / 60
	minutes := minutesElapsed % 60
	var timeElapsedText string
	if hours > 0 && minutes > 0 {
		timeElapsedText = fmt.Sprintf("%d hours and %d minutes", hours, minutes)
	} else if hours > 0 {
		timeElapsedText = fmt.Sprintf("%d hours", hours)
	} else {
		timeElapsedText = fmt.Sprintf("%d minutes", minutes)
	}

	// Create webhook payload
	payload := types.DiscordWebhook{
		Username:  "Shadow Empire Assistant",
		AvatarURL: "https://raw.githubusercontent.com/auricom/home-ops/main/docs/src/assets/logo.png",
		Content:   fmt.Sprintf("⏰ Reminder! It's still your turn, <@%s>! (%s elapsed)", discordID, timeElapsedText),
		Embeds: []types.Embed{
			{
				Color: 0xFF9900, // Orange-yellow for reminder
				Thumbnail: types.Thumbnail{
					URL: "https://upload.wikimedia.org/wikipedia/en/4/4f/Shadow_Empire_cover.jpg",
				},
				Fields: []types.Field{
					{
						Name:  "📋 Save File Instructions",
						Value: fmt.Sprintf("After completing your turn, please save the file as:\n```\n%s_turn%d_%s\n```", gameName, turnNumber, nextPlayerSaveName),
					},
				},
				Footer: types.Footer{
					Text: "Made with ❤️ by Solon",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	// Pass username for logging purposes in sendDiscordWebhook
	return sendDiscordWebhook(&payload, username, discordID, false, cfg)
}

// maskID masks Discord IDs in logs
func maskID(id string) string {
	if len(id) <= 4 {
		return "****"
	}
	return "****" + id[len(id)-4:]
}

// parseRetryAfter figures out how long to wait from Discord rate limit headers
func parseRetryAfter(h http.Header) time.Duration {
	// Prefer Retry-After (seconds or date), or X-RateLimit-Reset-After (seconds)
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
		// If not integer, ignore for simplicity
	}
	if v := h.Get("X-RateLimit-Reset-After"); v != "" {
		if dur, err := strconv.ParseFloat(v, 64); err == nil {
			return time.Duration(dur * float64(time.Second))
		}
	}
	// fallback: 3 seconds
	return 3 * time.Second
}
