package main

import (
	"holodex-checker-windows/internal"
	"time"

	"github.com/getlantern/systray"
	"github.com/sirupsen/logrus"
)



func main() {
	internal.SetLog()
	internal.SetEnv()

	km := internal.NewKaraokeManager()

	logrus.Info("checkHolodex started. Connecting to internet...")

	go func() {
		// Run the initial check for Holodex immediately
		internal.Monitor(km)

		// Run the check for Holodex every hour
		go func() {
			for internal.Running {
				now := time.Now()
				next := now.Truncate(time.Hour).Add(time.Hour)
				time.Sleep(time.Until(next))		
				internal.Monitor(km)
			}
		}()
	}()

	systray.Run(func() { internal.OnReady(km) }, internal.OnExit)
}
