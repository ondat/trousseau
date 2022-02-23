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
		return err
	}
	if configParams != nil {
		path := fmt.Sprintf("transit/keys/%s/config", name)
		_, err := cli.Logical().Write(path, configParams)
		if err != nil {
			return err
		}
	}
	return nil
}

func RotateVaultTransitKey(cli *api.Client, prefix, name string, params map[string]interface{}, configParams map[string]interface{}) error {
	path := fmt.Sprintf("%s/keys/%s/rotate", prefix, name)
	_, err := cli.Logical().Write(path, params)
	if err != nil {
		return err
	}
	return nil
}

func CreateVaultAppRole(cli *api.Client, prefix, name string, params map[string]interface{}) (*AppRoleCredentials, error) {
	path := fmt.Sprintf("auth/%s/role/%s", prefix, name)
	_, err := cli.Logical().Write(path, params)
	if err != nil {
		return nil, err
	}
	roleSecret, err := cli.Logical().Read(path + "/role-id")
	if err != nil {
		return nil, err
	}
	SecretIDSecret, err := cli.Logical().Write(path+"/secret-id", nil)
	if err != nil {
		return nil, err
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
	_, err := api.Logical().Write(fmt.Sprintf("sys/policy/%s", policyName), map[string]interface{}{
		"policy": policy,
	})
	if err != nil {
		return err
	}
	return nil
}

func CreateVaultToken(cli *api.Client, name string, params map[string]interface{}) (string, error) {
	r, err := cli.Logical().Write("/auth/token/create", params)
	if err != nil {
		return "", err
	}
	return r.Auth.ClientToken, nil
}
