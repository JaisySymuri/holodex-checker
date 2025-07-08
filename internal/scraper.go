package internal

import (
	"time"

	"github.com/sirupsen/logrus"
)

func focusScrape(videoID string) error {
	apiClient := NewAPIClient(xApiKey)
	var apiVideos []APIVideoInfo

	// Fetch from Holodex API with retry
	err := retry(30, 10*time.Second, func() error {
		var err error
		apiVideos, err = apiClient.FetchVideos("Hololive", "singing")
		return err
	})
	if err != nil {
		logrus.Error("FetchVideos failed after retries: ", err)
		return err
	}

	// Filter videos matching the given YouTube link
	var filteredVideos []APIVideoInfo
	for _, v := range apiVideos {
		VideoID := "https://www.youtube.com/watch?v=" + v.ID
		logrus.Debugf("Checking video: %s", VideoID)

		if VideoID == videoID {
			filteredVideos = append(filteredVideos, APIVideoInfo{
				TopicID:          v.TopicID,
				Channel:        v.Channel,
				Status:     v.Status,				
				ID:    VideoID,
			})
			break
		}
	}

	// No match found
	if len(filteredVideos) == 0 {
		if len(apiVideos) > 0 {
			logrus.Infof("Focus mode: No 'Singing' stream scheduled for %s - %s. The stream might've been canceled", apiVideos[0].Channel.Name, videoID)
		} else {
			logrus.Infof("Focus mode: No 'Singing' stream scheduled for link %s", videoID)
		}
		return nil
	}

	// Notify using the filtered list
	err = retry(30, 10*time.Second, func() error {
		return focusNotifyMe(filteredVideos)
	})
	if err != nil {
		logrus.Error("focusNotifyMe failed after retries: ", err)
		return err
	}

	return nil
}

