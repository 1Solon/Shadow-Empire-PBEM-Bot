package userparser

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// UserMapping holds the order, username, and Discord ID for a user.
type UserMapping struct {
	Order     int
	Username  string
	DiscordID string
}

// ParseUsers parses username to Discord ID mappings from a comma-separated environment variable
// Format: "1 Username1 DiscordId1,2 Username2 DiscordId2"
// Returns a slice of UserMapping sorted by the order number.
func ParseUsers(envVarName string) ([]UserMapping, error) {
	envVar := os.Getenv(envVarName)
	if envVar == "" {
		return nil, fmt.Errorf("environment variable %s not found", envVarName)
	}
	return parseUsersFromString(envVar)
}

// ParseUsersFromString parses mappings from a raw string value instead of reading the environment.
// Format: "1 Username1 DiscordId1,2 Username2 DiscordId2"
func ParseUsersFromString(raw string) ([]UserMapping, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty user mappings string")
	}
	return parseUsersFromString(raw)
}

func parseUsersFromString(input string) ([]UserMapping, error) {
	var userMappings []UserMapping
	pairs := strings.Split(input, ",")
	for i, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), " ", 3) // Split into 3 parts: order, username, discordId
		if len(parts) == 3 {
			orderStr := parts[0]
			username := parts[1]
			discordId := parts[2]

			order, err := strconv.Atoi(orderStr)
			if err != nil {
				return nil, fmt.Errorf("invalid order number '%s' in mapping part %d: %w", orderStr, i+1, err)
			}

			if username != "" && discordId != "" {
				userMappings = append(userMappings, UserMapping{
					Order:     order,
					Username:  username,
					DiscordID: discordId,
				})
			} else {
				return nil, fmt.Errorf("invalid format in mapping part %d: username or discordId is empty", i+1)
			}
		} else {
			return nil, fmt.Errorf("invalid format in mapping part %d: expected 'order username discordId', got '%s'", i+1, strings.TrimSpace(pair))
		}
	}

	// Sort the mappings by the Order field
	sort.Slice(userMappings, func(i, j int) bool {
		return userMappings[i].Order < userMappings[j].Order
	})

	// Check for duplicate order numbers
	orders := make(map[int]bool)
	for _, mapping := range userMappings {
		if orders[mapping.Order] {
			return nil, fmt.Errorf("duplicate order number %d found in mappings", mapping.Order)
		}
		orders[mapping.Order] = true
	}

	// Check for sequential order numbers starting from 1 (optional but good practice)
	for i, mapping := range userMappings {
		if mapping.Order != i+1 {
			fmt.Printf("⚠️ Warning: User mapping order numbers are not sequential starting from 1 (found order %d at position %d)\n", mapping.Order, i+1)
		}
	}

	if len(userMappings) == 0 {
		return nil, fmt.Errorf("no valid user mappings found")
	}

	return userMappings, nil
}
