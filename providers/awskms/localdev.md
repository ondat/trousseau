# Local development

This document describes how to develop Trousseau AWS KMS provider on your local machine.

Please follow base documentation at [localdev.md](../localdev.md)

## Login to AWS

Log in and create profile file at `~/.aws/credentials`.

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
task go:run:proxy
task go:run:awskms
ENABLED_PROVIDERS="--enabled-providers awskms" task go:run:trousseau
```
