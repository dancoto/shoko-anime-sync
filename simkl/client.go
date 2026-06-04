// Package simkl contains the client implementation and payloads needed to
// scrobble and sync watch history to Simkl.
package simkl


import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	ClientID   string
	AppName    string
	HTTPClient *http.Client
}

func NewClient(clientID, appName string) *Client {
	return &Client{
		ClientID: clientID,
		AppName:  appName,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type TVPayload struct {
	Shows []Show `json:"shows"`
}

type Show struct {
	IDs     map[string]string `json:"ids"`
	Seasons []Season          `json:"seasons"`
}

type Season struct {
	Number   int       `json:"number"`
	Episodes []Episode `json:"episodes"`
}

type Episode struct {
	Number int `json:"number"`
}

type MoviePayload struct {
	Movies []Movie `json:"movies"`
}

type Movie struct {
	IDs    map[string]string `json:"ids"`
	Status string            `json:"status"`
}

type StringOrInt int

func (soi *StringOrInt) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch val := v.(type) {
	case float64:
		*soi = StringOrInt(val)
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		*soi = StringOrInt(i)
	}
	return nil
}

type SyncResponse struct {
	Added struct {
		Movies   int `json:"movies"`
		Shows    int `json:"shows"`
		Episodes int `json:"episodes"`
		Statuses []struct {
			Request struct {
				IDs struct {
					AniDB StringOrInt `json:"anidb"`
				} `json:"ids"`
				Type string `json:"type"`
			} `json:"request"`
		} `json:"statuses"`
	} `json:"added"`
	NotFound struct {
		Movies   []any `json:"movies"`
		Shows    []any `json:"shows"`
		Episodes []any `json:"episodes"`
	} `json:"not_found"`
}

// ScrobbleAnime updates a show history
func (client *Client) ScrobbleAnime(anidbID int, episodeNumber int, seasonNumber int, token string) (int, error) {
	tvPayload := TVPayload{
		Shows: []Show{
			{
				IDs: map[string]string{"anidb": strconv.Itoa(anidbID)},
				Seasons: []Season{
					{
						Number: seasonNumber,
						Episodes: []Episode{
							{Number: episodeNumber},
						},
					},
				},
			},
		},
	}
	matchedID, scrobbleErr := client.sendScrobble(tvPayload, token, false)
	if scrobbleErr == nil && matchedID == 0 {
		matchedID = anidbID
	}
	return matchedID, scrobbleErr
}

// ScrobbleAnimeMovie updates a movie history
func (client *Client) ScrobbleAnimeMovie(anidbID int, watchStatus string, token string) (int, error) {
	moviePayload := MoviePayload{
		Movies: []Movie{
			{
				IDs:    map[string]string{"anidb": strconv.Itoa(anidbID)},
				Status: watchStatus,
			},
		},
	}
	matchedID, scrobbleErr := client.sendScrobble(moviePayload, token, true)
	if scrobbleErr == nil && matchedID == 0 {
		matchedID = anidbID
	}
	return matchedID, scrobbleErr
}

func (client *Client) sendScrobble(payload any, token string, isMovie bool) (int, error) {
	jsonData, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", marshalErr)
	}

	simklURL := fmt.Sprintf("https://api.simkl.com/sync/history?client_id=%s&app-name=%s&app-version=1.0", client.ClientID, client.AppName)
	httpRequest, creationErr := http.NewRequest("POST", simklURL, bytes.NewBuffer(jsonData))
	if creationErr != nil {
		return 0, fmt.Errorf("failed to create request: %w", creationErr)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", fmt.Sprintf("%s/1.0", client.AppName))
	httpRequest.Header.Set("Authorization", "Bearer "+token)

	httpResponse, executionErr := client.HTTPClient.Do(httpRequest)
	if executionErr != nil {
		return 0, fmt.Errorf("failed to send request: %w", executionErr)
	}
	defer func() { _ = httpResponse.Body.Close() }()

	if httpResponse.StatusCode != http.StatusCreated && httpResponse.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("simkl API rejected update with status %d", httpResponse.StatusCode)
	}

	bodyBytes, readErr := io.ReadAll(httpResponse.Body)
	if readErr != nil {
		return 0, fmt.Errorf("failed to read response body: %w", readErr)
	}

	var syncResp SyncResponse
	if decodeErr := json.Unmarshal(bodyBytes, &syncResp); decodeErr != nil {
		return 0, fmt.Errorf("failed to decode response: %w", decodeErr)
	}

	if isMovie {
		if len(syncResp.NotFound.Movies) > 0 {
			log.Printf("[Warning] Simkl failed to match movie. Raw Response: %s", string(bodyBytes))
			return 0, fmt.Errorf("simkl failed to match movie (movie in not_found list)")
		}
	} else {
		if len(syncResp.NotFound.Shows) > 0 || len(syncResp.NotFound.Episodes) > 0 {
			log.Printf("[Warning] Simkl failed to match episode. Raw Response: %s", string(bodyBytes))
			return 0, fmt.Errorf("simkl failed to match episode (show/episode in not_found list)")
		}
	}

	// Retrieve AniDB ID
	var matchedID int
	if len(syncResp.Added.Statuses) > 0 {
		matchedID = int(syncResp.Added.Statuses[0].Request.IDs.AniDB)
	}

	return matchedID, nil
}
