### Build status:
[![gosec](https://github.com/Trousseau-io/trousseau/actions/workflows/gosec-scanner-on-pull_request.yaml/badge.svg?branch=main)](https://github.com/Trousseau-io/trousseau/actions/workflows/gosec-scanner-on-pull_request.yaml)
[![CodeQL](https://github.com/Trousseau-io/trousseau/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Trousseau-io/trousseau/actions/workflows/codeql-analysis.yml)
[![golangci-lint](https://github.com/Trousseau-io/trousseau/actions/workflows/go-lint-scan-pull_request.yaml/badge.svg)](https://github.com/Trousseau-io/trousseau/actions/workflows/go-lint-scan-pull_request.yaml)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5460/badge)](https://bestpractices.coreinfrastructure.org/projects/5460)

Welcome to the Trousseau Git repo!

### why Trousseau

Kubernetes platform users are all facing the very same question:  
***how to handle Secrets?***  

While there are significant efforts to improve Kubernetes component layers, [the state of Secret Management is not receiving much interests](https://fosdem.org/2021/schedule/event/kubernetes_secret_management/).   
Using *etcd* to store API object definition & states, Kubernetes secrets are encoded in Base64 and shipped into the key value store database.  Even if the filesystems on which *etcd* runs are encrypted, the secrets are still not.   

Instead of leveraging the native Kubernetes way to manage secrets, commercial and open source solutions solve this design flaw by leveraging different approaches all using different toolsets or practices. This leads to training and maintaining niche skills and tools increasing cost and complexity of Kubernetes day 0, 1 and 2. 

Once deployed, Trousseau will enable seamless secret management using the native Kubernetes API and ```kubectl``` CLI usage while leveraging an existing Key Management Service (KMS) provider.  
How? By using using the [Kubernetes KMS provider](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/) framework to provide an envelop encryption scheme to encrypt secrets on the fly.

### what is Trousseau

Trousseau is: 

* open-source project
* design based on [Kubernetes KMS provider design](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/)
* design to be a framework for any KMS provider (see release notes)

### about the name
The name ***trousseau*** comes from the French language and is usually associated with keys like in ***trousseau de clés*** meaning ***keyring***.
