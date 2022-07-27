<!-- 
<p align="center">
    <img src="https://github.com/ondat/trousseau/blob/main/assets/logo-horizontal.png" >
</p> -->

<h1 align="center">
  <br>
  <a href="https://github.com/ondat/trousseau/blob/main/assets/logo-horizontal.png"><img src="https://github.com/ondat/trousseau/blob/main/assets/logo-horizontal.png" alt="Trousseau" ></a>
  <br>
</h1>

<h4 align="center">A multi KMS solution supporting the <a href="https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/" target="_blank">Kubernetes Provider Plugin</a> data encryption for Secrets and ConfigMap in etcd.</h4>


<p align="center">
    <a href="https://goreportcard.com/report/github.com/ondat/trousseau">
        <img src="https://goreportcard.com/badge/github.com/ondat/trousseau" /></a>
    <a href="https://lgtm.com/projects/g/ondat/trousseau/alerts/">
        <img alt="Total alerts" src="https://img.shields.io/lgtm/alerts/g/ondat/trousseau.svg?logo=lgtm&logoWidth=18"/></a>
    <a href="https://github.com/ondat/trousseau/actions/workflows/e2e-on-pr.yml" alt="end-2-end build">
        <img src="https://github.com/ondat/trousseau/actions/workflows/e2e-on-pr.yml/badge.svg" /></a>
    <a href="https://bestpractices.coreinfrastructure.org/projects/5460" alt="CII Best Practices">
        <img src="https://bestpractices.coreinfrastructure.org/projects/5460/badge" /></a>
    <a href="https://github.com/ondat/trousseau/pkgs/container/trousseau" alt="pulled images">
        <img src="https://img.shields.io/badge/pulled%20images-15.2k-brightgreen" /></a>
</p>

<p align="center">
  <a href="#key-features">Key Features</a> •
  <a href="https://github.com/ondat/trousseau/wiki">Why</a> •
  <a href="https://github.com/ondat/trousseau/wiki/Trousseau-Deployment">Documentation</a> •
  <a href="https://github.com/ondat/trousseau/wiki/Press">Press</a> •
  <a href="https://www.ondat.io/trousseau">Hands-on Lab</a> •
  <a href="#how-to-test">How to test</a> •
  <a href="https://github.com/orgs/ondat/projects/3">Roadmap</a> •
  <a href="https://github.com/ondat/trousseau/blob/main/CONTRIBUTING.md">Contributing</a> •
  <a href="https://github.com/ondat/trousseau/blob/main/LICENSE">License</a> •
  <a href="https://github.com/ondat/trousseau/blob/main/SECURITY.md">Security</a>
</p>

<p align="center">
    <img src="https://github.com/ondat/trousseau/blob/main/assets/Ondat%20Diagram-w-all.png" height="400">
</p>

## Key Features

* Kubernetes native - no additional CLI tooling, respectful of the concern APIs (like Secrets, ConfigMap, ...)
* Encryption of sensitive data payload on the fly and store in *etcd* 
* Multi KMS support - one KMS or two KMS at the same time[1]
  * HashiCorp Vault (Community and Enterprise editions)
  * AWS Key Vault
  * Azure KeyVault 
* Redesign to full micro-service architecture decloupling the core components for maximum resiliency and distributed handling
  * proxy socket to address the Kubernetes API request for encryption/decryption
  * trousseau to handle the proxy requests and KMS interaction
  * KMS socket to address the connection towards the KMS providers 
* Prometheus endpoint 

Notes: 

