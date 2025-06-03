package internal

import (
	"context"
	"fmt"
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

func focusScrape(link string) error {
	// Initialize a new HolodexScraper instance.
	hScraper := &HolodexScraper{}

	// Attempt to scrape holodex.net with retries.
	err := retry(30, 10*time.Second, func() error {
		return hScraper.checkHolodex("https://holodex.net/")
	})
	if err != nil {
		logrus.Error("checkHolodex failed after retries: ", err)
		return err
	}

	// Filter the videos based on the provided link.
	filteredVideos := []VideoInfo{}
	for _, video := range hScraper.videoInfos {
		logrus.Debugf("Checking video: %s", video.YoutubeLink)
		if video.YoutubeLink == link {
			filteredVideos = append(filteredVideos, video)
			break
		}
	}
	hScraper.videoInfos = filteredVideos

	// If no matching stream is found, log a message and exit.
	if len(filteredVideos) == 0 {
		// Check if there is at least one video to retrieve channel info.
		if len(hScraper.videoInfos) > 0 {
			logrus.Infof("Focus mode: No 'Singing' stream scheduled for %s - %s. The stream might've been canceled", hScraper.videoInfos[0].Channel, link)
		} else {
			logrus.Infof("Focus mode: No 'Singing' stream scheduled for link %s", link)
		}
		return nil
	}

	// Notify with the filtered video info.
	err = retry(30, 10*time.Second, func() error {
		return focusNotifyMe(hScraper.videoInfos)
	})
	if err != nil {
		logrus.Error("notifyMe failed after retries: ", err)
		return err
	}

	return nil
}
