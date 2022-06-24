package awskms

import (
	"errors"
	"testing"

	"github.com/ondat/trousseau/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	testcases := map[string]struct {
		config   *Config
		hostname string
		validate func(*testing.T, providers.EncryptionClient, error)
	}{
		"OK": {
			config: &Config{
				KeyArn: "arn:aws:kms:eu-west-2:660555521754:key/0ac016bac031-a0ff-065c95b0-4ac2-91eb",
			},
			hostname: "foo.bar",
			validate: func(t *testing.T, client providers.EncryptionClient, err error) {
				t.Helper()

				require.Nil(t, err)

				awsClient, ok := client.(*awsKmsWrapper)
				assert.True(t, ok, "Failed to cast client")
				assert.Equal(t, "foo.bar", awsClient.hostname)
				assert.Equal(t, "eu-west-2", awsClient.region)
			},
		},
		"Hostname cleanup": {
			config: &Config{
				KeyArn: "arn:aws:kms:eu-west-2:660555521754:key/0ac016bac031-a0ff-065c95b0-4ac2-91eb",
			},
			hostname: "fo&o.bar_bar",
			validate: func(t *testing.T, client providers.EncryptionClient, _ error) {
				t.Helper()

				awsClient, ok := client.(*awsKmsWrapper)
				assert.True(t, ok, "Failed to cast client")
				assert.Equal(t, "foo.barbar", awsClient.hostname)
			},
		},
		"Bad ARN": {
			config: &Config{
				KeyArn: "X_arn:aws:kms:eu-west-2:660598621754:key/0ac016bac031-a0ff-065c95b0-4ac2-91eb",
			},
			validate: func(t *testing.T, client providers.EncryptionClient, err error) {
				t.Helper()

				assert.Equal(t, err, errors.New("no valid ARN found in X_arn:aws:kms:eu-west-2:660598621754:key/0ac016bac031-a0ff-065c95b0-4ac2-91eb"))
			},
		},
	}

	for name, tc := range testcases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for i := 0; i < 3; i++ {
				client, err := New(tc.config, tc.hostname)

				tc.validate(t, client, err)
			}
		})
	}
}
