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
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type VideoInfo struct {
	Topic          string
	Channel        string
	LiveStatus     string
	UpcomingStatus string
}

// Custom Log Formatter
type SimpleFormatter struct{}

// Implementing Logrus Formatter interface
func (f *SimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Formatting the timestamp in a more readable format
	timeFormat := time.Now().Format("2006-01-02 15:04:05") // Example: 2025-01-23 14:45:18
	// Create the log message without log level
	message := fmt.Sprintf("%s %s\n", timeFormat, entry.Message)
	return []byte(message), nil
}


// Function to send a message to Telegram
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

	// Handle specific status codes
	if resp.StatusCode == 209 || resp.StatusCode == 210 {
		logrus.Warnf("WhatsApp API returned status code %d. Skipping retry and continuing...", resp.StatusCode)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("whatsApp API failed to receive message, status code: %d", resp.StatusCode)
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
		logrus.Errorf("Attempt %d failed: %v. Retrying in %s...", i+1, err, sleep)
		time.Sleep(sleep)
	}
	return fmt.Errorf("all attempts failed")
}

// makeFoundMessage generates the message when a "Singing" topic is found
func makeFoundMessage(info VideoInfo, botToken string, chatID string, phoneNumber string, apiKey string) error {
	message := fmt.Sprintf(
		"Ubuntu. Found '%s' with channel '%s'\nLive Status: %s\nUpcoming Status: %s\n",
		info.Topic, info.Channel, info.LiveStatus, info.UpcomingStatus,
	)

	logrus.Info(message)

	// Send to Telegram and WhatsApp
	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}

	return nil
}

// makeNotFoundMessage generates the message when no "Singing" topic is found
func makeNotFoundMessage(botToken string, chatID string, phoneNumber string, apiKey string) error {
	message := "No 'Singing' stream scheduled."

	logrus.Info(message)

	// Send to Telegram and WhatsApp
	if err := sendMessageToTelegram(botToken, chatID, message); err != nil {
		return err
	}
	if err := sendMessageToWhatsApp(phoneNumber, apiKey, message); err != nil {
		return err
	}

	return nil
}

func checkHolodex(botToken string, chatID string, phoneNumber string, apiKey string) error {
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

	var videoInfos []VideoInfo

	// Perform browser actions
	err := chromedp.Run(ctx,
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
	if err != nil {
		// Check if the error is due to "no space left on device"
		if strings.Contains(err.Error(), "no space left on device") {
			// Send disk full message to Telegram and WhatsApp
			if err := makeDiskFullMessage(botToken, chatID, phoneNumber, apiKey); err != nil {
				return err
			}
			logrus.Info("Disk is full. Sleeping for 6 hours to allow cleanup.")
			time.Sleep(6 * time.Hour)
			// Return nil so that (for example) an outer loop can try again.
			return nil
		}
		return fmt.Errorf("failed to fetch data from Holodex: %w", err)
	}

	// Process video info
	found := false
	for _, info := range videoInfos {
		if info.Topic == "Singing" {
			if err := makeFoundMessage(info, botToken, chatID, phoneNumber, apiKey); err != nil {
				return err
			}
			found = true			
		}
	}

	if !found {
		if err := makeNotFoundMessage(botToken, chatID, phoneNumber, apiKey); err != nil {
			return err
		}
	}
	return nil
}

// makeDiskFullMessage sends a disk-full error message to Telegram and WhatsApp.
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

func main() {
	logrus.SetFormatter(&SimpleFormatter{})
	
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Fatalf("Error loading .env file: %v", err)
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	phoneNumber := os.Getenv("WHATSAPP_PHONE_NUMBER")
	apiKey := os.Getenv("WHATSAPP_API_KEY")

	logrus.Info("checkHolodex started. Connecting to internet...")

	// Run the initial check for Holodex immediately
	err = retry(30, 10*time.Second, func() error {
		return checkHolodex(botToken, chatID, phoneNumber, apiKey)
	})
	if err != nil {
		logrus.Error("checkHolodex failed after retries: ", err)
	}

	// Schedule the task to run every hour at the top of the hour
	for {
		now := time.Now()
		// Calculate the next hour's start time
		next := now.Truncate(time.Hour).Add(time.Hour)
		time.Sleep(time.Until(next))

		// Run the check for Holodex
		err = retry(30, 10*time.Second, func() error {
			return checkHolodex(botToken, chatID, phoneNumber, apiKey)
		})
		if err != nil {
			logrus.Error("checkHolodex failed after retries: ", err)
		}
	}

	// for {
	// 	// Run the check for Holodex every minute
	// 	checkHolodex(botToken, chatID, phoneNumber, apiKey)

	// 	// Sleep for 1 minute before the next run
	// 	time.Sleep(1 * time.Minute)
	// }
}
