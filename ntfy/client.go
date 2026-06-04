// Package ntfy provides a client implementation to send notifications
// to a ntfy instance.
package ntfy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Payload struct {
	Topic    string `json:"topic"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

// SendNotification dispatches a notification payload to the ntfy server
func (client *Client) SendNotification(payload Payload) error {
	jsonData, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal ntfy payload: %w", marshalErr)
	}

	httpRequest, creationErr := http.NewRequest("POST", client.BaseURL, bytes.NewBuffer(jsonData))
	if creationErr != nil {
		return fmt.Errorf("failed to create request: %w", creationErr)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if client.Token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+client.Token)
	}

	httpResponse, executionErr := client.HTTPClient.Do(httpRequest)
	if executionErr != nil {
		return fmt.Errorf("failed to send request: %w", executionErr)
	}
	defer func() { _ = httpResponse.Body.Close() }()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy API returned status %d", httpResponse.StatusCode)
	}

	return nil
}
