package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// FocusMode holds the ticker and a channel to signal stop.
type FocusMode struct {
	ticker   *time.Ticker
	stopChan chan struct{}
}

// focusModes is a registry of active focus modes.
// It is protected by a mutex for concurrent access.
var (
	focusModes   = make(map[string]*FocusMode)
	focusModesMu sync.Mutex
)

// startFocusMode starts calls checkHolodex for the given link every 4 minutes.
// If focus mode is already running for the link, it does nothing.
func startFocusMode(link string) {
	focusModesMu.Lock()
	defer focusModesMu.Unlock()

	// Check if already running.
	if _, exists := focusModes[link]; exists {
		fmt.Printf("Focus mode already running for %s\n", link)
		return
	}

	// Create a new FocusMode instance.
	fm := &FocusMode{
		ticker:   time.NewTicker(4 * time.Minute),
		stopChan: make(chan struct{}),
	}
	focusModes[link] = fm

	// Launch a goroutine that scrape holodex every 4 minutes.
	go func() {
		defer func() {
			focusModesMu.Lock()
			delete(focusModes, link)
			focusModesMu.Unlock()
		}()
	
		// Immediately trigger the first print.
		logrus.Info("Scraping:", link)

		if err := focusScrape(link); err != nil {
			logrus.Errorf("Error in focus mode: %v", err)
			return
		}

		for {
			select {
			case <-fm.ticker.C:
				if err := focusScrape(link); err != nil {
					logrus.Errorf("Error in focus mode: %v", err)
					return
				}
			case <-fm.stopChan:
				fm.ticker.Stop()
				return
			}
		}
	}()
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

// stopFocusMode stops the focus mode for the given link.
func stopFocusMode(link string) {
	focusModesMu.Lock()
	defer focusModesMu.Unlock()

	if fm, exists := focusModes[link]; exists {
		// Signal the goroutine to stop and remove it from the registry.
		close(fm.stopChan)
		delete(focusModes, link)
		fmt.Printf("Focus mode stopped for %s\n", link)
	} else {
		fmt.Printf("No focus mode running for %s\n", link)
	}
}

func stopAllFocusModes() {
    focusModesMu.Lock()
    defer focusModesMu.Unlock()
    for link, fm := range focusModes {
        close(fm.stopChan)
        delete(focusModes, link)
        fmt.Printf("Focus mode stopped for %s\n", link)
    }
}

// scheduleFocusMode schedules the start of focus mode for each event.
// When the scheduled time is reached, it calls startFocusMode for the link.
func scheduleFocusMode(events map[string]time.Time) {
	for link, eventTime := range events {
		delay := time.Until(eventTime)
		if delay < 0 {
			// Skip events that are already in the past.
			logrus.Warnf("Skipping event for %s because event time %s is in the past.", link, eventTime.Format(time.RFC3339))

			continue
		}

		// Log that focus mode is scheduled.
		logrus.Infof("Scheduling focus mode for %s at %s (in %s)", link, eventTime.Format(time.RFC3339), delay)

		// Schedule startFocusMode to be called at the event time.
		go func(link string, delay time.Duration) {
			time.AfterFunc(delay, func() {
				startFocusMode(link)
			})
		}(link, delay)
	}
}
