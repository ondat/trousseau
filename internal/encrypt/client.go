package encrypt

import (
	errors "errors"

	cfg "github.com/ondat/trousseau/internal/config"
)

type EncryptionClient interface {
	Decrypt(data []byte) ([]byte, error)
	Encrypt(data []byte) ([]byte, error)
}

func NewService(config cfg.ProviderConfig) (EncryptionClient, error) {
	switch config.GetProvider() {
	case "vault":
		cfgVault := config.GetVaultConfig()
		return newClientWrapper(&cfgVault)
	default:
		return nil, errors.New("unknown provider")
	}
}
