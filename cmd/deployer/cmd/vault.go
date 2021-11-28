package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/Trousseau-io/trousseau/internal/config"
	"github.com/Trousseau-io/trousseau/internal/utils"
	"github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v1"
)

var (
	namesPrefix            string
	cfgConfigPath          string
	keyName                string
	policyName             string
	allowDeleteKey         bool
	transitKeysStoragePath string
)

func init() {
	vaultCmd.PersistentFlags().StringVarP(&namesPrefix, "prefix", "x", "", "prefix name")
	vaultCmd.PersistentFlags().StringVarP(&transitKeysStoragePath, "transit-keys-storage-path", "", "transit", "transit keys storage path")
	vaultCmd.PersistentFlags().StringVarP(&keyName, "key-name", "k", "test", "transit key name")
	vaultCmd.PersistentFlags().BoolVarP(&allowDeleteKey, "allow-delete_key", "m", false, "setup transit key with allow delete flag")
	vaultCmd.PersistentFlags().StringVarP(&policyName, "policy-name", "", "test", "policy key name")
	vaultCmd.PersistentFlags().StringVarP(&cfgConfigPath, "vault-cfg-path", "c", "tests/e2e/generated_manifests/config.yaml", "generate config path")
	rootCmd.AddCommand(vaultCmd)

}

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "vault",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func newVaultClient() (*api.Client, error) {
	cfg := api.DefaultConfig()
	return api.NewClient(cfg)
}

type appCfg struct {
	Provider string
	Vault    config.VaultConfig
}

func createVaultConfig(vaultCfg config.VaultConfig) {
	cfg := appCfg{Provider: "vault", Vault: vaultCfg}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	err = os.WriteFile(cfgConfigPath, data, 0600)
	if err != nil {
		log.Fatalln(err)
	}
}
func prefixedNames(prefix, name string) string {
	if prefix == "" {
		return name
	} else {
		return fmt.Sprintf("%s-%s", prefix, name)
	}
}
func prepareTransitKeyWithPolicy(cl *api.Client, policy, key string) {
	configData := map[string]interface{}{
		"deletion_allowed": allowDeleteKey,
	}
	err := utils.CreateVaultTransitKey(cl, transitKeysStoragePath, key, nil, configData)
	if err != nil {
		log.Fatalln(err)
	}
	err = utils.CreateVaultPolicy(cl, policy, key)
	if err != nil {
		log.Fatalln(err)
	}
}
