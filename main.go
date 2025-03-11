package main

import (
	"time"

	"github.com/getlantern/systray"
	"github.com/sirupsen/logrus"
)

func mainLogic(km *KaraokeManager) {
	

	hScraper := &HolodexScraper{}
	var karaokeStreams []VideoInfo

	err := retry(30, 10*time.Second, func() error {
		return hScraper.checkHolodex("https://holodex.net/")
	})
	if err != nil {
		logrus.Error("checkHolodex failed after retries: ", err)
	}

	err = retry(30, 10*time.Second, func() error {
		var err error
		karaokeStreams, err = karaokeHandler(hScraper.videoInfos)
		return err
	})

	if err != nil {
		logrus.Error("notifyMe failed after retries: ", err)
	}

	ks, err := getStartTime(karaokeStreams)
	if err != nil {
		logrus.Error("Errors encountered while retrieving start times: ", err)
	}

	// Update the manager.
	km.SetStreams(ks)

	// Schedule focus mode for each karaoke stream.
	go scheduleFocusMode(ks)	
}

func main() {
	setLog()
	setEnv()

	km := NewKaraokeManager()

	logrus.Info("checkHolodex started. Connecting to internet...")

	go func() {
		// Run the initial check for Holodex immediately
		mainLogic(km)

		// Run the check for Holodex every hour
		go func() {
			for running {
				now := time.Now()
				next := now.Truncate(time.Hour).Add(time.Hour)
				time.Sleep(time.Until(next))		
				mainLogic(km)
			}
		}()
	}()

	systray.Run(func() { onReady(km) }, onExit)
}
