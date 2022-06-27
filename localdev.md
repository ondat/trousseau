# Local development

This document describes how to develop Trousseau on your local machine.

Requirements:

* install and set up Docker
* install taskfile https://taskfile.dev/#/installation

## Fetch dependencies

Trousseau development environment has some binary dependencies. To download them all please execute the task below:

```bash
task fetch:all
```

## Run Trousseau components

```bash
mkdir bin/debug
(cd proxy ; go mod tidy && go run main.go --listen-addr unix://../bin/proxy.socket --trousseau-addr ../bin/trousseau.socket)
(cd providers/debug ; go mod tidy && go run main.go --listen-addr unix://../../bin/debug/debug.socket)
(cd trousseau ; go mod tidy && go run main.go --enabled-providers debug --socket-location ../bin --listen-addr unix://../bin/trousseau.socket --zap-encoder=console --v=5)
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
kubectl get secret -o yaml
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

## Run end to end tests

Test are found in tests/e2e/kuttl directory. To run full test please execute the command below:

```bash
task go:e2e-tests
```