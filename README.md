# trousseau-tsh

Requirements:

* install taskfile https://taskfile.dev/#/installation

* set env variables

```bash
VAULT_ADDR
VAULT_TOKEN
DOCKER_REGISTRY
DOCKER_USERNAME
DOCKERHUB_TOKEN
IMAGE_NAME
IMAGE_VERSION
```

Local development

* install tools

```bash
task fetch:kind - install kind binary in bin folder
```

* create local cluster

```bash
task cluster:create - create kind k8s cluster with default configuration based on Taskfile.yml vars
task cluster:create KUBERENETES_VERSION=1.21.2  - create kind k8s cluster with specified k8s version
```

DEVELOP :)

* clean environment

```bash
task cluster:delete - remove kind cluster
```

## DEMO

Vault: https://vault-storageos.aws.tshdev.io/

Deployer tools

```bash
go run cmd/deployer/main.go -h
go run cmd/deployer/main.go vault -h
```

### TOKEN
Creating config for vault token

```bash

go run cmd/deployer/main.go vault token -x demo-token
```

Generated config

```bash
task generate:manifests
cat tests/e2e/generated_manifests/config.yaml
```

Create cluster

```bash
task cluster:create
```

Display logs

```bash
docker exec -ti kms-vault-control-plane bash
```

Load examples

```bash
task example:load

example:before-key-rotate
```

Rotate key in vault

```bash
go run cmd/deployer/main.go vault rotate-key  -x demo-token

example:after-key-rotate
```

Metrics
```bash
task prometheus:deploy
task grafana:port-forward
```

Grafana dashboard - user admin password prom-operator

http://127.0.0.1:8300/

Delete cluster

```bash
task cluster:delete
```

### APP ROLE
Creating config for vault app role

```bash
go run cmd/deployer/main.go vault app-role -x demo-app-role
```

Generated config

```bash
generate:manifests
cat tests/e2e/generated_manifests/config.yaml
```

Create cluster

```bash
task cluster:create
```

Display logs

```bash
docker exec -ti kms-vault-control-plane bash
```
