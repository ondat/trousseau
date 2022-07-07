# Local development

This document describes how to develop Trousseau Vault provider on your local machine.

Please follow base documentation at [localdev.md](/../../localdev.md)

## Create Vault in developer mode

To spin up a Vault localy please execute the following command:

```bash
docker run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=vault-kms-demo' -p 8200:8200 -d --name=trousseau-local-vault vault
```

You can validate your Vault instance by performing a login:

```bash
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 trousseau-local-vault vault login vault-kms-demo  
```

Enable transit engine:
```bash
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 trousseau-local-vault vault secrets enable transit
```

## Run Trousseau components

Use command line or our favorite IDE to start Trousseau components on your machine:

```bash
task go:run:proxy
task go:run:vault
ENABLED_PROVIDERS="--enabled-providers=vault" task go:run:trousseau
```
