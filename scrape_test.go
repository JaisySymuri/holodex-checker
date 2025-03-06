package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const mockHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Mock Holodex</title>
</head>
<body>
    <a class="video-card no-decoration d-flex video-card-fluid flex-column" href="https://www.youtube.com/watch?v=example1">
        <div class="video-topic rounded-tl-sm">Topic 1</div>
        <div class="channel-name video-card-subtitle">Channel 1</div>
        <div class="video-card-subtitle"><span class="text-live">Live Now</span></div>
        <div class="video-duration rounded-br-sm video-duration-live">1:23:45</div>
    </a>
    <a class="video-card no-decoration d-flex video-card-fluid flex-column" href="https://www.youtube.com/watch?v=example2">
        <div class="video-topic rounded-tl-sm">Topic 2</div>
        <div class="channel-name video-card-subtitle">Channel 2</div>
        <div class="video-card-subtitle"><span class="text-upcoming">Upcoming</span></div>
        <div class="video-duration rounded-br-sm video-duration-live">2:34:56</div>
    </a>
</body>
</html>
`

func TestCheckHolodex(t *testing.T) {
	// Create a mock server with the mock HTML content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// Create an instance of HolodexScraper
	hScraper := &HolodexScraper{}

	// Run the checkHolodex method with the mock server URL
	err := hScraper.checkHolodex(server.URL)
	assert.NoError(t, err)

	// Validate the scraped video information
	expected := []VideoInfo{
		{
			Topic:          "Topic 1",
			Channel:        "Channel 1",
			LiveStatus:     "Live Now",
			UpcomingStatus: "",
			Duration:       "1:23:45",
			YoutubeLink:    "https://www.youtube.com/watch?v=example1",
		},
		{
			Topic:          "Topic 2",
			Channel:        "Channel 2",
			LiveStatus:     "",
			UpcomingStatus: "Upcoming",
			Duration:       "2:34:56",
			YoutubeLink:    "https://www.youtube.com/watch?v=example2",
		},
	}

	assert.Equal(t, expected, hScraper.videoInfos)

	// Access and print the data to verify
	for _, videoInfo := range hScraper.videoInfos {
		t.Logf("VideoInfo: %+v\n", videoInfo)
	}
}

func TestIsolatedGetStartTime(t *testing.T) {
	videoInfos := []VideoInfo{
		{UpcomingStatus: "", YoutubeLink: "https://www.youtube.com/watch?v=example1"},
		{UpcomingStatus: "Starts in 4 hours (5:00 PM)", YoutubeLink: "https://www.youtube.com/watch?v=example2"},
		{UpcomingStatus: "Starts in 20 minutes (5:00 PM)", YoutubeLink: "https://www.youtube.com/watch?v=example3"},
		{UpcomingStatus: "Starts 3/7/2025 (5:00 PM)", YoutubeLink: "https://www.youtube.com/watch?v=example4"},
		{UpcomingStatus: "Starts 1/1/2025 (5:00 PM)", YoutubeLink: "https://www.youtube.com/watch?v=example5"},
	}

	startTimes, err := getStartTime(videoInfos)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Get the current date and time
	now := time.Now()

	expectedStartTimes := map[string]time.Time{
		"https://www.youtube.com/watch?v=example1": {},
		"https://www.youtube.com/watch?v=example2": time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, time.Local),
		"https://www.youtube.com/watch?v=example3": time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, time.Local),
		"https://www.youtube.com/watch?v=example4": time.Date(now.Year(), now.Month(), 7, 17, 0, 0, 0, time.Local),
		"https://www.youtube.com/watch?v=example5": {}, // Skipped due to past date
	}

	fmt.Println("Test Results:")
	fmt.Println("---------------------------------------------------------------")
	fmt.Printf("%-50s | %-30s | %-25s\n", "Youtube Link", "Upcoming Status", "Parsed Start Time")
	fmt.Println("---------------------------------------------------------------")

	for _, video := range videoInfos {
		startTime, exists := startTimes[video.YoutubeLink]
		if !exists {
			startTime = time.Time{} // Default zero value of time.Time
		}
		fmt.Printf("%-50s | %-30s | %-25v\n", video.YoutubeLink, video.UpcomingStatus, startTime)
	}

	for _, video := range videoInfos {
		expectedTime := expectedStartTimes[video.YoutubeLink]
		actualTime, exists := startTimes[video.YoutubeLink]
		if !exists {
			actualTime = time.Time{}
		}
		assert.Equal(t, expectedTime, actualTime, "Mismatch for video: "+video.YoutubeLink)
	}

	fmt.Println("Test passed successfully")
}

func TestScheduleFocusMode(t *testing.T) {
	now := timeNow()
	scheduledEvents := map[string]time.Time{
		"https://www.youtube.com/watch?v=example2": time.Date(now.Year(), now.Month(), now.Day(), 14, 06, 0, 0, time.Local),
		"https://www.youtube.com/watch?v=example3": time.Date(now.Year(), now.Month(), now.Day(), 14, 05, 30, 0, time.Local),
	}
	// Schedule the events.
	scheduleFocusMode(scheduledEvents)

	// For demonstration purposes, we'll manually stop the focus mode for one link after some time.
	go func() {
		// Wait enough time to see a few prints.
		time.Sleep(15 * time.Second)
		stopFocusMode("https://www.youtube.com/watch?v=example2")
	}()

	// Keep the program running so that scheduled tasks can execute.
	select {}
}
