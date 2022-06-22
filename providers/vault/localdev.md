# Local development

This document describes how to develop Trousseau Vault provider on your local machine.

Please follow base documentation at [localdev.md](../localdev.md)

Requirements:

* `vault.loc` hostname needs to be resolved to your local machine, or alternatively tou have to change `scripts/hcvault/archives/localdev/config.yaml` to point to a working Vault instance

## Create Vault in developer mode

To spin up a Vault localy please execute the following command:

```bash
docker run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=vault-kms-demo' -p 8200:8200 -d --name=dev-vault vault
```

You can validate your Vault instance by performing a login:

```bash
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -it dev-vault vault login vault-kms-demo  
```

Enable transit engine:
```bash
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -it dev-vault vault secrets enable transit
```

## Run Trousseau components

Use command line or our favorite IDE to start Trousseau components on your machine:

```bash
mkdir bin/vault
(cd proxy ; go mod tidy && go run main.go --listen-addr unix://../bin/proxy.socket --trousseau-addr ../bin/trousseau.socket)
(cd providers/vault ; go mod tidy && go run main.go --config-file-path ../../scripts/hcvault/archives/localdev/vault.yaml --listen-addr unix://../../bin/vault/vault.socket --zap-encoder=console --v=5)
(cd trousseau ; go mod tidy && go run main.go --enabled-providers vault --socket-location ../bin --listen-addr unix://../bin/trousseau.socket --zap-encoder=console --v=5)
```
