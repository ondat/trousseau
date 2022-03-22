package encrypt

import (
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"
	"sync"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/ondat/trousseau/internal/config"
	errors "github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const defaultTransitPath = "transit"
const defaultAuthPath = "auth"

// Handle all communication with Vault server.
type vaultWrapper struct {
	client      *vaultapi.Client
	encryptPath string
	decryptPath string
	authPath    string
	rwmutex     sync.RWMutex
	config      *config.VaultConfig
}

// Initialize a client wrapper for vault kms provider.
func newClientWrapper(config *config.VaultConfig) (*vaultWrapper, error) {
	client, err := newVaultApiClient(config)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to create vault client")
	}

	// Vault transit path is configurable. "path", "/path", "path/" and "/path/"
	// are the same.
	transit := defaultTransitPath
	if config.TransitPath != "" {
		transit = config.TransitPath
	}

	// auth path is configurable. "path", "/path", "path/" and "/path/" are the same.
	auth := defaultAuthPath
	if config.AuthPath != "" {
		auth = config.AuthPath
	}
	wrapper := &vaultWrapper{
		client:      client,
		encryptPath: path.Join("v1", transit, "encrypt"),
		decryptPath: path.Join("v1", transit, "decrypt"),
		authPath:    path.Join(auth),
		config:      config,
	}

	// Set token for the vaultapi.client.
	if len(config.Token) != 0 {
		client.SetToken(config.Token)
	} else {
		if err := wrapper.getInitialToken(config); err != nil {
			return nil, errors.WithMessage(err, "unable to get initial token")
		}
	}

	return wrapper, nil
}

func newVaultApiClient(config *config.VaultConfig) (*vaultapi.Client, error) {
	vaultConfig := vaultapi.DefaultConfig()
	vaultConfig.Address = config.Address

	tlsConfig := &vaultapi.TLSConfig{
		CACert:        config.VaultCACert,
		ClientCert:    config.ClientCert,
		ClientKey:     config.ClientKey,
		TLSServerName: config.TLSServerName,
	}
	if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
		return nil, errors.WithMessagef(err, "unable to configure TLS for %s", config.TLSServerName)
	}

	return vaultapi.NewClient(vaultConfig)
}

func (c *vaultWrapper) getInitialToken(config *config.VaultConfig) error {
	switch {
	case config.ClientCert != "" && config.ClientKey != "":
		token, err := c.tlsToken(config)
		if err != nil {
			return errors.Wrap(err, "rotating token through TLS auth backend")
		}
		c.client.SetToken(token)
	case config.RoleID != "":
		token, err := c.appRoleToken(config)
		if err != nil {
			return errors.Wrap(err, "rotating token through app role backend")
		}
		c.client.SetToken(token)
	default:
		// configuration has already been validated, flow should not reach here
		return errors.New("the Vault authentication configuration is invalid")
	}

	return nil
}

func (c *vaultWrapper) tlsToken(config *config.VaultConfig) (string, error) {
	path := path.Join("/", c.authPath, "cert", "login")
	resp, err := c.client.Logical().Write(path, nil)
	if err != nil {
		return "", errors.WithMessagef(err, "unable to write TLS via API on %s", path)
	} else if resp.Auth == nil {
		return "", errors.New("authentication information not found")
	}

	return resp.Auth.ClientToken, nil
}

func (c *vaultWrapper) appRoleToken(config *config.VaultConfig) (string, error) {
	data := map[string]interface{}{
		"role_id":   config.RoleID,
		"secret_id": config.SecretID,
	}
	path := path.Join("/", c.authPath, "approle", "login")
	resp, err := c.client.Logical().Write(path, data)
	if err != nil {
		return "", errors.WithMessagef(err, "unable to write app role token via API on %s", path)
	} else if resp.Auth == nil {
		return "", errors.New("authentication information not found")
	}

	return resp.Auth.ClientToken, nil
}
func (c *vaultWrapper) Encrypt(data []byte) ([]byte, error) {
	response, err := c.withRefreshToken(true, c.config.KeyNames[0], string(data))
	if err != nil {
		return nil, errors.WithMessage(err, "unable to encrypt data")
	}
	return []byte(response), nil

}
func (c *vaultWrapper) Decrypt(data []byte) ([]byte, error) {
	response, err := c.withRefreshToken(false, c.config.KeyNames[0], string(data))
	if err != nil {
		return nil, errors.WithMessage(err, "unable to decrypt data")
	}
	return []byte(response), nil
}

func (c *vaultWrapper) request(path string, data interface{}) (*vaultapi.Secret, error) {
	req := c.client.NewRequest("POST", "/"+path)
	if err := req.SetJSONBody(data); err != nil {
		return nil, errors.WithMessagef(err, "unable to set request JSON on %s", path)
	}

	resp, err := c.client.RawRequest(req)
	if err != nil {
		if strings.Contains(err.Error(), "Code: 403") {
			return nil, &forbiddenError{err: err}
		}
		return nil, errors.Wrapf(err, "error making POST request on %s", path)
	}
	if resp == nil {
		return nil, errors.Wrapf(err, "no response received for POST request on %s", path)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected response code: %v received for POST request to %v", resp.StatusCode, path)
	}
	return vaultapi.ParseSecret(resp.Body)
}

// Return this error when get HTTP code 403.
type forbiddenError struct {
	err error
}

func (e *forbiddenError) Error() string {
	return fmt.Sprintf("error %s", e.err)
}

func (c *vaultWrapper) withRefreshToken(isEncrypt bool, key, data string) (string, error) {
	// Execute operation first time.
	var result string
	var err error
	func() {
		c.rwmutex.RLock()
		defer c.rwmutex.RUnlock()
		if isEncrypt {
			result, err = c.encryptLocked(key, data)
		} else {
			result, err = c.decryptLocked(key, data)
		}
	}()
	if err == nil || c.config.Token != "" {
		return result, nil
	}
	_, ok := err.(*forbiddenError)
	if !ok {
		return result, err
	}
	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()
	err = c.refreshTokenLocked(c.config)
	if err != nil {
		return result, errors.Wrap(err, "error refresh token request")
	}
	klog.V(2).Infof("vault token refreshed")
	if isEncrypt {
		result, err = c.encryptLocked(key, data)
	} else {
		result, err = c.decryptLocked(key, data)
	}
	return result, err
}
func (c *vaultWrapper) refreshTokenLocked(config *config.VaultConfig) error {
	return c.getInitialToken(config)
}

func (c *vaultWrapper) encryptLocked(key string, data string) (string, error) {
	dataReq := map[string]string{"plaintext": data}
	resp, err := c.request(path.Join(c.encryptPath, key), dataReq)
	if err != nil {
		return "", errors.Wrap(err, "error during encrypt request")
	}

	result, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return result, errors.Errorf("failed type assertion of vault encrypt response type: %v to string", reflect.TypeOf(resp.Data["ciphertext"]))
	}

	return result, nil

}

func (c *vaultWrapper) decryptLocked(key string, data string) (string, error) {
	dataReq := map[string]string{"ciphertext": string(data)}
	resp, err := c.request(path.Join(c.decryptPath, c.config.KeyNames[0]), dataReq)
	if err != nil {
		return "", errors.Wrap(err, "error during decrypt request")
	}

	result, ok := resp.Data["plaintext"].(string)
	if !ok {
		return result, errors.Errorf("failed type assertion of vault decrypt response type: %v to string", reflect.TypeOf(resp.Data["plaintext"]))
	}

	return result, nil
}
