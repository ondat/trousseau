package cmd

import (
	"log"

	"github.com/Trousseau-io/trousseau/internal/config"
	"github.com/Trousseau-io/trousseau/internal/utils"
	"github.com/spf13/cobra"
)

var (
	appRoleName         string
	authRoleStoragePath string
)

func init() {
	vaultAppRoleCmd.PersistentFlags().StringVarP(&appRoleName, "app-role-name", "a", "test", "name app role")
	vaultAppRoleCmd.PersistentFlags().StringVarP(&authRoleStoragePath, "app-role-path", "", "approle", "app role storage name")
	vaultCmd.AddCommand(vaultAppRoleCmd)

}

var vaultAppRoleCmd = &cobra.Command{
	Use:   "app-role",
	Short: "Create approle in vault",
	Run: func(cmd *cobra.Command, args []string) {
		role := prefixedNames(namesPrefix, appRoleName)
		key := prefixedNames(namesPrefix, keyName)
		policy := prefixedNames(namesPrefix, policyName)
		cl, err := newVaultClient()
		if err != nil {
			log.Fatalln(err)
		}
		prepareTransitKeyWithPolicy(cl, policy, key)

		credentials, err := utils.CreateVaultAppRole(cl, authRoleStoragePath, role, map[string]interface{}{
			"policies": policy,
		})
		if err != nil {
			log.Fatalln(err)
		}
		createVaultConfig(config.VaultConfig{
			KeyNames: []string{key},
			Address:  cl.Address(),
			RoleID:   credentials.RoleID,
			SecretID: credentials.SecretID,
		})
	},
}
