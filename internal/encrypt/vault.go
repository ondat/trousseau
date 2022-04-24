package encrypt

import (
	"fmt"
	"net/http"
	"path"
	"reflect"
	"sync"

	"errors"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/ondat/trousseau/internal/config"
	"k8s.io/klog/v2"
)

const (
	defaultTransitPath = "transit"
	defaultAuthPath    = "auth"
)

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
func newClientWrapper(vaultConfig *config.VaultConfig) (*vaultWrapper, error) {
	client, err := newVaultAPIClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create vault client: %w", err)
	}

	// Vault transit path is configurable. "path", "/path", "path/" and "/path/"
	// are the same.
	transit := defaultTransitPath
	if vaultConfig.TransitPath != "" {
		transit = vaultConfig.TransitPath
	}

	// auth path is configurable. "path", "/path", "path/" and "/path/" are the same.
	auth := defaultAuthPath
	if vaultConfig.AuthPath != "" {
		auth = vaultConfig.AuthPath
	}

	wrapper := &vaultWrapper{
		client:      client,
		encryptPath: path.Join("v1", transit, "encrypt"),
		decryptPath: path.Join("v1", transit, "decrypt"),
		authPath:    path.Join(auth),
		config:      vaultConfig,
	}

	// Set token for the vaultapi.client.
	if vaultConfig.Token != "" {
		client.SetToken(vaultConfig.Token)
	} else {
		if err := wrapper.getInitialToken(vaultConfig); err != nil {
			return nil, fmt.Errorf("unable to get initial token: %w", err)
		}
	}

	return wrapper, nil
}

func newVaultAPIClient(vaultConfig *config.VaultConfig) (*vaultapi.Client, error) {
	defaultConfig := vaultapi.DefaultConfig()
	defaultConfig.Address = vaultConfig.Address

	tlsConfig := &vaultapi.TLSConfig{
		CACert:        vaultConfig.VaultCACert,
		ClientCert:    vaultConfig.ClientCert,
		ClientKey:     vaultConfig.ClientKey,
		TLSServerName: vaultConfig.TLSServerName,
	}
	if err := defaultConfig.ConfigureTLS(tlsConfig); err != nil {
		return nil, fmt.Errorf("unable to configure TLS for %s: %w", vaultConfig.TLSServerName, err)
	}

	return vaultapi.NewClient(defaultConfig)
}

func (c *vaultWrapper) getInitialToken(vaultConfig *config.VaultConfig) error {
	switch {
	case vaultConfig.ClientCert != "" && vaultConfig.ClientKey != "":
		token, err := c.tlsToken()
		if err != nil {
			return fmt.Errorf("rotating token through TLS auth backend: %w", err)
		}

		c.client.SetToken(token)
	case vaultConfig.RoleID != "":
		token, err := c.appRoleToken(vaultConfig)
		if err != nil {
			return fmt.Errorf("rotating token through app role backend: %w", err)
		}

		c.client.SetToken(token)
	default:
		// configuration has already been validated, flow should not reach here
		return errors.New("the Vault authentication configuration is invalid")
	}

	return nil
}

func (c *vaultWrapper) tlsToken() (string, error) {
	loginPath := path.Join("/", c.authPath, "cert", "login")

	resp, err := c.client.Logical().Write(loginPath, nil)
	if err != nil {
		return "", fmt.Errorf("unable to write TLS via API on %s: %w", loginPath, err)
	} else if resp.Auth == nil {
		return "", errors.New("authentication information not found")
	}

	return resp.Auth.ClientToken, nil
}

func (c *vaultWrapper) appRoleToken(vaultConfig *config.VaultConfig) (string, error) {
	data := map[string]interface{}{
		"role_id":   vaultConfig.RoleID,
		"secret_id": vaultConfig.SecretID,
	}
	loginPath := path.Join("/", c.authPath, "approle", "login")

	resp, err := c.client.Logical().Write(loginPath, data)
	if err != nil {
		return "", fmt.Errorf("unable to write app role token via API on %s: %w", loginPath, err)
	} else if resp.Auth == nil {
		return "", errors.New("authentication information not found")
	}

	return resp.Auth.ClientToken, nil
}
func (c *vaultWrapper) Encrypt(data []byte) ([]byte, error) {
	response, err := c.withRefreshToken(true, c.config.KeyNames[0], string(data))
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt data: %w", err)
	}

	return []byte(response), nil
}
func (c *vaultWrapper) Decrypt(data []byte) ([]byte, error) {
	response, err := c.withRefreshToken(false, c.config.KeyNames[0], string(data))
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt data: %w", err)
	}

	return []byte(response), nil
}

func (c *vaultWrapper) request(requestPath string, data interface{}) (*vaultapi.Secret, error) {
	req := c.client.NewRequest("POST", "/"+requestPath)
	if err := req.SetJSONBody(data); err != nil {
		return nil, fmt.Errorf("unable to set request JSON on %s: %w", requestPath, err)
	}

	resp, err := c.client.RawRequest(req)
	if err != nil {
		if resp.StatusCode == http.StatusForbidden {
			return nil, newForbiddenError(err)
		}

		return nil, fmt.Errorf("error making POST request on %s: %w", requestPath, err)
	} else if resp == nil {
		return nil, fmt.Errorf("no response received for POST request on %s: %w", requestPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %v received for POST request to %v", resp.StatusCode, requestPath)
	}

	return vaultapi.ParseSecret(resp.Body)
}

func (c *vaultWrapper) withRefreshToken(isEncrypt bool, key, data string) (string, error) {
	// Execute operation first time.
	var (
		result string
		err    error
	)

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

	if _, ok := err.(*forbiddenError); !ok {
		return result, fmt.Errorf("error during connection: %w", err)
	}

	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()

	err = c.refreshTokenLocked(c.config)
	if err != nil {
		return result, fmt.Errorf("error refresh token request: %w", err)
	}

	klog.Infof("vault token refreshed")

	if isEncrypt {
		result, err = c.encryptLocked(key, data)
	} else {
		result, err = c.decryptLocked(key, data)
	}

	if err != nil {
		err = fmt.Errorf("error during en/de-cryption: %w", err)
	}

	return result, err
}
func (c *vaultWrapper) refreshTokenLocked(vaultConfig *config.VaultConfig) error {
	return c.getInitialToken(vaultConfig)
}

func (c *vaultWrapper) encryptLocked(key, data string) (string, error) {
	dataReq := map[string]string{"plaintext": data}

	resp, err := c.request(path.Join(c.encryptPath, key), dataReq)
	if err != nil {
		return "", fmt.Errorf("error during encrypt request: %w", err)
	}

	result, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return result, fmt.Errorf("failed type assertion of vault encrypt response type: %v to string", reflect.TypeOf(resp.Data["ciphertext"]))
	}

	return result, nil
}

func (c *vaultWrapper) decryptLocked(_, data string) (string, error) {
	dataReq := map[string]string{"ciphertext": data}

	resp, err := c.request(path.Join(c.decryptPath, c.config.KeyNames[0]), dataReq)
	if err != nil {
		return "", fmt.Errorf("error during decrypt request: %w", err)
	}

	result, ok := resp.Data["plaintext"].(string)
	if !ok {
		return result, fmt.Errorf("failed type assertion of vault decrypt response type: %v to string", reflect.TypeOf(resp.Data["plaintext"]))
	}

	return result, nil
}
