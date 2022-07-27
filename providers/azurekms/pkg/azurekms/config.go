package azurekms

// Config contains the details of connection.
type Config struct {
	// ConfigFilePath Path for Azure Cloud Provider config file
	ConfigFilePath string `json:"configFilePath" yaml:"configFilePath"`
	// KeyVaultName Azure Key Vault name
	KeyVaultName string `json:"keyVaultName" yaml:"keyVaultName"`
	// KeyName Azure Key Vault KMS key name
	KeyName string `json:"keyName" yaml:"keyName"`
	// KeyVersion Azure Key Vault KMS key version
	KeyVersion string `json:"keyVersion" yaml:"keyVersion"`
	// ManagedHMS Azure Key Vault Managed HSM
	ManagedHMS bool `json:"managedHSM,omitempty" yaml:"managedHSM,omitempty"`
}
