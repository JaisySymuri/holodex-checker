package internal

import "net/http"

var (
	botToken    string
	chatID      string
	phoneNumber string
	apiKey      string
	Running     bool = true
	xApiKey     string
)

// type ScrapperVideoInfo struct {
// 	Topic          string
// 	Channel        string
// 	LiveStatus     string
// 	UpcomingStatus string
// 	Duration       string
// 	YoutubeLink    string
// }

type HolodexScraper struct {
	videoInfos []APIVideoInfo
}

type Channel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Org         string `json:"org"`
	Suborg      string `json:"suborg"`
	Type        string `json:"type"`
	Photo       string `json:"photo"`
	EnglishName string `json:"english_name"`
}

type APIVideoInfo struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Type           string  `json:"type"`     // "stream" or "clip"
	TopicID        string  `json:"topic_id"` // e.g. "singing"
	PublishedAt    string  `json:"published_at"`
	AvailableAt    string  `json:"available_at"`
	Duration       int     `json:"duration"` // seconds
	Status         string  `json:"status"`   // "upcoming", "live", etc.
	StartScheduled string  `json:"start_scheduled"`
	LiveViewers    int     `json:"live_viewers"`
	Channel        Channel `json:"channel"`
}

// APIClient handles communication with the Holodex API.
type APIClient struct {
	BaseURL string
	xApiKey  string
	Client  *http.Client
}
