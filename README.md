# Build status:
[![run golang-ci-lint](https://github.com/Trousseau-io/trousseau/actions/workflows/go-lint-scan-pull_request.yaml/badge.svg?branch=main)](https://github.com/Trousseau-io/trousseau/actions/workflows/go-lint-scan-pull_request.yaml)  
[![run gosec scanner](https://github.com/Trousseau-io/trousseau/actions/workflows/gosec-scanner-on-pull_request.yaml/badge.svg?branch=main)](https://github.com/Trousseau-io/trousseau/actions/workflows/gosec-scanner-on-pull_request.yaml)

# Trousseau KMS provider plugin for Vault
Table of contents  
* [Setup Vault](#setup-vault)  
  * [Requirements](#requirements)  
  * [Shell Environment Variables](#shell-environment-variables)  
  * [Enable a Vault Transit Engine](#enable-a-vault-transit-engine)  
  * [Setup Kubernetes](#setup-kubernetes)  
  * [RKE Specifics](#rke-specifics)  
  * [Enable Trousseau KMS Vault](#enable-trousseau-kms-vault)  
* [Setup monitoring](#setup-monitoring)  

## Setup Vault

### Requirements
The following are required:
- a working kubernetes cluster 
- a Vault instance (Community or Enterprise)
- a SSH access to the control plane nodes as an admin
- the necessary user permissions to handle files in ```etc``` and restart serivces, root is best, sudo is better ;)
- the vault cli tool 
- the kubectl cli tool

### Shell Environment Variables
Export environment variables to reach out the Vault instance:

```bash
export VAULT_ADDR="https://addresss:8200"
export VAULT_TOKEN="s.oYpiOmnWL0PFDPS2ImJTdhRf.CxT2N"
```
   
**NOTE: when using the Vault Enterprise, the concept of namespace is introduced.**   
This requires an additional environment variables to target the base root namespace:

```bash
export VAULT_NAMESPACE=admin
```
or a sub namespace like admin/gke01

```bash
export VAULT_NAMESPACE=admin/gke01
```

### Enable a Vault Transit engine

Make sure to have a Transit engine enable within Vault:

```bash
vault secrets enable transit

Success! Enabled the transit secrets engine at: transit/
```

List the secret engines:
```bash
vault secrets list
Path          Type            Accessor                 Description
----          ----            --------                 -----------
cubbyhole/    ns_cubbyhole    ns_cubbyhole_491a549d    per-token private secret storage
identity/     ns_identity     ns_identity_01d57d96     identity store
sys/          ns_system       ns_system_d0f157ca       system endpoints used for control, policy and debugging
transit/      transit         transit_3a41addc         n/a
```

**NOTE about missing VAULT_NAMESPACE**  
Not exporting the VAULT_NAMESPACE will results in a similar error message when enabling the transit engine or even trying to list them:

```
vault secrets enable transit

Error enabling: Error making API request.

URL: POST https://vault-dev.vault.3c414da7-6890-49b8-b635-e3808a5f4fee.aws.hashicorp.cloud:8200/v1/sys/mounts/transit
Code: 403. Errors:

* 1 error occurred:
        * permission denied
```

Finally, create a transit key:

```bash
vault write -f transit/keys/vault-kms-demo
Success! Data written to: transit/keys/vault-kms-demo
```

## Kubernetes
### Vanilla k8s (like GKE)
**The following steps needs to be executed on every node part of the control plane; usually one master node in dev/test environment, 3 in production environment.**

The Trousseau KMS Vault provider plugin needs to be set as a Pod starting along with the kube-apimanager.  
To do so, the ```vault-kms-provider.yaml``` configuration file from ```scripts/k8s``` can be used as a template and should be added to every nodes part of the control plane within the directory ```/etc/kubernetes/manifests/```.   

**Note that only the image version and the Vault namespace are open for changes to match your enviroment and everything else is at your own risks.**

Then, create the directory ```/opt/vault-kms/``` to hosts the trousseau configuration files:
* ```config.yaml``` to be update to match your environment
* ```encryption_config.yaml``` as-is and to not modify for any reasons

Add the parameter ```--encryption-provider-config=/opt/vault-kms/encryption_config.yaml``` within the ```kube-apiserver.yaml``` configuration file which is usually located in the folder ```/etc/kubernetes/manifests``` and add the extra volumes bindings for ```/opt/vault-kms```. 

An example is available with the directory ```scripts/k8s``` with commented sections.   
**Edit your own ```kube-apiserver.yaml``` file and not copy/paste the entire content of the example file.**

**NOTES: depending on the Kubernetes distribution, the kubelet might not include the ```/etc/kubernetes/manifests``` for ```staticPodPath```. 
Verifiy ```kubelet-config.yaml``` within ```/etc/kubernetes``` to ensure this parameter is present.**

Finally, restart the ```kube-apiserver``` to apply the configuration. Trousseau should start allow with it.

### RKE (not RKE2) Specifics
When deploying using rke (not RKE2) and after successfuly deploying a working kubernetes using your ```cluster.yml``` with ```rke up```, modify the following sections of your ```cluster.yml```:

the ```kube-api``` section:
```YAML
  kube-api:
    image: ""
    extra_args:
      encryption-provider-config: /opt/vault-kms/encryption_config.yaml
    extra_binds: 
      - "/opt/vault-kms:/opt/vault-kms"
```

the ```kubelet``` section:
```YAML
  kubelet:
    image: ""
    extra_args: 
      pod-manifest-path: "/etc/kubernetes/manifests"
    extra_binds: 
      - "/opt/vault-kms:/opt/vault-kms"
```

Once everything in place, perform a ```rke up``` to reload the configuration.

### RKE2 Specifics
Building a Kubernetes RKE2 cluster is a different approach then with RKE fromn a configuration perpective. Here is a quick step by step approach:

* prepare the directory ```/opt/vault-kms``` like explain within the Vanilla Kubernetes
* add the following content in the ```config.yaml``` file in ```/etc/rancher/rke2/``` of each control plane node:
``` 
kube-apiserver-extra-env:
  - "--encryption-provider-config=/opt/vault-kms/encryption_config.yaml" 
kube-apiserver-extra-mount:
  - "/opt/vault-kms:/opt/vault-kms"
```
* add the file ```vault-kms-provider.yaml``` from the folder ```scripts/rke2``` in the folder ```/var/lib/rancher/rke2/server/manifests``` of each control plane node.
* restart the ```rke2-server``` service via ```systemctl restart rke2-server.service``` or reboot the node.

Note that based on the above, the cluster can be build from the ground up without first creating and then updating with the additional configuraiton. 


## Setup monitoring
Trousseau is coming with a Prometheus endpoint for monitoring with basic Grafana dashboard.  

An example of configuration for the Prometheus endpoint access is available within the folder ```scripts/templates/monitoring``` with the name ```prometheus.yaml```. 

An example of configuration for the Grafana dashboard configuration is available within the folder ```scripts/templates/monitoring``` with the name ```grafana-dashboard.yaml```. 
