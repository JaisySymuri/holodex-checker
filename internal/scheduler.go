package internal

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

// StartFocusMode starts calls checkHolodex for the given link every 4 minutes.
// If focus mode is already running for the link, it does nothing.
func StartFocusMode(videoID string) {
	focusModesMu.Lock()
	defer focusModesMu.Unlock()

	// Check if already running.
	if _, exists := focusModes[videoID]; exists {
		fmt.Printf("Focus mode already running for %s\n", videoID)
		return
	}

	// Create a new FocusMode instance.
	fm := &FocusMode{
		ticker:   time.NewTicker(2 * time.Minute),
		stopChan: make(chan struct{}),
	}
	focusModes[videoID] = fm

	// Launch a goroutine that scrape holodex every 2 minutes.
	go func() {
		defer func() {
			focusModesMu.Lock()
			delete(focusModes, videoID)
			focusModesMu.Unlock()
		}()
	
		// Immediately trigger the first print.
		logrus.Info("Scraping:", videoID)

		if err := focusScrape(videoID); err != nil {
			logrus.Errorf("Error in focus mode: %v", err)
			return
		}

		for {
			select {
			case <-fm.ticker.C:
				if err := focusScrape(videoID); err != nil {
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
	for videoID, eventTime := range events {
		delay := time.Until(eventTime)
		if delay < 0 {
			// Skip events that are already in the past.
			logrus.Warnf("Skipping event for %s because event time %s is in the past.", videoID, eventTime.Format(time.RFC3339))

			continue
		}

		// Log that focus mode is scheduled.
		logrus.Infof("Scheduling focus mode for %s at %s (in %s)", videoID, eventTime.Format(time.RFC3339), delay)

		// Schedule startFocusMode to be called at the event time.
		go func(videoID string, delay time.Duration) {
			time.AfterFunc(delay, func() {
				StartFocusMode(videoID)
			})
		}(videoID, delay)
	}
}
