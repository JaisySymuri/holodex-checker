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

	// Run the browser actions
	var elements []string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://holodex.net/"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.video-topic.rounded-tl-sm')).map(e => e.innerText)`, &elements),
	)
	if err != nil {
		log.Fatal(err)
	}

	if len(elements) > 0 {
		fmt.Println("Found the following elements with class 'video-topic rounded-tl-sm':")
		for _, text := range elements {
			fmt.Println(text)
		}
	} else {
		fmt.Println("No elements with class 'video-topic rounded-tl-sm' found in the page content")
	}
}
