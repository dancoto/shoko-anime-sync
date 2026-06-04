package main

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"shoko-anime-sync/jellyfin"
	"shoko-anime-sync/shoko"
	"shoko-anime-sync/simkl"
)

type RoundTripFunc func(httpRequest *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	return f(httpRequest)
}

func TestHandleAnimeSync(t *testing.T) {
	tests := []struct {
		name                 string
		shokoResponse        string
		webhookPayload       jellyfin.WebhookPayload
		expectedBodyContains string
		shouldScrobble       bool
	}{
		{
			name: "Movie Sync - Dispatches to Simkl as Movie status completed",
			shokoResponse: `{
				"IDs": {
					"TMDB": {
						"Movie": [12345]
					}
				},
				"AniDB": {
					"AnimeID": 6773,
					"Type": "Movie",
					"EpisodeNumber": 1
				}
			}`,
			webhookPayload: jellyfin.WebhookPayload{
				ShokoEpisodeIDStr:     "3011",
				NotificationUsername:  "userA",
				NotificationType:      "PlaybackStop",
				PlaybackPositionTicks: 90,
				RunTimeTicks:          100,
			},
			expectedBodyContains: `"movies":[{"ids":{"anidb":"6773"},"status":"completed"}]`,
			shouldScrobble:       true,
		},
		{
			name: "Special Sync - Season 0",
			shokoResponse: `{
				"IDs": {
					"TMDB": {
						"Movie": []
					}
				},
				"AniDB": {
					"AnimeID": 8899,
					"Type": "Special",
					"EpisodeNumber": 2
				}
			}`,
			webhookPayload: jellyfin.WebhookPayload{
				ShokoEpisodeIDStr:     "4022",
				NotificationUsername:  "userA",
				NotificationType:      "PlaybackStop",
				PlaybackPositionTicks: 85,
				RunTimeTicks:          100,
			},
			expectedBodyContains: `"shows":[{"ids":{"anidb":"8899"},"seasons":[{"number":0,"episodes":[{"number":2}]}]}]`,
			shouldScrobble:       true,
		},
		{
			name: "Regular Series Sync - Season 1",
			shokoResponse: `{
				"IDs": {
					"TMDB": {
						"Movie": []
					}
				},
				"AniDB": {
					"AnimeID": 9911,
					"Type": "TV",
					"EpisodeNumber": 5
				}
			}`,
			webhookPayload: jellyfin.WebhookPayload{
				ShokoEpisodeIDStr:     "5033",
				NotificationUsername:  "userA",
				NotificationType:      "PlaybackStop",
				PlaybackPositionTicks: 80,
				RunTimeTicks:          100,
			},
			expectedBodyContains: `"shows":[{"ids":{"anidb":"9911"},"seasons":[{"number":1,"episodes":[{"number":5}]}]}]`,
			shouldScrobble:       true,
		},
		{
			name: "Regular Series Sync - Incomplete status (less than 80%) does not scrobble",
			shokoResponse: `{
				"IDs": {
					"TMDB": {
						"Movie": []
					}
				},
				"AniDB": {
					"AnimeID": 9911,
					"Type": "TV",
					"EpisodeNumber": 5
				}
			}`,
			webhookPayload: jellyfin.WebhookPayload{
				ShokoEpisodeIDStr:     "5033",
				NotificationUsername:  "userA",
				NotificationType:      "PlaybackStop",
				PlaybackPositionTicks: 50,
				RunTimeTicks:          100,
			},
			expectedBodyContains: "",
			shouldScrobble:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize clients
			shokoClient = shoko.NewClient("http://mock-shoko", "test-shoko-token")
			simklClient = simkl.NewClient("test-client", "test-app")

			// Setup config
			config = Config{
				SimklClientID: "test-client",
				AppName:       "test-app",
				ShokoToken:    "test-shoko-token",
				ShokoURL:      "http://mock-shoko",
				Users: map[string]string{
					"usera": "test-userA-token",
				},
			}

			var simklRequestBody []byte
			var simklWG sync.WaitGroup
			if tt.shouldScrobble {
				simklWG.Add(1)
			}

			// Mock Transport for Shoko HTTP Client
			shokoClient.HTTPClient.Transport = RoundTripFunc(func(httpRequest *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(tt.shokoResponse)),
					Header:     make(http.Header),
				}, nil
			})

			// Mock Transport for Simkl HTTP Client
			simklClient.HTTPClient.Transport = RoundTripFunc(func(httpRequest *http.Request) (*http.Response, error) {
				defer simklWG.Done()
				var readErr error
				simklRequestBody, readErr = io.ReadAll(httpRequest.Body)
				if readErr != nil {
					t.Fatalf("failed to read mock simkl request body: %v", readErr)
				}

				var mockResponse string
				if strings.Contains(string(simklRequestBody), "movies") {
					mockResponse = `{
						"added": {
							"movies": 1,
							"shows": 0,
							"episodes": 0,
							"statuses": [
								{
									"request": {
										"ids": {
											"anidb": "6773"
										},
										"type": "movie"
									}
								}
							]
						},
						"not_found": {
							"movies": [],
							"shows": [],
							"episodes": []
						}
					}`
				} else {
					mockResponse = `{
						"added": {
							"movies": 0,
							"shows": 1,
							"episodes": 1,
							"statuses": [
								{
									"request": {
										"ids": {
											"anidb": "14116"
										},
										"type": "show"
									}
								}
							]
						},
						"not_found": {
							"movies": [],
							"shows": [],
							"episodes": []
						}
					}`
				}

				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(mockResponse)),
					Header:     make(http.Header),
				}, nil
			})

			// Invoke the logic under test
			handleAnimeSync(tt.webhookPayload)

			// Wait for the async scrobble goroutine if it was supposed to happen
			if tt.shouldScrobble {
				simklWG.Wait()
				bodyString := string(simklRequestBody)
				if !strings.Contains(bodyString, tt.expectedBodyContains) {
					t.Errorf("Expected request body to contain '%s', got '%s'", tt.expectedBodyContains, bodyString)
				}
			} else {
				// Wait a tiny moment to verify no requests occurred
				if len(simklRequestBody) > 0 {
					t.Errorf("Expected no scrobble request, but got: %s", string(simklRequestBody))
				}
			}
		})
	}
}
