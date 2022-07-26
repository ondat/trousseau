# Local development

This document describes how to develop Trousseau Azure KMS provider on your local machine.

Please follow base documentation at [localdev.md](/../../localdev.md)

## Login to Azure

Log in and create config file (Please follow official [documentation](https://cloud-provider-azure.sigs.k8s.io/install/configs/)).

```json
{
    "cloud":"AzurePublicCloud",
    "tenantId": "...",
    "aadClientId": "...",
    "aadClientSecret": "...",
    "subscriptionId": "...",
    "resourceGroup": "...",
    "location": "...",
    "cloudProviderBackoff": false,
    "useManagedIdentityExtension": false,
    "useInstanceMetadata": true
}
```

Edit config file at [awskms.yaml](/../../tests/e2e/kuttl/kube-v1.24/azurekms.yaml):

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