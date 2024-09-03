package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/chromedp"
)

func main() {
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

	// Print results
	found := false
	for _, info := range videoInfos {
		if info.Topic == "Minecraft" {
			fmt.Printf("Found 'Minecraft' with channel '%s'\n", info.Channel)
			found = true
		}
	}
	if !found {
		fmt.Println("No 'Minecraft' topics found.")
	}
}
