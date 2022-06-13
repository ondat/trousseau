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

## Start cluster with encryption support

For local testing we suggest to use Kind to create a cluster. Everything is configured for you so please run the command below:

```bash
task cluster:create SCRIPT=scripts/hcvault/archives/localdev
```

You are ready for create secrets!

### Verify secret encryption

To verify encryption please create a secret and check value in ETCD.

```
kubectl create secret -n default generic trousseau-test --from-literal=FOO=bar
docker exec kms-vault-control-plane bash -c 'apt update && apt install -y etcd-client' # only once
docker exec -it -e ETCDCTL_API=3 -e SSL_OPTS='--cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/apiserver-etcd-client.crt --key=/etc/kubernetes/pki/apiserver-etcd-client.key --endpoints=localhost:2379' kms-vault-control-plane \
bash -c 'etcdctl $SSL_OPTS get --keys-only=false --prefix /registry/secrets/default'
```

You have to see encrypted data in ETCD dump.

### Cleanup cluster

After you have finished fun on Trousseau you should terminate the cluster with the following command:

```bash
task cluster:delete
```

