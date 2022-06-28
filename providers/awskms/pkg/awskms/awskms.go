package awskms

import (
	"encoding/base64"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/ondat/trousseau/pkg/logger"
	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/pkg/version"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

const maxRoleSessionNameLength = 64

var (
	keyArnRegexp          = regexp.MustCompile(`^arn:aws[\w-]*:kms:(.+):\d+:(key|alias)/.+$`)
	hostnameCleanupRegexp = regexp.MustCompile("[^a-zA-Z0-9=,.@-]+")
)

// Handle all communication with AWS KMS server.
type awsKmsWrapper struct {
	config      *Config
	sessionOpts session.Options
	hostname    string
	region      string
}

// New creates an instance of the KMS client.
func New(config *Config, hostname string) (providers.EncryptionClient, error) {
	matches := keyArnRegexp.FindStringSubmatch(config.KeyArn)
	if matches == nil {
		klog.Error("No valid ARN found")
		return nil, fmt.Errorf("no valid ARN found in %s", config.KeyArn)
	}

	opts := session.Options{
		Profile: config.Profile,
		Config: aws.Config{
			Region:                        aws.String(matches[1]),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
		SharedConfigState: session.SharedConfigEnable,
	}

	if config.Endpoint != "" {
		var resolver endpoints.ResolverFunc = func(_, _ string, _ ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
			return endpoints.ResolvedEndpoint{
				URL: config.Endpoint,
			}, nil
		}

		opts.Config.Endpoint = aws.String(config.Endpoint)
		opts.Config.EndpointResolver = resolver
		opts.Config.DisableEndpointHostPrefix = aws.Bool(true)
	}

	if config.AssumeRoleMFAToken != "" {
		opts.AssumeRoleTokenProvider = func() (string, error) {
			return config.AssumeRoleMFAToken, nil
		}
	}

	return &awsKmsWrapper{
		config:      config,
		sessionOpts: opts,
		hostname:    hostnameCleanupRegexp.ReplaceAllString(hostname, ""),
		region:      matches[1],
	}, nil
}

// Encrypt encrypts input.
func (c *awsKmsWrapper) Encrypt(data []byte) ([]byte, error) {
	klog.V(logger.Info3).InfoS("Encrypting...")

	sess, err := c.createSession()
	if err != nil {
		klog.InfoS("Unable to create session", "error", err.Error())
		return nil, fmt.Errorf("unable to create session: %w", err)
	}

	klog.V(logger.Debug2).InfoS("Encrypting data", "data", utils.SecretToLog(string(data)))

	response, err := kms.New(sess, &c.sessionOpts.Config).Encrypt(&kms.EncryptInput{Plaintext: data, KeyId: &c.config.KeyArn, EncryptionContext: c.config.EncryptionContext})
	if err != nil {
		klog.InfoS("Unable to encrypt data", "error", err.Error())
		return nil, fmt.Errorf("unable to encrypt data: %w", err)
	}

	klog.V(logger.Debug2).InfoS("Encrypted data", "data", utils.SecretToLog(string(response.CiphertextBlob)))

	return []byte(base64.StdEncoding.EncodeToString(response.CiphertextBlob)), nil
}

// Decrypt decrypts input.
func (c *awsKmsWrapper) Decrypt(data []byte) ([]byte, error) {
	klog.V(logger.Info3).InfoS("Decrypting...")

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		klog.InfoS("Failed decode encrypted data", "error", err.Error())
		return nil, fmt.Errorf("failed decode encrypted data: %w", err)
	}

	sess, err := c.createSession()
	if err != nil {
		klog.InfoS("Unable to create session", "error", err.Error())
		return nil, fmt.Errorf("unable to create session: %w", err)
	}

	klog.V(logger.Debug2).InfoS("Decrypting data", "data", utils.SecretToLog(string(data)))

	response, err := kms.New(sess, &c.sessionOpts.Config).Decrypt(&kms.DecryptInput{CiphertextBlob: decoded, EncryptionContext: c.config.EncryptionContext})
	if err != nil {
		klog.InfoS("Unable to decrypt data", "error", err.Error())
		return nil, fmt.Errorf("unable to decrypt data: %w", err)
	}

	klog.V(logger.Debug2).InfoS("Decrypted data", "data", utils.SecretToLog(string(response.Plaintext)))

	return response.Plaintext, nil
}

func (c *awsKmsWrapper) Version() *pb.VersionResponse {
	return &pb.VersionResponse{Version: version.APIVersion, RuntimeName: version.Runtime, RuntimeVersion: version.BuildVersion}
}

func (c *awsKmsWrapper) createSession() (*session.Session, error) {
	klog.V(logger.Info3).InfoS("Creating new session...")
	klog.V(logger.Debug1).InfoS("Creating new session", "options", c.sessionOpts)

	sess, err := session.NewSessionWithOptions(c.sessionOpts)
	if err != nil {
		klog.InfoS("Unable to create new session", "error", err.Error())
		return nil, fmt.Errorf("unable to create new session: %w", err)
	}

	if c.config.RoleArn != "" {
		return c.createStsSession(sess)
	}

	return sess, nil
}

func (c *awsKmsWrapper) createStsSession(sess *session.Session) (*session.Session, error) {
	hostname := c.hostname
	if len(hostname) >= maxRoleSessionNameLength {
		hostname = hostname[:maxRoleSessionNameLength]
	}

	klog.V(logger.Info3).InfoS("Creating new STS session...", "hostname", hostname)

	stsService := sts.New(sess)

	out, err := stsService.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         &c.config.RoleArn,
		RoleSessionName: &hostname,
	})
	if err != nil {
		klog.InfoS("Unable to assume role", "error", err.Error())
		return nil, fmt.Errorf("unable to assume role: %w", err)
	}

	sess, err = session.NewSession(&c.sessionOpts.Config, &aws.Config{Credentials: credentials.NewStaticCredentials(*out.Credentials.AccessKeyId, *out.Credentials.SecretAccessKey, *out.Credentials.SessionToken)})
	if err != nil {
		klog.InfoS("Unable to create new session", "error", err.Error())
		return nil, fmt.Errorf("unable to create new session: %w", err)
	}

	return sess, nil
}
