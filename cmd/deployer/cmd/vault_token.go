package cmd

import (
	"log"

	"github.com/Trousseau-io/trousseau-tsh/pkg/config"
	"github.com/Trousseau-io/trousseau-tsh/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	tokenName string
)

func init() {
	vaultTokenCmd.PersistentFlags().StringVarP(&tokenName, "token-name", "t", "test", "name token")
	vaultCmd.AddCommand(vaultTokenCmd)
}

var vaultTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Create token in vault",
	Run: func(cmd *cobra.Command, args []string) {
		token := prefixedNames(namesPrefix, tokenName)
		key := prefixedNames(namesPrefix, keyName)
		policy := prefixedNames(namesPrefix, policyName)
		cl, err := newVaultClient()
		if err != nil {
			log.Fatalln(err)
		}
		prepareTransitKeyWithPolicy(cl, policy, key)
		clientToken, err := utils.CreateVaultToken(cl, token, map[string]interface{}{
			"display_name": token,
			"policies":     policy,
		})
		if err != nil {
			log.Fatalln(err)
		}
		createVaultConfig(config.VaultConfig{
			KeyNames: []string{key},
			Address:  cl.Address(),
			Token:    clientToken,
		})
	},
}
