package main

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	// Load environment variables once before all tests
	setEnv()

	// Set log level to Debug
	logrus.SetLevel(logrus.DebugLevel)

	// Run tests
	os.Exit(m.Run())
}
