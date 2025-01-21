# Holodex Checker

A Go application that checks for specific video topics on [Holodex](https://holodex.net/) and sends notifications via Telegram and WhatsApp when a match is found.

## Features

- Scrapes the Holodex website for video topics.
- Looks for a video topic named "Singing."
- Sends notifications with details of the video to both Telegram and WhatsApp.
- Supports periodic checks at an hourly interval.
- Includes retry logic for failed network calls.

## Prerequisites

Before running the app, ensure the following prerequisites are in place:

- Google Chrome installed
- Go 1.18+ installed
- `.env` file containing the following environment variables:
  - `TELEGRAM_BOT_TOKEN`: Your Telegram bot token.
  - `TELEGRAM_CHAT_ID`: The Telegram chat ID to send messages to.
  - `WHATSAPP_PHONE_NUMBER`: Your phone number for WhatsApp notifications.
  - `WHATSAPP_API_KEY`: Your API key for sending WhatsApp messages.

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/JaisySymuri/holodex-checker.git
   cd holodex-checker
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Create a `.env` file in the root of the project with the following content (fill in with your respective values):

   ```env
   TELEGRAM_BOT_TOKEN=your-telegram-bot-token
   TELEGRAM_CHAT_ID=your-telegram-chat-id
   WHATSAPP_PHONE_NUMBER=your-whatsapp-phone-number
   WHATSAPP_API_KEY=your-whatsapp-api-key
   ```

## How to Run

Run the application by executing the following command:

```bash
go run main.go
```

The app will attempt to check for the "Singing" video topic immediately and will continue to check every hour thereafter. If it finds the "Singing" topic, it will send details via Telegram and WhatsApp.

## Code Walkthrough

### `checkHolodex` Function
- Uses `chromedp` to automate a headless browser session to scrape the Holodex homepage.
- It looks for video cards with the "Singing" topic and retrieves details about the video.
- Sends a notification via Telegram and WhatsApp if a matching video is found.

### `sendMessageToTelegram` Function
- Sends a message to a specified Telegram chat using the Telegram Bot API.

### `sendMessageToWhatsApp` Function
- Sends a message to a specified WhatsApp phone number using the CallMeBot API.

### `retry` Function
- A utility function to retry failed network calls a specified number of times before giving up.

## Scheduling & Retry Logic
- The app attempts the video check every hour. If the check fails, it will retry up to 30 times with a 10-second delay between attempts.

## Contributing

Feel free to fork the repository, make changes, and create a pull request. Contributions are always welcome!

## License

This project is open-source and available under the MIT License.
```