package ntfy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	"shoko-anime-sync/ntfy"
)

type TestConfig struct {
	NtfyURL   string `yaml:"ntfy_url"`
	NtfyToken string `yaml:"ntfy_token"`
	NtfyTopic string `yaml:"ntfy_topic"`
}

func TestSendMockNotification(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", request.Method)
		}
		if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
		if authHeader := request.Header.Get("Authorization"); authHeader != "Bearer mock-token" {
			t.Errorf("Expected Authorization Bearer mock-token, got %s", authHeader)
		}

		var payload ntfy.Payload
		if decodeErr := json.NewDecoder(request.Body).Decode(&payload); decodeErr != nil {
			t.Errorf("Failed to decode JSON request: %v", decodeErr)
		}

		if payload.Topic != "anime-sync" {
			t.Errorf("Expected topic 'anime-sync', got %s", payload.Topic)
		}
		if payload.Title != "Mock Title" {
			t.Errorf("Expected title 'Mock Title', got %s", payload.Title)
		}
		if payload.Message != "Mock Message" {
			t.Errorf("Expected message 'Mock Message', got %s", payload.Message)
		}
		if payload.Priority != 4 {
			t.Errorf("Expected priority 4, got %d", payload.Priority)
		}

		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ntfyClient := ntfy.NewClient(mockServer.URL, "mock-token")
	sendNotificationErr := ntfyClient.SendNotification(ntfy.Payload{
		Topic:    "anime-sync",
		Title:    "Mock Title",
		Message:  "Mock Message",
		Priority: 4,
	})

	if sendNotificationErr != nil {
		t.Fatalf("SendNotification failed: %v", sendNotificationErr)
	}
}

func TestSendRealNotification(t *testing.T) {
	// Skip real notification unless run explicitly or config is present
	configPath := "../config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Skipping real ntfy test because config.yaml is not found at root")
	}

	configFileBytes, fileReadErr := os.ReadFile(configPath)
	if fileReadErr != nil {
		t.Fatalf("Failed to read config file: %v", fileReadErr)
	}

	var testConfig TestConfig
	if unmarshalErr := yaml.Unmarshal(configFileBytes, &testConfig); unmarshalErr != nil {
		t.Fatalf("Failed to parse config file: %v", unmarshalErr)
	}

	if testConfig.NtfyURL == "" {
		t.Skip("Skipping real ntfy test because ntfy_url is empty in config.yaml")
	}

	t.Logf("Dispatching real test notification to %s", testConfig.NtfyURL)

	ntfyClient := ntfy.NewClient(testConfig.NtfyURL, testConfig.NtfyToken)
	topic := testConfig.NtfyTopic
	if topic == "" {
		topic = "anime-sync"
	}

	sendNotificationErr := ntfyClient.SendNotification(ntfy.Payload{
		Topic:    topic,
		Title:    "Failed to update Simkl with anime (Test)",
		Message:  "AniDB ID: 6773, Season: 1, Episode: 1 (Linter and Connection Test)\nLink: https://anidb.net/anime/6773",
		Priority: 4,
	})

	if sendNotificationErr != nil {
		t.Fatalf("Failed to dispatch test notification: %v", sendNotificationErr)
	}

	t.Log("Successfully dispatched notification to ntfy server!")
}
