package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

// NewAPIClient constructs a new Holodex API client.
func NewAPIClient(apiKey string) *APIClient {
	return &APIClient{
		BaseURL: "https://holodex.net/api/v2/live",
		xApiKey:  xApiKey,
		Client:  &http.Client{},
	}
}

func (c *APIClient) FetchVideos(org string, topic string) ([]APIVideoInfo, error) {
	types := []string{"stream", "placeholder"} // You can add more types here
	var allVideos []APIVideoInfo

	for _, videoType := range types {
		// Build query parameters
		params := url.Values{}
		params.Set("org", org)
		params.Set("topic", topic)
		params.Set("status", "new,upcoming,live")
		params.Set("type", videoType)
		params.Set("limit", "50")

		// Final URL
		fullURL := fmt.Sprintf("%s?%s", c.BaseURL, params.Encode())

		// Build request
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-APIKEY", c.xApiKey)

		// Make request
		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// Decode JSON
		var videos []APIVideoInfo
		if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
			return nil, err
		}

		allVideos = append(allVideos, videos...)
	}

	// Optional: print fetched videos
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("FetchVideos: Fetched %d videos", len(allVideos))
		for _, v := range allVideos {
			fmt.Printf("%s🎵 %s (%s) by %s\n", v.TopicID, v.Title, v.Status, v.Channel.Name)
			fmt.Printf("   YouTube: https://www.youtube.com/watch?v=%s\n", v.ID)
			fmt.Println("--------------------------------------------------")
		}
	}

	return allVideos, nil
}
