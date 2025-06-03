package main

import (
	"holodex-checker-windows/internal"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	// Load environment variables once before all tests
	internal.SetEnv()

	// Set log level to Debug
	logrus.SetLevel(logrus.DebugLevel)

	// Run tests
	os.Exit(m.Run())
}
