package main

import (
	"os"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"github.com/sirupsen/logrus"
)

func onReady(km *KaraokeManager) {
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
	stopFocusMode := systray.AddMenuItem("Stop focus", "Stopping focus mode for the earliest stream")

	go func() {
		for {
			select {
			case <-startMenuItem.ClickedCh:
				if !running {
					running = true
					logrus.Info("checkHolodex started")
					go mainLogic(km)
				}
			case <-pauseMenuItem.ClickedCh:
				if running {
					running = false
					logrus.Info("checkHolodex paused")
				}
			case <-restartMenuItem.ClickedCh:
				running = false
				logrus.Info("checkHolodex restarting")
				time.Sleep(2 * time.Second)
				running = true
				go mainLogic(km)
			case <-hideConsoleMenuItem.ClickedCh:
				syscall.NewLazyDLL("kernel32.dll").NewProc("FreeConsole").Call()
				logrus.Info("Console window hidden")
			case <-stopFocusMode.ClickedCh:
				km.StopFocusModeForFirst()
			case <-exitMenuItem.ClickedCh:
				systray.Quit()
				logrus.Info("Exiting...")
				return
			}
		}
	}()
}

func onExit() {
	logrus.Info("Application exited")
}
