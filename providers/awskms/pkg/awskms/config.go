package awskms

// Config contains the details of connection.
type Config struct {
	// Endpoint service endpoint, optional
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	// Name of profile
	Profile string `json:"profile" yaml:"profile"`
	// Arn of key
	KeyArn string `json:"keyArn" yaml:"keyArn"`
	// Arn of role
	RoleArn string `json:"roleArn" yaml:"roleArn"`
	// Context of encryption, optional
	EncryptionContext map[string]*string `json:"encryptionContext,omitempty" yaml:"encryptionContext,omitempty"`
}
