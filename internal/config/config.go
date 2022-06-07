package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ondat/trousseau/internal/logger"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

type ProviderConfig interface {
	GetProvider() string
	GetVaultConfig() VaultConfig
}

func New(cfpPath string) (ProviderConfig, error) {
	klog.V(logger.Info2).InfoS("Populating AppConfig...", "path", cfpPath)

	viper.SetConfigType("yaml")

	file, err := os.ReadFile(filepath.Clean(cfpPath))
	if err != nil {
		klog.ErrorS(err, "Unable to open config file", "path", cfpPath)
		return nil, fmt.Errorf("unable to open config file %s: %w", cfpPath, err)
	}

	err = viper.ReadConfig(bytes.NewBuffer(file))
	if err != nil {
		klog.ErrorS(err, "Unable to read config file", "path", cfpPath)
		return nil, fmt.Errorf("unable to read config file %s: %w", cfpPath, err)
	}

	var cfg appConfig
	if err = viper.Unmarshal(&cfg); err != nil {
		klog.ErrorS(err, "Unable to unmarshal config file", "path", cfpPath)
		return nil, fmt.Errorf("unable to unmarshal config file %s: %w", cfpPath, err)
	}

	klog.V(logger.Debug2).Info("Current config", "config", cfg)

	return &cfg, nil
}

type appConfig struct {
	Provider string
	Vault    VaultConfig
}

func (c *appConfig) GetProvider() string {
	return c.Provider
}

func (c *appConfig) GetVaultConfig() VaultConfig {
	return c.Vault
}

type VaultConfig struct {
	// The names of encryption key for Vault transit communication
	KeyNames []string `json:"keyNames"`

	// Vault listen address, for example https://localhost:8200
	Address string `json:"addr"`

	// Token authentication information
	Token string `json:"token"`

	// TLS certificate authentication information
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`

	// AppRole authentication information
	RoleID   string `json:"roleID"`
	SecretID string `json:"secretID"`

	// CACert is the path to a PEM-encoded CA cert file to use to verify the
	// Vault server SSL certificate.
	VaultCACert string `json:"vaultCACert"`

	// TLSServerName, if set, is used to set the SNI host when connecting via TLS.
	TLSServerName string `json:"tlsServerName"`

	// The path for transit API, default is "transit"
	TransitPath string `json:"transitPath"`

	// The path for auth backend, default is "auth"
	AuthPath string `json:"authPath"`
}
