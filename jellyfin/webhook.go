// Package jellyfin encapsulates the structures and logic for parsing
// and handling incoming webhook notifications sent by Jellyfin.
package jellyfin


import "strings"

// WebhookPayload represents the incoming payload from Jellyfin's webhook
type WebhookPayload struct {
	ShokoEpisodeIDStr     string `json:"Provider_shoko episode"` // Maps e.g. "3011"
	EpisodeNumber         int    `json:"EpisodeNumber"`
	SeasonNumber          int    `json:"SeasonNumber"`
	NotificationType      string `json:"NotificationType"`
	SaveReason            string `json:"SaveReason"`
	PlaybackPositionTicks int64  `json:"PlaybackPositionTicks"`
	RunTimeTicks          int64  `json:"RunTimeTicks"`
	NotificationUsername  string `json:"NotificationUsername"`
}

// WatchStatus values for Simkl/internal
const (
	StatusWatching  = "watching"
	StatusCompleted = "completed"
	StatusHold      = "hold"
)

// HasShokoAnime checks if the payload represents a Shoko anime episode
func (payload *WebhookPayload) HasShokoAnime() bool {
	return payload.ShokoEpisodeIDStr != "" && !strings.Contains(payload.ShokoEpisodeIDStr, "Provider_")
}

// DetermineWatchStatus calculates completion status from ticks
func (payload *WebhookPayload) DetermineWatchStatus() string {
	if payload.NotificationType == "PlaybackStart" {
		return StatusWatching
	}

	if payload.RunTimeTicks == 0 {
		return StatusWatching
	}

	percentage := (float64(payload.PlaybackPositionTicks) / float64(payload.RunTimeTicks)) * 100.0
	if percentage >= 80.0 {
		return StatusCompleted
	}

	return StatusHold
}
