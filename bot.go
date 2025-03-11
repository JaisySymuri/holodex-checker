package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	botToken    string
	chatID      string
	phoneNumber string
	apiKey      string
	running     bool = true
)

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

// Only one stream should be retrieved since it filters by the link, but still maintaining the parameter as array? of VideoInfo struct since it's convinient for testing
func focusNotifyMe(videoInfos []VideoInfo) error {

	for _, info := range videoInfos {
		if info.Duration != "" {
			if err := makeStreamStartMessage(info, botToken, chatID, phoneNumber, apiKey); err != nil {
				return err
			}

		}
	}

	for _, info := range videoInfos {
		if info.Duration == "" {
			logrus.Infof("Focus mode: The stream scheduled for %s - %s hasn't started yet", info.Channel, info.YoutubeLink)
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

func makeFoundMessage(info VideoInfo, botToken string, chatID string, phoneNumber string, apiKey string) error {
	message := fmt.Sprintf(
		"Windows: Found '%s' with channel '%s'\nLive Status: %s\nUpcoming Status: %s\n",
		info.Topic, info.Channel, info.LiveStatus, info.UpcomingStatus,
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

func makeStreamStartMessage(info VideoInfo, botToken string, chatID string, phoneNumber string, apiKey string) error {
	// Extract video ID from "/watch/{videoID}" format
	videoID := strings.TrimPrefix(info.YoutubeLink, "/watch/")

	message := fmt.Sprintf("%s's karaoke stream has started! - https://youtu.be/%s", info.Channel, videoID)

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
	message := "Windows: No 'Singing' stream scheduled."

	logrus.Info(message)

	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}

	return nil
}

func makeDiskFullMessage(botToken string, chatID string, phoneNumber string, apiKey string) error {
	message := "Error: no space left on device. Disk is full. The app will sleep for 6 hours until cleanup occurs."
	logrus.Error(message)

	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}
	return nil
}
