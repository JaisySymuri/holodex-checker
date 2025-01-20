package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// Function to send a message to Telegram
func sendMessageToTelegram(botToken string, chatID int64, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	data.Set("text", message)

	resp, err := http.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}
	return nil
}

// Retry function for network calls
func retry(attempts int, sleep time.Duration, fn func() error) error {
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		log.Printf("Attempt %d failed: %v. Retrying in %s...", i+1, err, sleep)
		time.Sleep(sleep)
	}
	return fmt.Errorf("all attempts failed")
}

func checkHolodex(botToken string, chatID int64, phoneNumber string, apiKey string) {
	// Create a new context for the headless browser
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

	var videoInfos []VideoInfo

	// Retry the browser actions with 3 attempts
	err := retry(30, 10*time.Second, func() error {
		return chromedp.Run(ctx,
			chromedp.Navigate("https://holodex.net/"),
			chromedp.WaitVisible(`a.video-card.no-decoration.d-flex.video-card-fluid.flex-column`, chromedp.ByQuery),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('a.video-card.no-decoration.d-flex.video-card-fluid.flex-column')).map(card => {
				const topic = card.querySelector('div.video-topic.rounded-tl-sm')?.innerText.trim() || '';
				const channel = card.querySelector('div.channel-name.video-card-subtitle')?.innerText.trim() || '';
				const liveStatus = card.querySelector('div.video-card-subtitle span.text-live')?.innerText.trim() || '';
				const upcomingStatus = card.querySelector('div.video-card-subtitle span.text-upcoming')?.innerText.trim() || '';
				return { topic, channel, liveStatus, upcomingStatus };
			});`, &videoInfos),
		)
	})
	if err != nil {
		log.Printf("Failed to check Holodex after retries: %v", err)
		return
	}

	// Process video info
	found := false
	for _, info := range videoInfos {
		if info.Topic == "Singing" {
			message := fmt.Sprintf("Found '%s' with channel '%s'\nLive Status: %s\nUpcoming Status: %s\n", info.Topic, info.Channel, info.LiveStatus, info.UpcomingStatus)

			// Send to Telegram and WhatsApp
			retry(30, 10*time.Second, func() error { return sendMessageToTelegram(botToken, chatID, message) })
			retry(30, 10*time.Second, func() error { return sendMessageToWhatsApp(phoneNumber, apiKey, message) })

			fmt.Println(message)
			found = true
		}
	}
	if !found {
		message := "No 'Singing' topics found."

		// Send to Telegram and WhatsApp
		retry(30, 10*time.Second, func() error { return sendMessageToTelegram(botToken, chatID, message) })
		retry(30, 10*time.Second, func() error { return sendMessageToWhatsApp(phoneNumber, apiKey, message) })

		fmt.Println(message)
	}
}

// Function to send a message to WhatsApp using CallMeBot API
func sendMessageToWhatsApp(phoneNumber string, apiKey string, message string) error {
	apiURL := fmt.Sprintf("https://api.callmebot.com/whatsapp.php?phone=%s&text=%s&apikey=%s",
		url.QueryEscape(phoneNumber),
		url.QueryEscape(message),
		apiKey)

	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send WhatsApp message, status code: %d", resp.StatusCode)
	}
	return nil
}


func main() {
	// Bot Token and Chat ID
	botToken := "6644758424:AAGARzGvdtkRs-PKb7-bMol7HIH3Um41NNQ"
	chatID := int64(6250216578)

	// WhatsApp phone number and API key
	phoneNumber := "6289675639535"
	apiKey := "1925640"


	// Run the initial check for Holodex immediately
	checkHolodex(botToken, chatID, phoneNumber, apiKey)

	// Schedule the task to run every hour at the top of the hour
	for {
		now := time.Now()
		// Calculate the next hour's start time
		next := now.Truncate(time.Hour).Add(time.Hour)
		time.Sleep(time.Until(next))

		// Run the check for Holodex
		checkHolodex(botToken, chatID, phoneNumber, apiKey)
	}

	// for {
	// 	// Run the check for Holodex every minute
	// 	checkHolodex(botToken, chatID, phoneNumber, apiKey)
	
	// 	// Sleep for 1 minute before the next run
	// 	time.Sleep(1 * time.Minute)
	// }	
}
