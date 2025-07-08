package internal

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// changed for API version
func getStartTime(videoInfos []APIVideoInfo) (map[string]time.Time, error) {
	results := make(map[string]time.Time)
	now := timeNow()
	var errors []string

	for _, video := range videoInfos {
		if video.StartScheduled == "" {
			continue
		}

		startTimeUTC, err := time.Parse(time.RFC3339, video.StartScheduled)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid RFC3339 time for %s: %s", video.ID, video.StartScheduled))
			continue
		}

		startTime := startTimeUTC.In(now.Location())

		if startTime.Before(now) {
			errors = append(errors, fmt.Sprintf("Skipping past date for %s: %s", video.ID, video.StartScheduled))
			continue
		}

		results[video.ID] = startTime
	}

	// Conditionally show debug info
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("getStartTime: %d valid start times", len(results))
		for id, start := range results {
			logrus.Debugf("  - %s → %s", id, start.Format(time.RFC3339))
		}
		if len(errors) > 0 {
			logrus.Debugf("getStartTime: %d parsing/skipped errors", len(errors))
			for _, e := range errors {
				logrus.Debugf("  ✗ %s", e)
			}
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("encountered errors: %v", errors)
	}

	return results, nil
}



func karaokeHandler(videoInfos []APIVideoInfo) ([]APIVideoInfo, error) {
	var singingInfos []APIVideoInfo

	for _, info := range videoInfos {
		
			singingInfos = append(singingInfos, info)
			if err := makeFoundMessage(info, botToken, chatID, phoneNumber, apiKey); err != nil {
				return nil, err
			}
		
	}

	if len(singingInfos) == 0 {
		if err := makeNotFoundMessage(botToken, chatID, phoneNumber, apiKey); err != nil {
			return nil, err
		}
	}

	return singingInfos, nil
}
