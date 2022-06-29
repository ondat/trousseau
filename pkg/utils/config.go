package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ondat/trousseau/pkg/logger"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// ParseConfig parses config.
func ParseConfig(cfpPath string, cfg interface{}) error {
	klog.V(logger.Info2).InfoS("Parsing config...", "path", cfpPath)

	content, err := os.ReadFile(filepath.Clean(cfpPath))
	if err != nil {
		klog.ErrorS(err, "Unable to open config file", "path", cfpPath)
		return fmt.Errorf("unable to open config file %s: %w", cfpPath, err)
	}

	if err = yaml.Unmarshal(content, cfg); err != nil {
		klog.ErrorS(err, "Unable to unmarshal config file", "path", cfpPath)
		return fmt.Errorf("unable to unmarshal config file %s: %w", cfpPath, err)
	}

	klog.V(logger.Debug2).Info("Current config", "config", cfg)

	return nil
}
