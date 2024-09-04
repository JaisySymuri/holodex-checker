package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gofrs/flock"
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

func checkHolodex(botToken string, chatID int64, phoneNumber string, apiKey string) {
	// Create a new context for the headless browser
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Define a struct to hold the video topic and channel name
	type VideoInfo struct {
		Topic   string
		Channel string
	}

	// Run the browser actions
	var videoInfos []VideoInfo
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://holodex.net/"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a.video-card.no-decoration.d-flex.video-card-fluid.flex-column')).map(card => {
				const topic = card.querySelector('div.video-topic.rounded-tl-sm')?.innerText.trim() || '';
				const channel = card.querySelector('div.channel-name.video-card-subtitle')?.innerText.trim() || '';
				return { topic, channel };
			});
		`, &videoInfos),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check for Singing topic and send the message
	found := false
	for _, info := range videoInfos {
		if info.Topic == "Singing" {
			message := fmt.Sprintf("Found 'Singing' with channel '%s'\n", info.Channel)

			// Send to Telegram
			if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
				log.Fatal(err)
			}

			// Send to WhatsApp
			if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
				log.Fatal(err)
			}

			fmt.Println(message) // Optional: Also print to the terminal
			found = true
		}
	}
	if !found {
		message := "No 'Singing' topics found."

		// Send to Telegram
		if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
			log.Fatal(err)
		}

		// Send to WhatsApp
		if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
			log.Fatal(err)
		}

		fmt.Println(message) // Optional: Also print to the terminal
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
    lock := flock.NewFlock("app.lock")
    locked, err := lock.TryLock()
    if err != nil {
        log.Fatalf("Failed to acquire lock: %v", err)
    }
    if !locked {
        log.Println("Another instance is already running. Exiting.")
        return
    }
    defer lock.Unlock()

	// Hide the console window
	syscall.NewLazyDLL("kernel32.dll").NewProc("FreeConsole").Call()

	// Bot Token and Chat ID
	botToken := "6644758424:AAGARzGvdtkRs-PKb7-bMol7HIH3Um41NNQ"
	chatID := int64(6250216578)

	// WhatsApp phone number and API key
	phoneNumber := "628813583993"
	apiKey := "9490714"

	/// Run the initial check for Holodex immediately
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
}
