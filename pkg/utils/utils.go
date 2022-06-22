package utils

// SecretToLog truncates secret to log.
func SecretToLog(s string) string {
	b := []byte(s)
	return string(b[:len(b)/2]) + "..."
}
