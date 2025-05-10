package pkg

import (
	"fmt"
	"net/http"
	"net/url"
)

// SendTelegramMessage sends a message to all chat IDs from TELEGRAM_CHAT_IDS env var
// Returns a map of chat IDs to errors (nil means success)
func SendTelegramMessage(botToken string, chatIDs []string, message string) error {
	var result error

	if botToken == "" {
		return nil
	}

	// Send to each chat ID
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	for _, chatID := range chatIDs {
		// Send the message
		resp, err := http.PostForm(apiURL, url.Values{
			"chat_id": {chatID},
			"text":    {message},
		})

		// Handle errors
		if err != nil {
			result = err
			continue
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			result = fmt.Errorf("API returned status: %s", resp.Status)
			resp.Body.Close()
			continue
		}

		resp.Body.Close()
	}

	return result
}
