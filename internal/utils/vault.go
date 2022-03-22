package utils

import (
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

type AppRoleCredentials struct {
	SecretID string
	RoleID   string
}

func CreateVaultTransitKey(cli *api.Client, prefix, name string, params map[string]interface{}, configParams map[string]interface{}) error {
	path := fmt.Sprintf("%s/keys/%s", prefix, name)
	_, err := cli.Logical().Write(path, params)
	if err != nil {
		return errors.WithMessagef(err, "unable to create params for %s", path)
	}
	if configParams != nil {
		path := fmt.Sprintf("transit/keys/%s/config", name)
		_, err := cli.Logical().Write(path, configParams)
		if err != nil {
			return errors.WithMessagef(err, "unable to create config params for %s", path)
		}
	}
	return nil
}

func RotateVaultTransitKey(cli *api.Client, prefix, name string, params map[string]interface{}, configParams map[string]interface{}) error {
	path := fmt.Sprintf("%s/keys/%s/rotate", prefix, name)
	_, err := cli.Logical().Write(path, params)
	if err != nil {
		return errors.WithMessagef(err, "unable to rotate params for %s", path)
	}
	return nil
}

func CreateVaultAppRole(cli *api.Client, prefix, name string, params map[string]interface{}) (*AppRoleCredentials, error) {
	path := fmt.Sprintf("auth/%s/role/%s", prefix, name)
	_, err := cli.Logical().Write(path, params)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to create role for %s", path)
	}
	roleSecret, err := cli.Logical().Read(path + "/role-id")
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to read role for %s", path)
	}
	SecretIDSecret, err := cli.Logical().Write(path+"/secret-id", nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to read secret for %s", path)
	}
	return &AppRoleCredentials{
		RoleID:   roleSecret.Data["role_id"].(string),
		SecretID: SecretIDSecret.Data["secret_id"].(string),
	}, nil
}

func CreateVaultPolicy(api *api.Client, policyName string, keyName string) error {
	policy := fmt.Sprintf(`
	path "transit/encrypt/%s" {
		capabilities = ["update"]
	}
	path "transit/decrypt/%s" {
		capabilities = ["update"]
	}
	`, keyName, keyName)
	path := fmt.Sprintf("sys/policy/%s", policyName)
	_, err := api.Logical().Write(path, map[string]interface{}{
		"policy": policy,
	})
	if err != nil {
		return errors.WithMessagef(err, "unable to create policy for %s", path)
	}
	return nil
}

func CreateVaultToken(cli *api.Client, name string, params map[string]interface{}) (string, error) {
	path := "/auth/token/create"
	r, err := cli.Logical().Write(path, params)
	if err != nil {
		return "", errors.WithMessagef(err, "unable to create vault token for %s", path)
	}
	return r.Auth.ClientToken, nil
}
