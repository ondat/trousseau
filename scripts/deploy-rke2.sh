# to run after fetching the rke2 binaries via
# curl -sfL https://get.rke2.io | sh - 
mkdir -p /etc/rancher/rke2/
mkdir -p /opt/vault-kms/
mkdir -p /var/lib/rancher/rke2/server/cred/
mkdir -p /var/lib/rancher/rke2/server/manifests
cp k8s/config.yaml /opt/vault-kms/config.yaml
cp rke2/vault-kms-provider.yaml /var/lib/rancher/rke2/server/manifests/vault-kms-provider.yaml
cp rke2/config.yaml /etc/rancher/rke2/config.yaml 
cp rke2/encryption-config.json /var/lib/rancher/rke2/server/cred/encryption-config.json
