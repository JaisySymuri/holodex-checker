package internal

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Custom Log Formatter
type SimpleFormatter struct{}

func (f *SimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timeFormat := entry.Time.Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("%s %s\n", timeFormat, entry.Message)
	return []byte(message), nil
}

func SetLog() {
    logrus.SetFormatter(&SimpleFormatter{})

    // Open log file for append, create if not exists
    logFile, err := os.OpenFile("debug2.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        logrus.Fatalf("Failed to open log file: %v", err)
    }

    // Redirect os.Stdout to a file
    stdoutFile, err := os.OpenFile("stdout.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        logrus.Fatalf("Failed to open stdout file: %v", err)
    }
    os.Stdout = stdoutFile

    // Write logs to both redirected stdout and log file
    multiWriter := io.MultiWriter(os.Stdout, logFile)
    logrus.SetOutput(multiWriter)
}

