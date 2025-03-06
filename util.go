package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

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

func setEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Fatalf("Error loading .env file: %v", err)
	}

	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID = os.Getenv("TELEGRAM_CHAT_ID")
	phoneNumber = os.Getenv("WHATSAPP_PHONE_NUMBER")
	apiKey = os.Getenv("WHATSAPP_API_KEY")
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

// StopFocusModeForFirst stops focus mode for the earliest stream.
func (km *KaraokeManager) StopFocusModeForFirst() {
	km.mu.RLock()
	defer km.mu.RUnlock()
	for link := range km.streams {
		stopFocusMode(link)
		return
	}
}
