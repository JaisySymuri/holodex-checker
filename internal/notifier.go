package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Only one stream should be retrieved since it filters by the link, but still maintaining the parameter as array? of VideoInfo struct since it's convinient for testing
func focusNotifyMe(videoInfos []APIVideoInfo) error {
	for _, info := range videoInfos {
		fmt.Println("LiveStatus: ", info.Status)
		if info.Status == "live" {
			if err := makeStreamStartMessage(info, botToken, chatID, phoneNumber, apiKey); err != nil {
				return err
			}

		}
	}

	for _, info := range videoInfos {
		if info.Status != "live" {
			logrus.Infof("Focus mode: The stream scheduled for %s - %s hasn't started yet", info.Channel, info.ID)
		}
	}
	return nil
}

func sendMessageToTelegram(botToken string, chatID string, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	resp, err := http.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API failed to receive message, status code: %d", resp.StatusCode)
	}
	return nil
}

func sendMessageToWhatsApp(phoneNumber string, apiKey string, message string) error {
	apiURL := fmt.Sprintf("https://api.callmebot.com/whatsapp.php?phone=%s&text=%s&apikey=%s",
		url.QueryEscape(phoneNumber),
		url.QueryEscape(message),
		apiKey)

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 209 || resp.StatusCode == 210 {
		logrus.Warnf("WhatsApp API returned status code %d. Skipping retry and continuing...", resp.StatusCode)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("whatsApp API failed to receive message, status code: %d", resp.StatusCode)
	}
	return nil
}

func makeFoundMessage(info APIVideoInfo, botToken string, chatID string, phoneNumber string, apiKey string) error {
	var startTime time.Time
	var err error

	if info.StartScheduled != "" {
		startTime, err = time.Parse(time.RFC3339, info.StartScheduled)
		if err != nil {
			logrus.Debugf("Start Scheduled time for %s is not in RFC3339 format: %s", info.ID, info.StartScheduled)
			return fmt.Errorf("failed to parse StartScheduled time: %w", err)
		}
	} else {
		logrus.Debugf("Start Scheduled time for %s is empty, skipping parse", info.ID)
	}

	durationUntilStart := time.Until(startTime)

	message := fmt.Sprintf(
		"Live Status: %s\nAPI: Found '%s' with channel '%s'\nStarts In: %s\n",
		info.TopicID, info.Channel, info.Status, formatDuration(durationUntilStart),
	)

	logrus.Info(message)

	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}

	return nil
}

func makeStreamStartMessage(info APIVideoInfo, botToken string, chatID string, phoneNumber string, apiKey string) error {
	// Extract video ID from "/watch/{videoID}" format
	videoID := strings.TrimPrefix(info.ID, "/watch/")

	// Updated URL format
	message := fmt.Sprintf("%s's karaoke stream has started! - https://www.youtube.com/watch?v=%s", info.Channel, videoID)

	logrus.Info(message)

	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}

	return nil
}

func makeNotFoundMessage(botToken string, chatID string, phoneNumber string, apiKey string) error {
	message := "API: No 'Singing' stream scheduled."

	logrus.Info(message)

	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}

	return nil
}


