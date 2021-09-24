package cmd

import (
	"log"

	"github.com/Trousseau-io/trousseau-tsh/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	vaultCmd.AddCommand(vaultRotateKeyCmd)

}

var vaultRotateKeyCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Roate transit key in vault",
	Run: func(cmd *cobra.Command, args []string) {
		cl, err := newVaultClient()
		if err != nil {
			log.Fatalln(err)
		}
		key := prefixedNames(namesPrefix, keyName)
		err = utils.RotateVaultTransitKey(cl, transitKeysStoragePath, key, nil, nil)
		if err != nil {
			log.Fatalln(err)
		}
	},
}
