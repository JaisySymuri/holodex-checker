package main

import (
	"time"

	"github.com/getlantern/systray"
	"github.com/sirupsen/logrus"
)

func mainLogic(km *KaraokeManager) {
	// Run the initial check for Holodex immediately

	hScraper := &HolodexScraper{}

	err := retry(30, 10*time.Second, func() error {
		return hScraper.checkHolodex("https://holodex.net/")
	})
	if err != nil {
		logrus.Error("checkHolodex failed after retries: ", err)
	}

	err = retry(30, 10*time.Second, func() error {
		return notifyMe(hScraper.videoInfos)
	})
	if err != nil {
		logrus.Error("notifyMe failed after retries: ", err)
	}

	ks, err := getStartTime(hScraper.videoInfos)
	if err != nil {
		logrus.Error("Errors encountered while retrieving start times: ", err)
	}

	// Update the manager.
	km.SetStreams(ks)

	// Schedule focus mode for each karaoke stream.
	scheduleFocusMode(ks)

	// Schedule the task to run every hour at the top of the hour
	for running {
		now := time.Now()
		next := now.Truncate(time.Hour).Add(time.Hour)
		time.Sleep(time.Until(next))

		err = retry(30, 10*time.Second, func() error {
			return hScraper.checkHolodex("https://holodex.net/")
		})
		if err != nil {
			logrus.Error("checkHolodex failed after retries: ", err)
		}
	}
}

func main() {
	setLog()
	setEnv()

	km := NewKaraokeManager()

	logrus.Info("checkHolodex started. Connecting to internet...")

	go func() {
		mainLogic(km)
	}()

	systray.Run(func() { onReady(km) }, onExit)
}
