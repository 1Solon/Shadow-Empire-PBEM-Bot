package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Discord webhook request structure
type DiscordWebhook struct {
	Username  string  `json:"username"`
	AvatarURL string  `json:"avatar_url"`
	Content   string  `json:"content"`
	Embeds    []Embed `json:"embeds"`
}

type Embed struct {
	Color     int       `json:"color"`
	Thumbnail Thumbnail `json:"thumbnail"`
	Fields    []Field   `json:"fields"`
	Footer    Footer    `json:"footer"`
	Timestamp string    `json:"timestamp"`
}

type Thumbnail struct {
	URL string `json:"url"`
}

type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Footer struct {
	Text string `json:"text"`
}

// prepareWebhookURL adds the wait=true parameter to the webhook URL
func prepareWebhookURL() (string, error) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")

	if webhookURL == "" {
		fmt.Println("❌ DISCORD_WEBHOOK_URL environment variable is not set")
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

// getGameName returns the configured game name or the default
func getGameName() string {
	gameName := os.Getenv("GAME_NAME")
	if gameName == "" {
		return "pbem1"
	}
	return gameName
}

// sendDiscordWebhook sends a webhook with retry logic and status code handling
func sendDiscordWebhook(payload *DiscordWebhook, username, discordID string, isRename bool) error {
	webhookURL, err := prepareWebhookURL()
	if err != nil {
		return err
	}

	// Marshal JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Add retry logic
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Send request
		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			fmt.Printf("❌ Attempt %d: Failed to send Discord notification: %v\n", attempt, err)
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
			fmt.Printf("ℹ️ Discord returned status 204 for %s to %s (%s)\n", msgType, username, discordID)
			fmt.Printf("ℹ️ This usually means the webhook was accepted but verify it appeared in Discord\n")
			return nil
		case 200:
			msgType := ""
			if isRename {
				msgType = "Rename "
			}
			fmt.Printf("✅ %snotification sent to %s (%s) successfully\n", msgType, username, discordID)
			return nil
		case 429:
			fmt.Printf("⚠️ Attempt %d: Discord rate limit hit (429). Response: %s\n", attempt, string(body))
			if attempt < maxRetries {
				// Wait longer between retries on rate limit
				time.Sleep(time.Duration(attempt*3) * time.Second)
				continue
			}
			return fmt.Errorf("discord rate limit exceeded after %d attempts", maxRetries)
		default:
			fmt.Printf("❌ Attempt %d: Discord returned unexpected status %d. Response: %s\n",
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
func SendWebHook(username, discordID, nextUser string) error {
	gameName := getGameName()

	// Create webhook payload
	payload := DiscordWebhook{
		Username:  "Shadow Empire Assistant",
		AvatarURL: "https://raw.githubusercontent.com/auricom/home-ops/main/docs/src/assets/logo.png",
		Content:   fmt.Sprintf("🎲 It's your turn, <@%s>!", discordID),
		Embeds: []Embed{
			{
				Color: 0xFFA500,
				Thumbnail: Thumbnail{
					URL: "https://upload.wikimedia.org/wikipedia/en/4/4f/Shadow_Empire_cover.jpg",
				},
				Fields: []Field{
					{
						Name:  "📋 Save File Instructions",
						Value: fmt.Sprintf("After completing your turn, please save the file as:\n```\n%s_turn#_%s\n```\n*(Replace # with the current turn number)*", gameName, nextUser),
					},
				},
				Footer: Footer{
					Text: "Made with ❤️ by Solon",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return sendDiscordWebhook(&payload, username, discordID, false)
}

// SendRenameWebHook sends a Discord webhook notification asking to rename a file
func SendRenameWebHook(username, discordID, filename string) error {
	gameName := getGameName()

	// Create webhook payload
	payload := DiscordWebhook{
		Username:  "Shadow Empire Assistant",
		AvatarURL: "https://raw.githubusercontent.com/auricom/home-ops/main/docs/src/assets/logo.png",
		Content:   fmt.Sprintf("⚠️ File naming issue detected in your save, <@%s>!", discordID),
		Embeds: []Embed{
			{
				Color: 0xFF0000, // Red color for warning
				Thumbnail: Thumbnail{
					URL: "https://upload.wikimedia.org/wikipedia/en/4/4f/Shadow_Empire_cover.jpg",
				},
				Fields: []Field{
					{
						Name: "📋 File Rename Required",
						Value: fmt.Sprintf("The save file you created `%s` doesn't match the configured game name.\n\nPlease rename it to follow the format:\n```\n%s_turn#_%s\n```\n*(Replace # with the current turn number and %s with the next player's name)*",
							filename, gameName, "[NextPlayerName]", "[NextPlayerName]"),
					},
				},
				Footer: Footer{
					Text: "Made with ❤️ by Solon",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return sendDiscordWebhook(&payload, username, discordID, true)
}