1. Trousseau will use each KMS provider to encrypt the data and combine both payload within the same secret data section. 
   This design is provide more resiliency in case of a KMS failure by introducing reduancy, and add a fast decryption appraoch with first to response decryption approach along with roundrobin.   
   At the current stade, there is no option to have multi KMS configured and targeting one specific entry for scenario like multi-tenancy and/or multi-staging environment. This is due to a missing annotation extension within the Kubernetes API that we have address to the Kubernetes project.(see issue [#146](https://github.com/ondat/trousseau/issues/146)) 

## How to test

⚠️ for production deployment, consult the [Documentation](https://github.com/ondat/trousseau/wiki)

Clone the repo and create your environment file:
```bash
TR_VERSION=31b93747fc4fd438a6b30de70ff16d4a45271366
TR_VERBOSE_LEVEL=1
TR_SOCKET_LOCATION=/opt/trousseau-kms
TR_PROXY_IMAGE=ghcr.io/ondat/trousseau:proxy-${TR_VERSION}
TR_TROUSSEAU_IMAGE=ghcr.io/ondat/trousseau:trousseau-${TR_VERSION}
# Please configure your KMS plugins, maximum 2
TR_ENABLED_PROVIDERS="--enabled-providers=awskms --enabled-providers=azurekms --enabled-providers=vault"
TR_AWSKMS_IMAGE=ghcr.io/ondat/trousseau:awskms-${TR_VERSION}
TR_AWSKMS_CONFIG=awskms.yaml # For Kubernetes, file must exists only for generation
TR_AWSKMS_CREDENTIALS=.aws/credentials
TR_AZUREKMS_IMAGE=ghcr.io/ondat/trousseau:azurekms-${TR_VERSION}
TR_AZUREKMS_CONFIG=azurekms.yaml # For Kubernetes, file must exists only for generation
TR_AZUREKMS_CREDENTIALS=config.json
TR_VAULT_IMAGE=ghcr.io/ondat/trousseau:vault-${TR_VERSION}
TR_VAULT_ADDRESS=https://127.0.0.1:8200
TR_VAULT_CONFIG=vault.yaml
```

Create shared items on target host:
```bash
mkdir -p $TR_SOCKET_LOCATION
sudo chown 10123:10123 $TR_SOCKET_LOCATION
sudo chown 10123:10123 $TR_AWSKMS_CREDENTIALS
# On case you haven't enable Vault agen config generation
sudo chown 10123:10123 $TR_VAULT_CONFIG
```

Create your config files:
```yaml
# awskms.yaml
profile: profile
keyArn: keyArn
# Optional fields
roleArn: roleArn
encryptionContext:
  foo: bar
```
```yaml
# azurekms.yaml
configFilePath: configFilePath
keyVaultName: keyVaultName
keyName: keyName
keyVersion: keyVersion
```
```yaml
# vault.yaml
keyNames:
-  keyNames
address: address
token: token
```

Generate service files or manifests:
```bash
make prod:generate:systemd ENV_LOCATION=./bin/trousseau-env
make prod:generate:docker-compose ENV_LOCATION=./bin/trousseau-env
make prod:generate:kustomize ENV_LOCATION=./bin/trousseau-env
make prod:generate:helm ENV_LOCATION=./bin/trousseau-env
```

Verify output:
```bash
ls -l generated_manifests/systemd
ls -l generated_manifests/docker-compose
ls -l generated_manifests/kustomize
ls -l generated_manifests/helm
```

Deploy the application and configure encryption:
```
kind: EncryptionConfiguration
apiVersion: apiserver.config.k8s.io/v1
resources:
  - resources:
      - secrets
    providers:
      - kms:
          name: vaultprovider
          endpoint: unix:///opt/trousseau-kms/proxy.socket
          cachesize: 1000
      - identity: {}
```

Reconfigure Kubernetes API server:
```
kind: ClusterConfiguration
apiServer:
  extraArgs:
    encryption-provider-config: "/etc/kubernetes/encryption-config.yaml"
  extraVolumes:
  - name: encryption-config
    hostPath: "/etc/kubernetes/encryption-config.yaml"
    mountPath: "/etc/kubernetes/encryption-config.yaml"
    readOnly: true
    pathType: File
  - name: sock-path
    hostPath: "/opt/trousseau-kms"
    mountPath: "/opt/trousseau-kms"
```

Finally restart Kubernetes API server.
