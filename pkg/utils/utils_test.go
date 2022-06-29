package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretToLog(t *testing.T) {
	assert.Equal(t, "ab...", SecretToLog("abcd"), "Wrong secret part returned")
}
