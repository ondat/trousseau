# Local development

This document describes how to develop Trousseau AWS KMS provider on your local machine.

Please follow base documentation at [localdev.md](../localdev.md)

## Create AWS KMS config

Edit config file at [awskms.yaml](../scripts/hcvault/archives/localdev/awskms.yaml):

```yaml
profile: profile
keyArn: keyArn
roleArn: roleArn
```

## Run Trousseau components

Use command line or our favorite IDE to start Trousseau components on your machine:

```bash
mkdir bin/awskms
(cd proxy ; go mod tidy && go run main.go --listen-addr unix://../bin/proxy.socket --trousseau-addr ../bin/trousseau.socket)
(cd providers/awskms ; go mod tidy && go run main.go --config-file-path ../../scripts/hcvault/archives/localdev/awskms.yaml --listen-addr unix://../../bin/awskms/awskms.socket --zap-encoder=console --v=5)
(cd trousseau ; go mod tidy && go run main.go --enabled-providers awskms --socket-location ../bin --listen-addr unix://../bin/trousseau.socket --zap-encoder=console --v=5)
```
