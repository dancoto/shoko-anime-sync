// Package main provides the entry point, configuration loader, and HTTP
// webhook listener for coordinating anime synchronization between Jellyfin,
// Shoko, and Simkl.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"shoko-anime-sync/jellyfin"
	"shoko-anime-sync/ntfy"
	"shoko-anime-sync/shoko"
	"shoko-anime-sync/simkl"
)

var (
	config      Config
	shokoClient *shoko.Client
	simklClient *simkl.Client
	ntfyClient  *ntfy.Client
)

func loadConfig() {
	configPath := "config.yaml"
	if _, err := os.Stat("/config/config.yaml"); err == nil {
		configPath = "/config/config.yaml"
	}
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	configFileBytes, fileReadErr := os.ReadFile(configPath)
	if fileReadErr != nil {
		log.Fatalf("FATAL: Failed to read config file at %s: %v", configPath, fileReadErr)
	}

	if unmarshalErr := yaml.Unmarshal(configFileBytes, &config); unmarshalErr != nil {
		log.Fatalf("FATAL: Failed to parse config file: %v", unmarshalErr)
	}

	// Validation
	if config.SimklClientID == "" {
		log.Fatal("FATAL: 'simkl_client_id' is required in config.yaml")
	}
	if config.AppName == "" {
		log.Fatal("FATAL: 'app_name' is required in config.yaml")
	}
	if config.ShokoToken == "" {
		log.Fatal("FATAL: 'shoko_token' is required in config.yaml")
	}
	if config.ShokoURL == "" {
		log.Fatal("FATAL: 'shoko_url' is required in config.yaml")
	}
	if config.NtfyURL != "" && config.NtfyTopic == "" {
		log.Fatal("FATAL: 'ntfy_topic' is required when 'ntfy_url' is configured in config.yaml")
	}
	if len(config.Users) == 0 {
		log.Fatal("FATAL: 'users' mapping is empty or missing in config.yaml")
	}

	// Convert config.Users keys to lowercase for case-insensitive lookup
	lowerUsers := make(map[string]string)
	for username, userToken := range config.Users {
		lowerUsers[strings.ToLower(username)] = userToken
	}
	config.Users = lowerUsers
}

func main() {
	loadConfig()

	// Initialize clients
	shokoClient = shoko.NewClient(config.ShokoURL, config.ShokoToken)
	simklClient = simkl.NewClient(config.SimklClientID, config.AppName)

	if config.NtfyURL != "" {
		ntfyClient = ntfy.NewClient(config.NtfyURL, config.NtfyToken)
	}

	http.HandleFunc("/webhook", handleUnifiedWebhook)
	log.Printf("Unified Media Bridge listening on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handleUnifiedWebhook(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var webhookPayload jellyfin.WebhookPayload
	if decodeErr := json.NewDecoder(request.Body).Decode(&webhookPayload); decodeErr != nil {
		http.Error(responseWriter, "Bad JSON", http.StatusBadRequest)
		return
	}

	if webhookPayload.HasShokoAnime() {
		handleAnimeSync(webhookPayload)
	}

	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write([]byte(`{"status":"processed"}`))
}

func handleAnimeSync(webhookPayload jellyfin.WebhookPayload) {
	log.Printf("[Shoko] Querying local Shoko Server for Episode ID: %s", webhookPayload.ShokoEpisodeIDStr)

	shokoData, fetchErr := shokoClient.GetEpisodeMetadata(webhookPayload.ShokoEpisodeIDStr)
	if fetchErr != nil {
		log.Printf("[Error] Local Shoko Server lookup failed: %v", fetchErr)
		return
	}

	anidbID := shokoData.AniDB.AnimeID
	jellyfinUsername := webhookPayload.NotificationUsername
	isMovie := len(shokoData.IDs.TMDB.Movie) == 1
	watchStatus := webhookPayload.DetermineWatchStatus()

	simklToken, exists := config.Users[strings.ToLower(jellyfinUsername)]
	if !exists {
		log.Printf("[Error] No Simkl token configured for user: %s", jellyfinUsername)
		return
	}

	if isMovie {
		// This is an anime movie, build/send the movie payload
		go func() {
			matchedID, scrobbleErr := simklClient.ScrobbleAnimeMovie(anidbID, watchStatus, simklToken)
			if scrobbleErr != nil {
				log.Printf("[Error] Failed to scrobble anime movie to Simkl: %v", scrobbleErr)
				if ntfyClient != nil {
					notificationErr := ntfyClient.SendNotification(ntfy.Payload{
						Topic:    config.NtfyTopic,
						Title:    "Failed to update Simkl with anime",
						Message:  fmt.Sprintf("Failed to sync movie. AniDB ID: %d\nLink: https://anidb.net/anime/%d", anidbID, anidbID),
						Priority: 4,
					})
					if notificationErr != nil {
						log.Printf("[Error] Failed to send ntfy notification: %v", notificationErr)
					}
				}
				return
			}
			log.Printf("[Simkl Success] %d set to %s for %s.", matchedID, watchStatus, jellyfinUsername)
		}()
	} else {
		// This is a series, follow main anime template
		epNumber := shokoData.AniDB.EpisodeNumber
		seasonNumber := 1
		if shokoData.AniDB.Type == "Special" {
			seasonNumber = 0
		}
		if watchStatus == jellyfin.StatusCompleted {
			go func() {
				matchedID, scrobbleErr := simklClient.ScrobbleAnime(anidbID, epNumber, seasonNumber, simklToken)
				if scrobbleErr != nil {
					log.Printf("[Error] Failed to scrobble anime to Simkl: %v", scrobbleErr)
					if ntfyClient != nil {
						notificationErr := ntfyClient.SendNotification(ntfy.Payload{
							Topic:    config.NtfyTopic,
							Title:    "Failed to update Simkl with anime",
							Message:  fmt.Sprintf("AniDB ID: %d, Season: %d, Episode: %d\nLink: https://anidb.net/anime/%d", anidbID, seasonNumber, epNumber, anidbID),
							Priority: 4,
						})
						if notificationErr != nil {
							log.Printf("[Error] Failed to send ntfy notification: %v", notificationErr)
						}
					}
					return
				}
				log.Printf("[Simkl Success] %d set to %s for %s.", matchedID, watchStatus, jellyfinUsername)
			}()
		}
	}
}
