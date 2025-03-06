package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Custom Log Formatter
type SimpleFormatter struct{}

func setLog() {
	logrus.SetFormatter(&SimpleFormatter{})

	// Set up logging to both terminal and debug.log file
	fileWriter := &PrependFileWriter{filename: "debug.log"}
	multiWriter := &SafeMultiWriter{writers: []io.Writer{os.Stdout, fileWriter}}
	logrus.SetOutput(multiWriter)

}

// Implementing Logrus Formatter interface
func (f *SimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timeFormat := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("%s %s\n", timeFormat, entry.Message)
	return []byte(message), nil
}

// Custom log writer to prepend logs to a file
type PrependFileWriter struct {
	filename string
}

func (w *PrependFileWriter) Write(p []byte) (n int, err error) {
	// Read existing content
	content, err := os.ReadFile(w.filename)
	if err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to read existing log file: %w", err)
	}

	// Prepend new log entry
	newContent := append(p, content...)

	// Write updated content back to the file
	err = os.WriteFile(w.filename, newContent, 0666)
	if err != nil {
		return 0, fmt.Errorf("failed to write log file: %w", err)
	}

	return len(p), nil
}

// Custom multi-writer to handle os.Stdout and file separately
type SafeMultiWriter struct {
	writers []io.Writer
}

func (w *SafeMultiWriter) Write(p []byte) (n int, err error) {
	for _, writer := range w.writers {
		_, err := writer.Write(p)
		if err != nil && writer == os.Stdout {
			// Ignore os.Stdout errors
			continue
		} else if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
