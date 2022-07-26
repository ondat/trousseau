package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	// -X github.com/ondat/trousseau/pkg/utils.SecretLogDivider=1
	SecretLogDivider string
	secretLogDivider = 2
)

func init() {
	if SecretLogDivider != "" {
		var err error

		secretLogDivider, err = strconv.Atoi(SecretLogDivider)
		if err != nil || secretLogDivider <= 0 {
			panic("Invalid github.com/ondat/trousseau/pkg/utils.SecretLogDivider=" + SecretLogDivider)
		}
	}
}

// SecretToLog truncates secret to log.
func SecretToLog(s string) string {
	b := []byte(s)

	var suffix string
	if secretLogDivider > 1 {
		suffix = "..."
	}

	return string(b[:len(b)/secretLogDivider]) + suffix
}

// RemoveFile removes given file.
func RemoveFile(path string) error {
	path = filepath.Clean(path)

	if _, err := os.Stat(path); err != nil {
		return nil
	}

	return os.Remove(path)
}

func WatchFile(path string) <-chan error {
	errChan := make(chan error)
	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			<-ticker.C

			if _, err := os.Stat(path); err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}
