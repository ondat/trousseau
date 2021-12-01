package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	// BuildDate is the date when the binary was built
	BuildDate string
	// GitCommit is the commit hash when the binary was built
	GitCommit string
	// BuildVersion is the version of the KMS binary
	BuildVersion = "dev"
	APIVersion   = "v1beta1"
	Runtime      = "HashiCorp Vault KMS"
)

// PrintVersion prints the current KMS plugin version
func PrintVersion() (err error) {
	pv := struct {
		BuildVersion string
		GitCommit    string
		BuildDate    string
	}{
		BuildDate:    BuildDate,
		BuildVersion: BuildVersion,
		GitCommit:    GitCommit,
	}

	var res []byte
	if res, err = json.Marshal(pv); err != nil {
		return
	}

	fmt.Printf(string(res) + "\n")
	return
}

// GetUserAgent returns UserAgent string to append to the agent identifier.
func GetUserAgent() string {
	return fmt.Sprintf("k8s-kms-vault/%s (%s/%s) %s/%s", BuildVersion, runtime.GOOS, runtime.GOARCH, GitCommit, BuildDate)
}
