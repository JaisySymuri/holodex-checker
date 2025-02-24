package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gofrs/flock"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)
// Function to send a message to Telegram
func sendMessageToTelegram(botToken string, chatID string, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	logrus.Infof("Sending message to Telegram: %s", message)

	resp, err := http.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		logrus.Errorf("Failed to send Telegram message: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Telegram API response status code: %d", resp.StatusCode)
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}

	logrus.Info("Telegram message sent successfully")
	return nil
}

// Function to send a message to WhatsApp using CallMeBot API
func sendMessageToWhatsApp(phoneNumber string, apiKey string, message string) error {
	apiURL := fmt.Sprintf("https://api.callmebot.com/whatsapp.php?phone=%s&text=%s&apikey=%s",
		url.QueryEscape(phoneNumber),
		url.QueryEscape(message),
		apiKey)

	logrus.Infof("Sending message to WhatsApp: %s", message)

	resp, err := http.Get(apiURL)
	if err != nil {
		logrus.Errorf("Failed to send WhatsApp message: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("WhatsApp API response status code: %d", resp.StatusCode)
		return fmt.Errorf("failed to send WhatsApp message, status code: %d", resp.StatusCode)
	}

	logrus.Info("WhatsApp message sent successfully")
	return nil
}

func checkHolodex(botToken string, chatID string, phoneNumber string, apiKey string) {
	logrus.Info("Starting Holodex check...")

	// Create a new context for the headless browser with extended headless options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancelCtx := chromedp.NewContext(allocatorCtx)
	defer cancelCtx()

	type VideoInfo struct {
		Topic          string
		Channel        string
		LiveStatus     string
		UpcomingStatus string
	}

	// Run the browser actions
	var videoInfos []VideoInfo
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://holodex.net/"),
		chromedp.WaitVisible(`a.video-card.no-decoration.d-flex.video-card-fluid.flex-column`, chromedp.ByQuery),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a.video-card.no-decoration.d-flex.video-card-fluid.flex-column')).map(card => {
				const topic = card.querySelector('div.video-topic.rounded-tl-sm')?.innerText.trim() || '';
				const channel = card.querySelector('div.channel-name.video-card-subtitle')?.innerText.trim() || '';
				const liveStatus = card.querySelector('div.video-card-subtitle span.text-live')?.innerText.trim() || '';
				const upcomingStatus = card.querySelector('div.video-card-subtitle span.text-upcoming')?.innerText.trim() || '';
				return { topic, channel, liveStatus, upcomingStatus };
			});
		`, &videoInfos),
	)
	if err != nil {
		logrus.Error("Chromedp execution failed: ", err)
		return
	}

	logrus.Infof("Fetched %d videos from Holodex", len(videoInfos))

	// Check for Singing topic
	found := false
	for _, info := range videoInfos {
		if info.Topic == "Singing" {
			message := fmt.Sprintf("Found '%s' with channel '%s'\nLive Status: %s\nUpcoming Status: %s\n",
				info.Topic, info.Channel, info.LiveStatus, info.UpcomingStatus)

			logrus.Info("Sending message: ", message)

			if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
				logrus.Error("Failed to send message to Telegram: ", err)
			}

			if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
				logrus.Error("Failed to send message to WhatsApp: ", err)
			}

			found = true
		}
	}
	if !found {
		noTopicMessage := "Windows: No 'Singing' topics found."
		logrus.Info(noTopicMessage)

		if err := sendMessageToTelegram(botToken, chatID, noTopicMessage); err != nil {
			logrus.Error("Failed to send 'No Singing' message to Telegram: ", err)
		}

		if err := sendMessageToWhatsApp(phoneNumber, apiKey, noTopicMessage); err != nil {
			logrus.Error("Failed to send 'No Singing' message to WhatsApp: ", err)
		}
	}
}



func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	err := godotenv.Load(".env")
	if err != nil {
		logrus.Error("Error loading .env file: ", err)
		return
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	phoneNumber := os.Getenv("WHATSAPP_PHONE_NUMBER")
	apiKey := os.Getenv("WHATSAPP_API_KEY")

	lock := flock.NewFlock("app.lock")
	locked, err := lock.TryLock()
	if err != nil {
		logrus.Error("Failed to acquire lock: ", err)
		return
	}
	if !locked {
		logrus.Warn("Another instance is already running. Exiting.")
		return
	}
	defer lock.Unlock()

	logrus.Info("Holodex Checker launched, waiting 4s to allow internet connection to establish...")
	time.Sleep(4 * time.Second)

	// Run the initial check for Holodex
	checkHolodex(botToken, chatID, phoneNumber, apiKey)

	// Schedule the task to run every hour at the top of the hour
	for {
		now := time.Now()
		next := now.Truncate(time.Hour).Add(time.Hour)
		sleepDuration := time.Until(next)

		logrus.Infof("Sleeping for %v until next check...", sleepDuration)
		time.Sleep(sleepDuration)

		checkHolodex(botToken, chatID, phoneNumber, apiKey)
	}
}
