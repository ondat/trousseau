package encrypt

import (
	"errors"

	cfg "github.com/ondat/trousseau/internal/config"
	"k8s.io/klog/v2"
)

type EncryptionClient interface {
	Decrypt(data []byte) ([]byte, error)
	Encrypt(data []byte) ([]byte, error)
}

func NewService(config cfg.ProviderConfig) (EncryptionClient, error) {
	switch config.GetProvider() {
	case "vault":
		cfgVault := config.GetVaultConfig()

		client, err := newClientWrapper(&cfgVault)
		if err != nil {
			klog.ErrorS(err, "Unable to create vault client", "server", cfgVault.TLSServerName)
		}

		return client, err
	default:
		return nil, errors.New("unknown provider")
	}
}
