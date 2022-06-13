package vault

// Config contains the details of connection.
type Config struct {
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
