package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sirupsen/logrus"
)

type VideoInfo struct {
	Topic          string
	Channel        string
	LiveStatus     string
	UpcomingStatus string
	Duration       string
	YoutubeLink    string
}

type HolodexScraper struct {
	videoInfos []VideoInfo
}

func (h *HolodexScraper) checkHolodex(holodexUrl string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancelCtx := chromedp.NewContext(allocatorCtx)
	defer cancelCtx()

	err := chromedp.Run(ctx,
		chromedp.Navigate(holodexUrl),
		chromedp.WaitVisible(`a.video-card.no-decoration.d-flex.video-card-fluid.flex-column`, chromedp.ByQuery),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a.video-card.no-decoration.d-flex.video-card-fluid.flex-column')).map(card => {
			const topic = card.querySelector('div.video-topic.rounded-tl-sm')?.innerText.trim() || '';
			const channel = card.querySelector('div.channel-name.video-card-subtitle')?.innerText.trim() || '';
			const liveStatus = card.querySelector('div.video-card-subtitle span.text-live')?.innerText.trim() || '';
			const upcomingStatus = card.querySelector('div.video-card-subtitle span.text-upcoming')?.innerText.trim() || '';
			const duration = card.querySelector('div.video-duration.rounded-br-sm.video-duration-live')?.innerText.trim() || '';
			const youtubeLink = card.getAttribute('href') || '';
			return { topic, channel, liveStatus, upcomingStatus, duration, youtubeLink };
		});`, &h.videoInfos),
	)
	if err != nil {
		if strings.Contains(err.Error(), "no space left on device") {
			if err := makeDiskFullMessage(botToken, chatID, phoneNumber, apiKey); err != nil {
				return err
			}
			logrus.Info("Disk is full. Sleeping for 6 hours to allow cleanup.")
			time.Sleep(6 * time.Hour)
			return nil
		}
		return fmt.Errorf("failed to fetch data from Holodex: %w", err)
	}

	return nil
}

// Simulate timeNow function
func timeNow() time.Time {
	return time.Now()
}

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

