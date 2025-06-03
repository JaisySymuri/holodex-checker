package internal

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func getStartTime(videoInfos []VideoInfo) (map[string]time.Time, error) {
	results := make(map[string]time.Time)
	now := timeNow()
	loc := now.Location()
	var errors []string

	dateRegex := regexp.MustCompile(`Starts (\d{1,2}/\d{1,2}/\d{4})`)
	timeRegex := regexp.MustCompile(`\((\d{1,2}:\d{2} (AM|PM))\)`)

	for _, video := range videoInfos {
		if video.UpcomingStatus == "" {
			continue
		}

		// Skip if no time component is found.
		timeMatch := timeRegex.FindStringSubmatch(video.UpcomingStatus)
		if timeMatch == nil {
			continue
		}

		// Parse the time component.
		parsedTime, err := time.ParseInLocation("3:04 PM", timeMatch[1], loc)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid time for %s: %s", video.YoutubeLink, timeMatch[1]))
			continue
		}

		// Parse the date if available; if not, default to today's date.
		dateMatch := dateRegex.FindStringSubmatch(video.UpcomingStatus)
		var startDate time.Time
		if dateMatch != nil {
			parsedDate, err := time.ParseInLocation("1/2/2006", dateMatch[1], loc)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Invalid date for %s: %s", video.YoutubeLink, dateMatch[1]))
				continue
			}
			startDate = parsedDate
		} else {
			startDate = now
		}

		startTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)

		// Skip if the resulting start time is in the past.
		if startTime.Before(now) {
			errors = append(errors, fmt.Sprintf("Skipping past date for %s: %s", video.YoutubeLink, video.UpcomingStatus))
			continue
		}

		results[video.YoutubeLink] = startTime
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("errors encountered: \n%s", strings.Join(errors, "\n"))
	}
	return results, nil
}

func karaokeHandler(videoInfos []VideoInfo) ([]VideoInfo, error) {
	var singingInfos []VideoInfo

	for _, info := range videoInfos {
		if info.Topic == "Singing" {
			singingInfos = append(singingInfos, info)
			if err := makeFoundMessage(info, botToken, chatID, phoneNumber, apiKey); err != nil {
				return nil, err
			}
		}
	}

	if len(singingInfos) == 0 {
		if err := makeNotFoundMessage(botToken, chatID, phoneNumber, apiKey); err != nil {
			return nil, err
		}
	}

	return singingInfos, nil
}