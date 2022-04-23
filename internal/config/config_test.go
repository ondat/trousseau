//go:build !integration
// +build !integration

package config_test

import (
	"log"
	"os"
	"testing"

	cfg "github.com/ondat/trousseau/internal/config"
	"github.com/stretchr/testify/assert"
)

var file = "./configtest.yaml"
var data = []byte(`---
provider: "vault"
vault:
  address: "http://localhost:9200"`)

func TestMain(m *testing.M) {
	setUp()

	retCode := m.Run()

	tearDown()

	os.Exit(retCode)
}

func setUp() {
	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.Write(data)
	f.Close()

	if err != nil {
		log.Fatal(err)
	}
}

func tearDown() {
	if err := os.Remove(file); err != nil {
		log.Fatal(err)
	}
}

func TestParseProvderInConfig(t *testing.T) {
	r, err := cfg.New(file)

	assert.NoError(t, err)
	assert.Equal(t, "vault", r.GetProvider(), "Provider should return vault")
}

func TestParseVaultAddressInConfig(t *testing.T) {
	r, err := cfg.New(file)

	vaultCfg := r.GetVaultConfig()

	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9200", vaultCfg.Address, "Config should return vault address")
}

func BenchmarkCreatingConfig(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := cfg.New(file)
		if err != nil {
			b.Fail()
		}
	}
}
