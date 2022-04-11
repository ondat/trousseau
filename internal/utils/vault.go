package utils

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

type AppRoleCredentials struct {
	SecretID string
	RoleID   string
}

func CreateVaultTransitKey(cli *api.Client, prefix, name string, params map[string]interface{}, configParams map[string]interface{}) error {
	path := fmt.Sprintf("%s/keys/%s", prefix, name)
	_, err := cli.Logical().Write(path, params)

	if err != nil {
		return fmt.Errorf("unable to create params for %s: %w", path, err)
	}

	if configParams != nil {
		path := fmt.Sprintf("transit/keys/%s/config", name)
		_, err := cli.Logical().Write(path, configParams)

		if err != nil {
			return fmt.Errorf("unable to create config params for %s: %w", path, err)
		}
	}

	return nil
}

func RotateVaultTransitKey(cli *api.Client, prefix, name string, params map[string]interface{}, configParams map[string]interface{}) error {
	path := fmt.Sprintf("%s/keys/%s/rotate", prefix, name)
	_, err := cli.Logical().Write(path, params)

	if err != nil {
		return fmt.Errorf("unable to rotate params for %s: %w", path, err)
	}

	return nil
}

func CreateVaultAppRole(cli *api.Client, prefix, name string, params map[string]interface{}) (*AppRoleCredentials, error) {
	path := fmt.Sprintf("auth/%s/role/%s", prefix, name)
	_, err := cli.Logical().Write(path, params)

	if err != nil {
		return nil, fmt.Errorf("unable to create role for %s: %w", path, err)
	}

	roleSecret, err := cli.Logical().Read(path + "/role-id")

	if err != nil {
		return nil, fmt.Errorf("unable to read role for %s: %w", path, err)
	}

	SecretIDSecret, err := cli.Logical().Write(path+"/secret-id", nil)

	if err != nil {
		return nil, fmt.Errorf("unable to read secret for %s: %w", path, err)
	}

	return &AppRoleCredentials{
		RoleID:   roleSecret.Data["role_id"].(string),
		SecretID: SecretIDSecret.Data["secret_id"].(string),
	}, nil
}

func CreateVaultPolicy(cli *api.Client, policyName string, keyName string) error {
	policy := fmt.Sprintf(`
	path "transit/encrypt/%s" {
		capabilities = ["update"]
	}
	path "transit/decrypt/%s" {
		capabilities = ["update"]
	}
	`, keyName, keyName)
	path := fmt.Sprintf("sys/policy/%s", policyName)
	_, err := cli.Logical().Write(path, map[string]interface{}{
		"policy": policy,
	})

	if err != nil {
		return fmt.Errorf("unable to create policy for %s: %w", path, err)
	}

	return nil
}

func CreateVaultToken(cli *api.Client, name string, params map[string]interface{}) (string, error) {
	path := "/auth/token/create"
	r, err := cli.Logical().Write(path, params)

	if err != nil {
		return "", fmt.Errorf("unable to create vault token for %s: %w", path, err)
	}

	return r.Auth.ClientToken, nil
}
