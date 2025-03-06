package main

import "testing"

func TestFocusNotifyMe(t *testing.T) {
	videoInfos := []VideoInfo{
		{
			Channel:     "channel1",
			Topic:       "topic1",
			YoutubeLink: "youtubeLink1",
			Duration:    "duration",
		},
		// In production, there should be only 1 element in videoInfos
		{
			Channel:     "channel2",
			Topic:       "topic2",
			YoutubeLink: "youtubeLink2",
		},
	}

	focusNotifyMe(videoInfos)
}
