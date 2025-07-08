package internal

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Simulate timeNow function
func timeNow() time.Time {
	return time.Now()
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		logrus.Errorf("Attempt %d failed: %v. Retrying in %s...", i+1, err, sleep)
		time.Sleep(sleep)
	}
	return fmt.Errorf("all attempts failed")
}

func SetEnv() {
	logrus.Info("Step 1: Calling inside the setENV")

	err := godotenv.Load(".env")
	if err != nil {
		logrus.Fatalf("Error loading .env file: %v", err)
	}

	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID = os.Getenv("TELEGRAM_CHAT_ID")
	phoneNumber = os.Getenv("WHATSAPP_PHONE_NUMBER")
	apiKey = os.Getenv("WHATSAPP_API_KEY")
	xApiKey = os.Getenv("XAPIKEY")
}

type KaraokeManager struct {
	streams map[string]time.Time
	mu      sync.RWMutex
}

func NewKaraokeManager() *KaraokeManager {
	return &KaraokeManager{
		streams: make(map[string]time.Time),
	}
}

// SetStreams replaces the current streams with new ones.
func (km *KaraokeManager) SetStreams(newStreams map[string]time.Time) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.streams = newStreams
}

// GetStreams returns a copy of the current streams.
func (km *KaraokeManager) GetStreams() map[string]time.Time {
	km.mu.RLock()
	defer km.mu.RUnlock()
	copy := make(map[string]time.Time, len(km.streams))
	for k, v := range km.streams {
		copy[k] = v
	}
	return copy
}

func Monitor(km *KaraokeManager) {
	apiClient := NewAPIClient(xApiKey)
	var karaokeStreams []APIVideoInfo

	err := retry(30, 10*time.Second, func() error {
		var err error
		karaokeStreams, err = apiClient.FetchVideos("Hololive", "singing")
		return err
	})
	if err != nil {
		logrus.Error("FetchVideos failed after retries: ", err)
	}	

	// same retry + handler
	err = retry(30, 10*time.Second, func() error {
		var err error
		karaokeStreams, err = karaokeHandler(karaokeStreams)
		return err
	})
	if err != nil {
		logrus.Error("karaokeHandler failed after retries: ", err)
	}

	ks, err := getStartTime(karaokeStreams)
	if err != nil {
		logrus.Error("Error retrieving start times: ", err)
	}

	km.SetStreams(ks)
	go scheduleFocusMode(ks)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "already started"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}


