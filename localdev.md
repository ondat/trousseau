# Local development

This document describes how to develop Trousseau on your local machine.

Requirements:

* install and set up Docker
* install taskfile https://taskfile.dev/#/installation
* `vault.loc` hostname needs to be resolved to your local machine, or alternatively tou have to change `scripts/hcvault/archives/localdev/config.yaml` to point to a working Vault instance

## Fetch dependencies

Trousseau development environment has some binary dependencies. To download them all please execute the task below:

```bash
task fetch:all
```

## Create Vault in developer mode

To spin up a Vault localy please execute the following command:

```bash
docker run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=vault-kms-demo' -p 8200:8200 -d --name=dev-vault vault
```

You can validate your Vault instance by performing a login:

```bash
docker exec -it dev-vault vault login -address=http://localhost:8200      
Token (will be hidden): vault-kms-demo
```

## Run Trousseau

Use command line or our favorite IDE to start Trousseau on your machine:

```bash
go run cmd/kubernetes-kms-vault/main.go --config-file-path scripts/hcvault/archives/localdev/config.yaml --listen-addr unix://vaultkms.socket --log-format-json=false
```

## Start cluster with encryption support

For local testing we suggest to use Kind to create a cluster. Everything is configured for you so please run the command below:

```bash
task cluster:create SCRIPT=scripts/hcvault/archives/localdev
```

You are ready for create secrets!

### Cleanup cluster

After you have finished fun on Trousseau you should terminate the cluster with the following command:

```bash
task cluster:delete
```
