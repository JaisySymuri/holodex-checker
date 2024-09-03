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

	// Define a slice to hold the elements' HTML
	var elementsHTML []string

	// Extract elements with the specified class and their HTML
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://holodex.net/"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.video-topic.rounded-tl-sm'))
			.map(e => ({ html: e.outerHTML, text: e.innerText }))
			.filter(item => item.text === 'Minecraft')`, &elementsHTML),
	)
	if err != nil {
		log.Fatal(err)
	}

	if len(elementsHTML) > 0 {
		fmt.Println("Found the following elements with text 'Minecraft':")
		for _, elementHTML := range elementsHTML {
			fmt.Println(elementHTML)
		}
	} else {
		fmt.Println("No elements with text 'Minecraft' found in the page content")
	}
}
