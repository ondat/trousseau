
<p align="center">
    <img src="https://raw.githubusercontent.com/ondat/trousseau/main/assets/android-chrome-192x192.png?sanitize=true" height="130">
</p>
<p align="center">
        <a href="https://lgtm.com/projects/g/ondat/trousseau/alerts/"><img alt="Total alerts" src="https://img.shields.io/lgtm/alerts/g/ondat/trousseau.svg?logo=lgtm&logoWidth=18"/></a>
      <a href="https://github.com/Trousseau-io/trousseau/actions/workflows/gosec-scanner-on-pull_request.yaml" alt="gosec">
        <img src="https://github.com/Trousseau-io/trousseau/actions/workflows/gosec-scanner-on-pull_request.yaml/badge.svg?branch=main" /></a>
      <a href="https://github.com/Trousseau-io/trousseau/actions/workflows/codeql-analysis.yml" alt="codeql">
        <img src="https://github.com/Trousseau-io/trousseau/actions/workflows/codeql-analysis.yml/badge.svg" /></a>
      <a href="https://github.com/Trousseau-io/trousseau/actions/workflows/go-lint-scan-pull_request.yaml" alt="golangci-lint">
        <img src="https://github.com/Trousseau-io/trousseau/actions/workflows/go-lint-scan-pull_request.yaml/badge.svg" /></a>
      <a href="https://bestpractices.coreinfrastructure.org/projects/5460" alt="CII Best Practices">
        <img src="https://bestpractices.coreinfrastructure.org/projects/5460/badge" /></a>
</p>

This is the home of [Trousseau](https://trousseau.io), an open-source project leveraging the [Kubernetes KMS provider framework](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/) to connect any Key Management Service the Kubernetes native way! 

This repo hosts:
* Trousseau code
* documentation [wiki](https://github.com/ondat/trousseau/wiki)
* container image for [HashiCorp Vault](https://github.com/ondat/trousseau/pkgs/container/trousseau)

## why Trousseau

Kubernetes platform users are all facing the very same question:  
***how to handle Secrets?***  

While there are significant efforts to improve Kubernetes component layers, [the state of Secret Management is not receiving much interests](https://fosdem.org/2021/schedule/event/kubernetes_secret_management/). Using *etcd* to store API object definition & states, Kubernetes secrets are encoded in Base64 and shipped into the key value store database.  Even if the filesystems on which *etcd* runs are encrypted, the secrets are still not.   

Instead of leveraging the native Kubernetes way to manage secrets, commercial and open source solutions solve this design flaw by leveraging different approaches all using different toolsets or practices. This leads to training and maintaining niche skills and tools increasing cost and complexity of Kubernetes day 0, 1 and 2. 

Once deployed, Trousseau will enable seamless secret management using the native Kubernetes API and ```kubectl``` CLI usage while leveraging an existing Key Management Service (KMS) provider.  
How? By using using the [Kubernetes KMS provider](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/) framework to provide an envelop encryption scheme to encrypt secrets on the fly.


<p align="center">
    <img src="https://raw.githubusercontent.com/Trousseau-io/trousseau/main/assets/trousseau_overview.png" height="480">
</p>

## about the name
The name ***trousseau*** comes from the French language and is usually associated with keys like in ***trousseau de clés*** meaning ***keyring***.
