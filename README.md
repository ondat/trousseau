
<p align="center">
    <img src="https://github.com/ondat/trousseau/blob/main/assets/logo-horizontal.png" >
</p>
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

-----

**Please note**: We take security and users' trust seriously. If you believe you have found a security issue in Trousseau, *please responsibly disclose* by following the [security policy](https://github.com/ondat/trousseau/security/policy). 

-----

This is the home of [Trousseau](https://trousseau.io), an open-source project leveraging the [Kubernetes KMS provider framework](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/) to connect with Key Management Services the Kubernetes native way! 

* Website: https://trousseau.io 
* Announcement & Forum: [GitHub Discussions](https://github.com/ondat/trousseau/discussions)
* Documentation: [GitHub Wiki](https://github.com/ondat/trousseau/wiki)
* Hands-on lab: [Tutorial](https://www.ondat.io/trousseau)
* Recording of the hands-on lab: [DoK London Meetup](https://www.youtube.com/watch?v=BldQHinAIYg) 

## Why Trousseau

Kubernetes platform users are all facing the very same question: ***how to handle Secrets?***  

While there are significant efforts to improve Kubernetes component layers, [the state of Secret Management is not receiving much interests](https://fosdem.org/2021/schedule/event/kubernetes_secret_management/). Using *etcd* to store API object definition & states, Kubernetes secrets are encoded in base64 and shipped into the key value store database.  Even if the filesystems on which *etcd* runs are encrypted, the secrets are still not.   

Instead of leveraging the native Kubernetes way to manage secrets, commercial and open source solutions solve this design flaw by leveraging different approaches all using different toolsets or practices. This leads to training and maintaining niche skills and tools increasing cost and complexity of Kubernetes. 

Once deployed, Trousseau will enable seamless secret management using the native Kubernetes API and ```kubectl``` CLI usage while leveraging an existing Key Management Service (KMS) provider.   

How? By using using the [Kubernetes KMS provider](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/) framework to provide an envelop encryption scheme to encrypt secrets on the fly.

<p align="center">
    <img src="https://github.com/ondat/trousseau/blob/main/assets/Ondat%20Diagram-w-all.png" height="600">
</p>

## About the name
The name ***trousseau*** comes from the French language and is usually associated with keys like in ***trousseau de clés*** meaning ***keyring***.

## Production reference
The following blog post provides an overview of a production use case for a Hong Kong based Service Provider leveraging Suse, RKE2, HashiCorp Vault and Trousseau to secure their workload hosted for Government agencies:
* https://www.ondat.io/news/trousseau-open-source-project-made-available-to-add-security-in-kubernetes 

### Run Trousseau in production
Clone the repo and create your environment file:
```bash
TR_VERSION=31b93747fc4fd438a6b30de70ff16d4a45271366
TR_VERBOSE_LEVEL=1
TR_SOCKET_LOCATION=/opt/trousseau-kms
TR_PROXY_IMAGE=ghcr.io/ondat/trousseau:proxy-${TR_VERSION}
TR_TROUSSEAU_IMAGE=ghcr.io/ondat/trousseau:trousseau-${TR_VERSION}
# Please configure your KMS plugins
TR_ENABLED_PROVIDERS="--enabled-providers=awskms --enabled-providers=vault"
TR_AWSKMS_IMAGE=ghcr.io/ondat/trousseau:awskms-${TR_VERSION}
TR_AWSKMS_CONFIG=awskms.yaml # For Kubernetes, file must exists only for generation
TR_AWSKMS_CREDENTIALS=.aws/credentials
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


## Roadmap
The roadmap items are described within [user story 50](https://github.com/ondat/trousseau/issues/50)  
Trousseau's roadmap milestone for v2 [here](https://github.com/orgs/ondat/projects/1/views/4](https://github.com/ondat/trousseau/milestone/2).

## Contributing Guidelines
We love your input! We want to make contributing to this project as easy and transparent as possible. You can find the full guidelines [here](https://github.com/ondat/trousseau/blob/main/CONTRIBUTING.md).

## Community
Please reach out for any questions or issues via one the following channels:  
* Raise an [issue or PR](https://github.com/ondat/trousseau/issues)
* Join us on [Slack](https://storageos.slack.com/archives/C03CPK9EHJR) 
* Follow us on Twitter [@ondat_io](https://twitter.com/ondat_io)

## License
Trousseau is under the Apache 2.0 license. See [LICENSE](https://github.com/ondat/trousseau/blob/main/LICENSE) file for details.
