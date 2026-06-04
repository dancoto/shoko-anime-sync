// Package shoko provides a client API to communicate with a local Shoko server
// instance to retrieve AniDB metadata for episodes.
package shoko


import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type EpisodeResponse struct {
	IDs struct {
		TMDB struct {
			Movie []int `json:"Movie"`
		} `json:"TMDB"`
	} `json:"IDs"`
	AniDB struct {
		AnimeID       int    `json:"AnimeID"`
		Type          string `json:"Type"`
		EpisodeNumber int    `json:"EpisodeNumber"`
	} `json:"AniDB"`
}

// GetEpisodeMetadata fetches episode metadata from Shoko
func (client *Client) GetEpisodeMetadata(episodeID string) (*EpisodeResponse, error) {
	shokoURL := fmt.Sprintf("%s/api/v3/Episode/%s?includeDataFrom=AniDB", client.BaseURL, episodeID)
	httpRequest, creationErr := http.NewRequest("GET", shokoURL, nil)
	if creationErr != nil {
		return nil, fmt.Errorf("failed to create request: %w", creationErr)
	}
	httpRequest.Header.Set("apiKey", client.Token)

	httpResponse, executionErr := client.HTTPClient.Do(httpRequest)
	if executionErr != nil {
		return nil, executionErr
	}
	defer func() { _ = httpResponse.Body.Close() }()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shoko API returned status %d", httpResponse.StatusCode)
	}

	var episodeMetadata EpisodeResponse
	if decodeErr := json.NewDecoder(httpResponse.Body).Decode(&episodeMetadata); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode response: %w", decodeErr)
	}

	return &episodeMetadata, nil
}
