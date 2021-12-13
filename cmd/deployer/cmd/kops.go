package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v1"
)

var kopsCfgFile string
var vaultCfgFile string
var kubeManifestCfgFile string
var deploymentManifestCfgFile string

func init() {
	kopsCmd.Flags().StringVarP(&kopsCfgFile, "kops-cluster-file", "", "./scripts/kops/cluster.yaml", "generate config path")
	kopsCmd.Flags().StringVarP(&vaultCfgFile, "vault-cluster-file", "", "./tests/e2e/generated_manifests/config.yaml", "generate config path")
	kopsCmd.Flags().StringVarP(&kubeManifestCfgFile, "encryption-config-file", "", "./scripts/encryption-config.yaml", "generate config path")
	kopsCmd.Flags().StringVarP(&deploymentManifestCfgFile, "deployment-config-file", "", "./tests/e2e/generated_manifests/kms.yaml", "generate config path")
	rootCmd.AddCommand(kopsCmd)

}

var kopsCmd = &cobra.Command{
	Use:   "generate-kops-config",
	Short: "generate kops config with vault-kms-provider support",
	Run: func(cmd *cobra.Command, args []string) {

		repoFile := kopsCfgFile
		repoFile = filepath.Clean(repoFile)
		if !strings.HasPrefix(repoFile, "./scripts/"){
			panic(fmt.Errorf("Unsafe input! - Use a the local subdirectory scripts to host your files"))
		}
		byContext, err := ioutil.ReadFile(repoFile)
		if err != nil {
			panic(err)
		}
		out, err := generateConfig(byContext)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(out))

	},
}

func generateConfig(cfg []byte) ([]byte, error) {
	var data map[string]interface{}
	err := yaml.Unmarshal(cfg, &data)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w ", err)
	}
	var s map[string]interface{}

	if err := mergo.Merge(&data, s); err != nil {
		return nil, fmt.Errorf("error during merge configs: %w ", err)
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error during marshal merged config: %w ", err)
	}
	return out, nil
}
