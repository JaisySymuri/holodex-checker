package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/getlantern/systray"
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
	timeFormat := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("%s %s\n", timeFormat, entry.Message)
	return []byte(message), nil
}

var (
	botToken    string
	chatID      string
	phoneNumber string
	apiKey      string
	running     bool = true
)

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

func checkHolodex(botToken string, chatID string, phoneNumber string, apiKey string) error {
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
		if strings.Contains(err.Error(), "no space left on device") {
			if err := makeDiskFullMessage(botToken, chatID, phoneNumber, apiKey); err != nil {
				return err
			}
			logrus.Info("Disk is full. Sleeping for 6 hours to allow cleanup.")
			time.Sleep(6 * time.Hour)
			return nil
		}
		return fmt.Errorf("failed to fetch data from Holodex: %w", err)
	}

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

func onReady() {
	iconData, err := os.ReadFile("favicon.ico")
	if err != nil {
		logrus.Fatalf("Failed to read icon file: %v", err)
	}

	systray.SetIcon(iconData)
	systray.SetTitle("Holodex Checker")
	systray.SetTooltip("Holodex Checker")

	startMenuItem := systray.AddMenuItem("Start", "Start checking Holodex")
	pauseMenuItem := systray.AddMenuItem("Pause", "Pause checking Holodex")
	restartMenuItem := systray.AddMenuItem("Restart", "Restart checking Holodex")
	exitMenuItem := systray.AddMenuItem("Exit", "Exit the application")
	hideConsoleMenuItem := systray.AddMenuItem("Hide Console", "Hide the console window")

	go func() {
		for {
			select {
			case <-startMenuItem.ClickedCh:
				if (!running) {
					running = true
					logrus.Info("checkHolodex started")
					go runChecker()
				}
			case <-pauseMenuItem.ClickedCh:
				if (running) {
					running = false
					logrus.Info("checkHolodex paused")
				}
			case <-restartMenuItem.ClickedCh:
				running = false
				logrus.Info("checkHolodex restarting")
				time.Sleep(2 * time.Second)
				running = true
				go runChecker()
			case <-hideConsoleMenuItem.ClickedCh:
				syscall.NewLazyDLL("kernel32.dll").NewProc("FreeConsole").Call()
				logrus.Info("Console window hidden")
			case <-exitMenuItem.ClickedCh:
				systray.Quit()
				logrus.Info("Exiting...")
				return
			}
		}
	}()
}

func runChecker() {
	// Run the initial check for Holodex immediately
	err := retry(30, 10*time.Second, func() error {
		return checkHolodex(botToken, chatID, phoneNumber, apiKey)
	})
	if err != nil {
		logrus.Error("checkHolodex failed after retries: ", err)
	}

	// Schedule the task to run every hour at the top of the hour
	for running {
		now := time.Now()
		next := now.Truncate(time.Hour).Add(time.Hour)
		time.Sleep(time.Until(next))

		err = retry(30, 10*time.Second, func() error {
			return checkHolodex(botToken, chatID, phoneNumber, apiKey)
		})
		if err != nil {
			logrus.Error("checkHolodex failed after retries: ", err)
		}
	}
}

func onExit() {
	logrus.Info("Application exited")
}


// Custom log writer to prepend logs to a file
type PrependFileWriter struct {
	filename string
}

func (w *PrependFileWriter) Write(p []byte) (n int, err error) {
	// Read existing content
	content, err := os.ReadFile(w.filename)
	if err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to read existing log file: %w", err)
	}

	// Prepend new log entry
	newContent := append(p, content...)

	// Write updated content back to the file
	err = os.WriteFile(w.filename, newContent, 0666)
	if err != nil {
		return 0, fmt.Errorf("failed to write log file: %w", err)
	}

	return len(p), nil
}

// Custom multi-writer to handle os.Stdout and file separately
type SafeMultiWriter struct {
	writers []io.Writer
}

func (w *SafeMultiWriter) Write(p []byte) (n int, err error) {
	for _, writer := range w.writers {
		_, err := writer.Write(p)
		if err != nil && writer == os.Stdout {
			// Ignore os.Stdout errors
			continue
		} else if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func main() {
	logrus.SetFormatter(&SimpleFormatter{})

	err := godotenv.Load(".env")
	if err != nil {
		logrus.Fatalf("Error loading .env file: %v", err)
	}

	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID = os.Getenv("TELEGRAM_CHAT_ID")
	phoneNumber = os.Getenv("WHATSAPP_PHONE_NUMBER")
	apiKey = os.Getenv("WHATSAPP_API_KEY")

	logrus.Info("checkHolodex started. Connecting to internet...")

	// Set up logging to both terminal and debug.log file
	fileWriter := &PrependFileWriter{filename: "debug.log"}
	multiWriter := &SafeMultiWriter{writers: []io.Writer{os.Stdout, fileWriter}}
	logrus.SetOutput(multiWriter)


	go runChecker()
	systray.Run(onReady, onExit)
}