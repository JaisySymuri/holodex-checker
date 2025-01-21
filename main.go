package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	// "os"
	// "os/signal"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gofrs/flock"
	"github.com/joho/godotenv"
)

// Function to send a message to Telegram
func sendMessageToTelegram(botToken string, chatID string, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
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

func checkHolodex(botToken string, chatID string, phoneNumber string, apiKey string) {
	// Create a new context for the headless browser with extended headless options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // Ensure headless mode
		chromedp.Flag("disable-gpu", true), // Disable GPU
		chromedp.Flag("no-sandbox", true),  // Sandbox might cause issues
		chromedp.Flag("disable-software-rasterizer", true), // Disable rasterization
		chromedp.Flag("mute-audio", true),  // Mute audio
		chromedp.Flag("hide-scrollbars", true), // Hide scrollbars to avoid UI
		chromedp.Flag("window-size", "100,100"), // Set window size to make sure it's headless
		chromedp.Flag("disable-extensions", true), // Disable extensions
		chromedp.Flag("remote-debugging-port", "0"), // Disable remote debugging to suppress UI
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

				// Check for both 'text-live' and 'text-upcoming' spans
				const liveStatus = card.querySelector('div.video-card-subtitle span.text-live')?.innerText.trim() || '';
				const upcomingStatus = card.querySelector('div.video-card-subtitle span.text-upcoming')?.innerText.trim() || '';

				return { topic, channel, liveStatus, upcomingStatus };
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
			message := fmt.Sprintf("Found '%s' with channel '%s'\nLive Status: %s\nUpcoming Status: %s\n", info.Topic, info.Channel, info.LiveStatus, info.UpcomingStatus)

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
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	phoneNumber := os.Getenv("WHATSAPP_PHONE_NUMBER")
	apiKey := os.Getenv("WHATSAPP_API_KEY")


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

	// Wait for a minute to allow internet connection to establish
	log.Println("Holodex Checker launched, waiting for 1 minute to allow internet connection to establish...")
	time.Sleep(4 * time.Second)




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
}
