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
	"github.com/ondat/trousseau/internal/logger"
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
	klog.V(logger.Debug1).Info("Initialize client wrapper...")

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
		klog.V(logger.Debug2).InfoS("Set token", "token", secretToLog(vaultConfig.Token))

		client.SetToken(vaultConfig.Token)
	} else {
		klog.V(logger.Debug2).InfoS("Get initial token...", "transit", transit, "auth", auth)

		if err := wrapper.getInitialToken(vaultConfig); err != nil {
			klog.ErrorS(err, "Unable to get initial token", "transit", transit, "auth", auth)
			return nil, fmt.Errorf("unable to get initial token: %w", err)
		}
	}

	return wrapper, nil
}

func newVaultAPIClient(vaultConfig *config.VaultConfig) (*vaultapi.Client, error) {
	klog.V(logger.Debug1).Info("Configuring TLS...")

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

	klog.V(logger.Debug2).InfoS("Initialize API client...", "config", *defaultConfig)

	return vaultapi.NewClient(defaultConfig)
}

func (c *vaultWrapper) getInitialToken(vaultConfig *config.VaultConfig) error {
	switch {
	case vaultConfig.ClientCert != "" && vaultConfig.ClientKey != "":
		klog.V(logger.Debug2).InfoS("Get initial token by", "cert", vaultConfig.ClientCert, "key", vaultConfig.ClientKey)

		token, err := c.tlsToken()
		if err != nil {
			return fmt.Errorf("rotating token through TLS auth backend: %w", err)
		}

		c.client.SetToken(token)
	case vaultConfig.RoleID != "":
		klog.V(logger.Debug2).InfoS("Get initial token by", "role", vaultConfig.RoleID)

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

	klog.V(logger.Debug1).InfoS("Get TLS token...", "path", loginPath)

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

	klog.V(logger.Debug1).InfoS("Get role token...", "path", loginPath, "data", data)

	resp, err := c.client.Logical().Write(loginPath, data)
	if err != nil {
		return "", fmt.Errorf("unable to write app role token via API on %s: %w", loginPath, err)
	} else if resp.Auth == nil {
		return "", errors.New("authentication information not found")
	}

	return resp.Auth.ClientToken, nil
}
func (c *vaultWrapper) Encrypt(data []byte) ([]byte, error) {
	klog.V(logger.Debug2).InfoS("Encrypting data...", "key", c.config.KeyNames[0], "data", secretToLog(string(data)))

	response, err := c.withRefreshToken(true, c.config.KeyNames[0], string(data))
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt data: %w", err)
	}

	klog.V(logger.Debug2).InfoS("Encrypted data", "key", c.config.KeyNames[0])

	return []byte(response), nil
}
func (c *vaultWrapper) Decrypt(data []byte) ([]byte, error) {
	klog.V(logger.Debug2).InfoS("Decrypting data...", "key", c.config.KeyNames[0])

	response, err := c.withRefreshToken(false, c.config.KeyNames[0], string(data))
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt data: %w", err)
	}

	klog.V(logger.Debug2).InfoS("decrypted data", "key", c.config.KeyNames[0], "data", secretToLog(response))

	return []byte(response), nil
}

func (c *vaultWrapper) request(requestPath string, data interface{}) (*vaultapi.Secret, error) {
	klog.V(logger.Debug1).InfoS("Creating request...", "path", requestPath)

	req := c.client.NewRequest("POST", "/"+requestPath)
	if err := req.SetJSONBody(data); err != nil {
		return nil, fmt.Errorf("unable to set request JSON on %s: %w", requestPath, err)
	}

	resp, err := c.client.RawRequest(req)
	if err != nil {
		code := -1
		if resp != nil {
			code = resp.StatusCode
		}

		klog.InfoS("Failed to send request", "code", code, "error", err.Error())

		if code == http.StatusForbidden {
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

	klog.V(logger.Debug2).Info("Parsing secret...")

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
		return result, fmt.Errorf("error during connection for %s: %w", key, err)
	}

	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()

	klog.V(logger.Debug1).Info("Refreshing token...")

	if err = c.refreshTokenLocked(c.config); err != nil {
		klog.Error(err, "Failed to refresh token")
		return result, fmt.Errorf("error refresh token request: %w", err)
	}

	klog.V(logger.Info1).Info("Vault token refreshed")

	if isEncrypt {
		result, err = c.encryptLocked(key, data)
	} else {
		result, err = c.decryptLocked(key, data)
	}

	if err != nil {
		klog.InfoS("Error during en/de-cryption", "isEncrypt", isEncrypt, "key", key)
		err = fmt.Errorf("error during en/de-cryption for %s: %w", key, err)
	}

	return result, err
}
func (c *vaultWrapper) refreshTokenLocked(vaultConfig *config.VaultConfig) error {
	return c.getInitialToken(vaultConfig)
}

func (c *vaultWrapper) encryptLocked(key, data string) (string, error) {
	klog.V(logger.Debug2).Info("Encrypting locked...", "key", key)

	dataReq := map[string]string{"plaintext": data}

	resp, err := c.request(path.Join(c.encryptPath, key), dataReq)
	if err != nil {
		klog.InfoS("Failed to encrypt locked", "key", key, "error", err.Error())
		return "", fmt.Errorf("error during encrypt request for %s: %w", key, err)
	}

	result, ok := resp.Data["ciphertext"].(string)
	if !ok {
		klog.InfoS("Failed to find ciphertext", "key", key)
		return result, fmt.Errorf("failed type assertion of vault encrypt response type for %s: %v to string", key, reflect.TypeOf(resp.Data["ciphertext"]))
	}

	return result, nil
}

func (c *vaultWrapper) decryptLocked(_, data string) (string, error) {
	klog.V(logger.Debug2).Info("Decrypting locked...")

	dataReq := map[string]string{"ciphertext": data}

	resp, err := c.request(path.Join(c.decryptPath, c.config.KeyNames[0]), dataReq)
	if err != nil {
		klog.InfoS("Failed to decrypt locked", "error", err.Error())
		return "", fmt.Errorf("error during decrypt request: %w", err)
	}

	result, ok := resp.Data["plaintext"].(string)
	if !ok {
		klog.InfoS("Failed to find plaintext representation")
		return result, fmt.Errorf("failed type assertion of vault decrypt response type: %v to string", reflect.TypeOf(resp.Data["plaintext"]))
	}

	return result, nil
}

func secretToLog(s string) string {
	b := []byte(s)
	return string(b[:len(b)/2]) + "..."
}
