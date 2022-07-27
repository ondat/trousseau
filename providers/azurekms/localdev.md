# Local development

This document describes how to develop Trousseau Azure KMS provider on your local machine.

Please follow base documentation at [localdev.md](/../../localdev.md)

## Login to Azure

Log in and create config file at [azurekms.json](/../../tests/e2e/kuttl/kube-v1.24/azurekms.json).

```json
{
    "cloud":"AzurePublicCloud",
    "tenantId": "...",
    "aadClientId": "...",
    "aadClientSecret": "...",
    "subscriptionId": "..."
}
```

Edit config file at [azurekms.yaml](/../../tests/e2e/kuttl/kube-v1.24/azurekms.yaml):

```yaml
configFilePath: configFilePath
keyVaultName: keyVaultName
keyName: keyName
keyVersion: keyVersion
```

## Run Trousseau components

Use command line or our favorite IDE to start Trousseau components on your machine:

```bash
task go:run:proxy
task go:run:azurekms
ENABLED_PROVIDERS="--enabled-providers=azurekms" task go:run:trousseau
```