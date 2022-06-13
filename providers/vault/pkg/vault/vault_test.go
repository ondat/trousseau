//go:build !integration
// +build !integration

package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getVaultConfig() *Config {
	return &Config{
		Address: "http://localhost:9200",
		Token:   "test",
	}
}

func TestCreatingVaultClient(t *testing.T) {
	_, err := New(getVaultConfig())
	assert.NoError(t, err)
}
