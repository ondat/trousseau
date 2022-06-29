package utils

import "strconv"

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
