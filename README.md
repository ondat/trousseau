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

### Setup Kubernetes

The Trousseau KMS Vault provider plugin needs to be set as a Pod starting along with the kube-apimanager. To do so, the ```vault-kms-provider.yaml``` configuration file has to be created within ```/etc/kubernetes/manifests/``` with the following content:

```YAML
apiVersion: v1
kind: Pod
metadata:
  name: vault-kms-provider
  namespace: kube-system
  labels:
    tier: control-plane
    app: vault-kms-provider
spec:
  priorityClassName: system-node-critical
  hostNetwork: true
  containers:
    - name: vault-kms-provider
      image: ghcr.io/trousseau-io/trousseau:v1.0.0-alpha
      imagePullPolicy: Always
      args:
        - -v=5
        - --config-file-path=/opt/vault-kms/config.yaml
        - --listen-addr=unix:///opt/vault-kms/vaultkms.socket                            # [REQUIRED] Version of the key to use
        - --log-format-json=false
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
          - ALL
        readOnlyRootFilesystem: true
        runAsUser: 0
      ports:
        - containerPort: 8787
          protocol: TCP
      livenessProbe:
        httpGet:
          path: /healthz
          port: 8787
        failureThreshold: 3
        periodSeconds: 10
      resources:
        requests:
          cpu: 50m
          memory: 64Mi
        limits:
          cpu: 300m
          memory: 256Mi
      volumeMounts:
        - name: etc-kubernetes
          mountPath: /opt/vault-kms
  volumes:
    - name: etc-kubernetes
      hostPath:
        path: /opt/vault-kms
```

Then, create the directory ```/opt/vault-kms/``` to hosts the trousseau configuration files:

```config.yaml```
```YAML
provider: vault
vault:
  keynames:
  -  vault-kms-demo
  address: https://addresss:8200
  token: s.oYpiOmnWL0PFDPS2ImJTdhRf.CxT2N
```

```encryption_config.yaml```
```YAML
kind: EncryptionConfiguration
apiVersion: apiserver.config.k8s.io/v1
resources:
  - resources:
      - secrets
    providers:
      - kms:
          name: vaultprovider
          endpoint: unix:///opt/vault-kms/vaultkms.socket
          cachesize: 1000
      - identity: {}
```

Finally, within the definition of ```kube-apiserver.yaml``` usually located within ```/etc/kubernetes/manifests```, add the parameter ```--encryption-provider-config=/opt/vault-kms/encryption_config.yaml```to call the related configuration and setup the volume mounts. 

The file would look like: 

```YAML
apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubeadm.kubernetes.io/kube-apiserver.advertise-address.endpoint: 10.0.10.4:6443
  creationTimestamp: null
  labels:
    component: kube-apiserver
    tier: control-plane
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-apiserver
    - --advertise-address=10.0.10.4
    - --allow-privileged=true
    - --anonymous-auth=True
    - --apiserver-count=3
    - --authorization-mode=Node,RBAC
    - --bind-address=0.0.0.0
    - --client-ca-file=/etc/kubernetes/ssl/ca.crt
    - --default-not-ready-toleration-seconds=300
    - --default-unreachable-toleration-seconds=300
    - --enable-admission-plugins=NodeRestriction
    - --enable-aggregator-routing=False
    - --enable-bootstrap-token-auth=true
    - --encryption-provider-config=/opt/vault-kms/encryption_config.yaml

... snip ...

    volumeMounts:
    - mountPath: /etc/ssl/certs
      name: ca-certs
      readOnly: true
    - mountPath: /etc/ca-certificates
      name: etc-ca-certificates
      readOnly: true
    - mountPath: /etc/ssl/etcd/ssl
      name: etcd-certs-0
      readOnly: true
    - mountPath: /etc/kubernetes/ssl
      name: k8s-certs
      readOnly: true
    - mountPath: /usr/local/share/ca-certificates
      name: usr-local-share-ca-certificates
      readOnly: true
    - mountPath: /usr/share/ca-certificates
      name: usr-share-ca-certificates
      readOnly: true
    - mountPath: /opt/vault-kms
      name: vaultkms
      readOnly: true
  hostNetwork: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /etc/ssl/certs
      type: DirectoryOrCreate
    name: ca-certs
  - hostPath:
      path: /etc/ca-certificates
      type: DirectoryOrCreate
    name: etc-ca-certificates
  - hostPath:
      path: /etc/ssl/etcd/ssl
      type: DirectoryOrCreate
    name: etcd-certs-0
  - hostPath:
      path: /etc/kubernetes/ssl
      type: DirectoryOrCreate
    name: k8s-certs
  - hostPath:
      path: /usr/local/share/ca-certificates
      type: DirectoryOrCreate
    name: usr-local-share-ca-certificates
  - hostPath:
      path: /usr/share/ca-certificates
      type: ""
    name: usr-share-ca-certificates
  - hostPath:
      path: /opt/vault-kms
      type: ""
    name: vaultkms
status: {}

```

**NOTES: depending on the Kubernetes distribution, the kubelet might not include the ```/etc/kubernetes/manifests``` for ```staticPodPath```. 
Verifiy ```kubelet-config.yaml``` within ```/etc/kubernetes``` to ensure this parameter is present.  
Here is an example of the file:

```YAML
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
nodeStatusUpdateFrequency: "10s"
failSwapOn: True
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: True
  x509:
    clientCAFile: /etc/kubernetes/ssl/ca.crt
authorization:
  mode: Webhook
staticPodPath: /etc/kubernetes/manifests
cgroupDriver: systemd
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
maxPods: 110
address: 10.0.10.4
readOnlyPort: 0
healthzPort: 10248
healthzBindAddress: 127.0.0.1
kubeletCgroups: /systemd/system.slice
clusterDomain: cluster.local
protectKernelDefaults: true
rotateCertificates: true
clusterDNS:
- 169.254.25.10
kubeReserved:
  cpu: 200m
  memory: 512Mi
resolvConf: "/run/systemd/resolve/resolv.conf"
eventRecordQPS: 5
shutdownGracePeriod: 60s
shutdownGracePeriodCriticalPods: 20s
```

## RKE Specifics
When deploying RKE, use version 1.3.3-rc2 of the tool to benefit of a fix related to handling YAML configuration within the ```cluster.yml``` definition. 

After successfuly deploying kubernetes using a standard ```cluster.yml``` with ```rke up```, modify the file as such along with the previous steps to setup the Vault and the configuration files:

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

### Enable Trousseau KMS Vault
At this stage, for none RKE based deployment, restart the kube-apimanager to along with the kubelet if necessary.  Refer to your kubernetes distribution to do so.


## Setup monitoring
Trousseau is coming with a Prometheus endpoint for monitoring with basic Grafana dashboard.  

An example of configuration for Prometheus endpoint access:

```YAML
---
kind: Service
apiVersion: v1
metadata:
  name: vault-kms-metrics
  namespace: kube-system
  labels:
     app: vault-kms-provider
spec:
  selector:
     app: vault-kms-provider
  ports:
  - name: metrics
    port: 8095
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: vault-kms-provider
  labels:
    release: prometheus
spec:
  namespaceSelector:
    matchNames:
      - kube-system
  selector:
    matchLabels:
      app: vault-kms-provider
  endpoints:
  - port: metrics
    path: /metrics
```

An example of configuration for Grafana dashboard:

```YAML
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: vault-kms-grafana-dashboard
  labels:
    grafana_dashboard: "true"
data:
  vault-kms-dashboard.json: |-
    {
      "annotations": {
        "list": [
          {
            "builtIn": 1,
            "datasource": "-- Grafana --",
            "enable": true,
            "hide": true,
            "iconColor": "rgba(0, 211, 255, 1)",
            "name": "Annotations & Alerts",
            "target": {
              "limit": 100,
              "matchAny": false,
              "tags": [],
              "type": "dashboard"
            },
            "type": "dashboard"
          }
        ]
      },
      "editable": true,
      "gnetId": null,
      "graphTooltip": 0,
      "id": 26,
      "iteration": 1632384872537,
      "links": [],
      "panels": [
        {
          "datasource": "${datasource}",
          "fieldConfig": {
            "defaults": {
              "color": {
                "mode": "palette-classic"
              },
              "custom": {
                "axisLabel": "",
                "axisPlacement": "auto",
                "barAlignment": 0,
                "drawStyle": "line",
                "fillOpacity": 0,
                "gradientMode": "none",
                "hideFrom": {
                  "legend": false,
                  "tooltip": false,
                  "viz": false
                },
                "lineInterpolation": "linear",
                "lineWidth": 1,
                "pointSize": 5,
                "scaleDistribution": {
                  "type": "linear"
                },
                "showPoints": "auto",
                "spanNulls": false,
                "stacking": {
                  "group": "A",
                  "mode": "none"
                },
                "thresholdsStyle": {
                  "mode": "off"
                }
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  },
                  {
                    "color": "red",
                    "value": 80
                  }
                ]
              }
            },
            "overrides": []
          },
          "gridPos": {
            "h": 8,
            "w": 24,
            "x": 0,
            "y": 0
          },
          "id": 30,
          "options": {
            "legend": {
              "calcs": [],
              "displayMode": "list",
              "placement": "bottom"
            },
            "tooltip": {
              "mode": "single"
            }
          },
          "targets": [
            {
              "exemplar": true,
              "expr": "(rate(kms_request_sum{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[1m]))*60",
              "hide": false,
              "interval": "",
              "legendFormat": "{{operation}}",
              "refId": "B"
            }
          ],
          "title": "Operations",
          "type": "timeseries"
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 8
          },
          "id": 17,
          "panels": [],
          "repeat": null,
          "title": "CPU Usage",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 9
          },
          "hiddenSeries": false,
          "id": 1,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "requests",
              "color": "#F2495C",
              "fill": 0,
              "hideTooltip": true,
              "legend": true,
              "linewidth": 2,
              "stack": false
            },
            {
              "alias": "limits",
              "color": "#FF9830",
              "fill": 0,
              "hideTooltip": true,
              "legend": true,
              "linewidth": 2,
              "stack": false
            }
          ],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container!=\"\"}) by (container)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{container}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(\n    kube_pod_container_resource_requests{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", resource=\"cpu\"}\n)\n",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "requests",
              "legendLink": null,
              "refId": "B",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(\n    kube_pod_container_resource_limits{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", resource=\"cpu\"}\n)\n",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "limits",
              "legendLink": null,
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "CPU Usage",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 16
          },
          "id": 18,
          "panels": [],
          "repeat": null,
          "title": "CPU Throttling",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 17
          },
          "hiddenSeries": false,
          "id": 2,
          "legend": {
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(increase(container_cpu_cfs_throttled_periods_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container!=\"\", image!=\"\"}[5m])) by (container) /sum(increase(container_cpu_cfs_periods_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m])) by (container)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{container}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 0.25,
              "yaxis": "left"
            }
          ],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "CPU Throttling",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": 1,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 24
          },
          "id": 19,
          "panels": [],
          "repeat": null,
          "title": "CPU Quota",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "columns": [],
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fontSize": "100%",
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 25
          },
          "id": 3,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "pageSize": null,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "showHeader": true,
          "sort": {
            "col": 0,
            "desc": true
          },
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "styles": [
            {
              "alias": "Time",
              "align": "auto",
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "pattern": "Time",
              "type": "hidden"
            },
            {
              "alias": "CPU Usage",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #A",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "CPU Requests",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #B",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "CPU Requests %",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #C",
              "thresholds": [],
              "type": "number",
              "unit": "percentunit"
            },
            {
              "alias": "CPU Limits",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #D",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "CPU Limits %",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #E",
              "thresholds": [],
              "type": "number",
              "unit": "percentunit"
            },
            {
              "alias": "Container",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "container",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "pattern": "/.*/",
              "thresholds": [],
              "type": "string",
              "unit": "short"
            }
          ],
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container!=\"\", image!=\"\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(cluster:namespace:pod_cpu:active:kube_pod_container_resource_requests{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "B",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container) / sum(cluster:namespace:pod_cpu:active:kube_pod_container_resource_requests{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "C",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(cluster:namespace:pod_cpu:active:kube_pod_container_resource_limits{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "D",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container) / sum(cluster:namespace:pod_cpu:active:kube_pod_container_resource_limits{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "E",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "CPU Quota",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "transform": "table",
          "type": "table-old",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 32
          },
          "id": 20,
          "panels": [],
          "repeat": null,
          "title": "Memory Usage",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 33
          },
          "hiddenSeries": false,
          "id": 4,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "requests",
              "color": "#F2495C",
              "dashes": true,
              "fill": 0,
              "hideTooltip": true,
              "legend": true,
              "linewidth": 2,
              "stack": false
            },
            {
              "alias": "limits",
              "color": "#FF9830",
              "dashes": true,
              "fill": 0,
              "hideTooltip": true,
              "legend": true,
              "linewidth": 2,
              "stack": false
            }
          ],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(container_memory_working_set_bytes{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container!=\"\", image!=\"\"}) by (container)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{container}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(\n    kube_pod_container_resource_requests{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", resource=\"memory\"}\n)\n",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "requests",
              "legendLink": null,
              "refId": "B",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(\n    kube_pod_container_resource_limits{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", resource=\"memory\"}\n)\n",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "limits",
              "legendLink": null,
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Memory Usage (WSS)",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 40
          },
          "id": 21,
          "panels": [],
          "repeat": null,
          "title": "Memory Quota",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "columns": [],
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fontSize": "100%",
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 41
          },
          "id": 5,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "pageSize": null,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "showHeader": true,
          "sort": {
            "col": 0,
            "desc": true
          },
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "styles": [
            {
              "alias": "Time",
              "align": "auto",
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "pattern": "Time",
              "type": "hidden"
            },
            {
              "alias": "Memory Usage (WSS)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #A",
              "thresholds": [],
              "type": "number",
              "unit": "bytes"
            },
            {
              "alias": "Memory Requests",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #B",
              "thresholds": [],
              "type": "number",
              "unit": "bytes"
            },
            {
              "alias": "Memory Requests %",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #C",
              "thresholds": [],
              "type": "number",
              "unit": "percentunit"
            },
            {
              "alias": "Memory Limits",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #D",
              "thresholds": [],
              "type": "number",
              "unit": "bytes"
            },
            {
              "alias": "Memory Limits %",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #E",
              "thresholds": [],
              "type": "number",
              "unit": "percentunit"
            },
            {
              "alias": "Memory Usage (RSS)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #F",
              "thresholds": [],
              "type": "number",
              "unit": "bytes"
            },
            {
              "alias": "Memory Usage (Cache)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #G",
              "thresholds": [],
              "type": "number",
              "unit": "bytes"
            },
            {
              "alias": "Memory Usage (Swap)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #H",
              "thresholds": [],
              "type": "number",
              "unit": "bytes"
            },
            {
              "alias": "Container",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "container",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "pattern": "/.*/",
              "thresholds": [],
              "type": "string",
              "unit": "short"
            }
          ],
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(container_memory_working_set_bytes{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container!=\"\", image!=\"\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(cluster:namespace:pod_memory:active:kube_pod_container_resource_requests{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "B",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(container_memory_working_set_bytes{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", image!=\"\"}) by (container) / sum(cluster:namespace:pod_memory:active:kube_pod_container_resource_requests{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "C",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(cluster:namespace:pod_memory:active:kube_pod_container_resource_limits{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "D",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(container_memory_working_set_bytes{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container!=\"\", image!=\"\"}) by (container) / sum(cluster:namespace:pod_memory:active:kube_pod_container_resource_limits{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "E",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(container_memory_rss{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container != \"\", container != \"POD\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "F",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(container_memory_cache{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container != \"\", container != \"POD\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "G",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum(container_memory_swap{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", container != \"\", container != \"POD\"}) by (container)",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "H",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Memory Quota",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "transform": "table",
          "type": "table-old",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 48
          },
          "id": 22,
          "panels": [],
          "repeat": null,
          "title": "Bandwidth",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 49
          },
          "hiddenSeries": false,
          "id": 6,
          "interval": "1m",
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(irate(container_network_receive_bytes_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[$__rate_interval])) by (pod)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Receive Bandwidth",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 49
          },
          "hiddenSeries": false,
          "id": 7,
          "interval": "1m",
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(irate(container_network_transmit_bytes_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[$__rate_interval])) by (pod)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Transmit Bandwidth",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 56
          },
          "id": 23,
          "panels": [],
          "repeat": null,
          "title": "Rate of Packets",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 57
          },
          "hiddenSeries": false,
          "id": 8,
          "interval": "1m",
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(irate(container_network_receive_packets_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[$__rate_interval])) by (pod)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Rate of Received Packets",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "pps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 57
          },
          "hiddenSeries": false,
          "id": 9,
          "interval": "1m",
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(irate(container_network_transmit_packets_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[$__rate_interval])) by (pod)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Rate of Transmitted Packets",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "pps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 64
          },
          "id": 24,
          "panels": [],
          "repeat": null,
          "title": "Rate of Packets Dropped",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 65
          },
          "hiddenSeries": false,
          "id": 10,
          "interval": "1m",
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(irate(container_network_receive_packets_dropped_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[$__rate_interval])) by (pod)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Rate of Received Packets Dropped",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "pps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 65
          },
          "hiddenSeries": false,
          "id": 11,
          "interval": "1m",
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum(irate(container_network_transmit_packets_dropped_total{namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[$__rate_interval])) by (pod)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Rate of Transmitted Packets Dropped",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "pps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 72
          },
          "id": 25,
          "panels": [],
          "repeat": null,
          "title": "Storage IO - Distribution(Pod - Read & Writes)",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": -1,
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 73
          },
          "hiddenSeries": false,
          "id": 12,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "ceil(sum by(pod) (rate(container_fs_reads_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m])))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "Reads",
              "legendLink": null,
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "ceil(sum by(pod) (rate(container_fs_writes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m])))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "Writes",
              "legendLink": null,
              "refId": "B",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "IOPS",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 73
          },
          "hiddenSeries": false,
          "id": 13,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum by(pod) (rate(container_fs_reads_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", pod=~\"$pod\"}[5m]))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "Reads",
              "legendLink": null,
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum by(pod) (rate(container_fs_writes_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", pod=~\"$pod\"}[5m]))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "Writes",
              "legendLink": null,
              "refId": "B",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "ThroughPut",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 80
          },
          "id": 26,
          "panels": [],
          "repeat": null,
          "title": "Storage IO - Distribution(Containers)",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": -1,
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 81
          },
          "hiddenSeries": false,
          "id": 14,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "ceil(sum by(container) (rate(container_fs_reads_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\", pod=\"$pod\"}[5m]) + rate(container_fs_writes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m])))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{container}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "IOPS(Reads+Writes)",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 10,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 81
          },
          "hiddenSeries": false,
          "id": 15,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "alertThreshold": true
          },
          "percentage": false,
          "pluginVersion": "8.1.2",
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_reads_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]) + rate(container_fs_writes_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{container}}",
              "legendLink": null,
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "ThroughPut(Read+Write)",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "collapsed": false,
          "datasource": null,
          "gridPos": {
            "h": 1,
            "w": 24,
            "x": 0,
            "y": 88
          },
          "id": 27,
          "panels": [],
          "repeat": null,
          "title": "Storage IO - Distribution",
          "type": "row"
        },
        {
          "aliasColors": {},
          "bars": false,
          "columns": [],
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fontSize": "100%",
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 89
          },
          "id": 16,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "pageSize": null,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "showHeader": true,
          "sort": {
            "col": 4,
            "desc": true
          },
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "styles": [
            {
              "alias": "Time",
              "align": "auto",
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "pattern": "Time",
              "type": "hidden"
            },
            {
              "alias": "IOPS(Reads)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": -1,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #A",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "IOPS(Writes)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": -1,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #B",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "IOPS(Reads + Writes)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": -1,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #C",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "Throughput(Read)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #D",
              "thresholds": [],
              "type": "number",
              "unit": "Bps"
            },
            {
              "alias": "Throughput(Write)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #E",
              "thresholds": [],
              "type": "number",
              "unit": "Bps"
            },
            {
              "alias": "Throughput(Read + Write)",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "Value #F",
              "thresholds": [],
              "type": "number",
              "unit": "Bps"
            },
            {
              "alias": "Container",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "link": false,
              "linkTargetBlank": false,
              "linkTooltip": "Drill down",
              "linkUrl": "",
              "pattern": "container",
              "thresholds": [],
              "type": "number",
              "unit": "short"
            },
            {
              "alias": "",
              "align": "auto",
              "colorMode": null,
              "colors": [],
              "dateFormat": "YYYY-MM-DD HH:mm:ss",
              "decimals": 2,
              "pattern": "/.*/",
              "thresholds": [],
              "type": "string",
              "unit": "short"
            }
          ],
          "targets": [
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_reads_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_writes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "B",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_reads_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]) + rate(container_fs_writes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "C",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_reads_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "D",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_writes_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "E",
              "step": 10
            },
            {
              "exemplar": true,
              "expr": "sum by(container) (rate(container_fs_reads_bytes_total{container!=\"\", namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]) + rate(container_fs_writes_bytes_total{container!=\"\",namespace=\"kube-system\", pod=\"vault-kms-provider-kms-vault-control-plane\"}[5m]))",
              "format": "table",
              "instant": true,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "F",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Current Storage IO",
          "tooltip": {
            "shared": false,
            "sort": 0,
            "value_type": "individual"
          },
          "transform": "table",
          "type": "table-old",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": 0,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        }
      ],
      "refresh": "10s",
      "schemaVersion": 30,
      "style": "dark",
      "tags": [],
      "templating": {
        "list": [
          {
            "current": {
              "selected": false,
              "text": "default",
              "value": "default"
            },
            "description": null,
            "error": null,
            "hide": 0,
            "includeAll": false,
            "label": null,
            "multi": false,
            "name": "datasource",
            "options": [],
            "query": "prometheus",
            "queryValue": "",
            "refresh": 1,
            "regex": "",
            "skipUrlSync": false,
            "type": "datasource"
          }
        ]
      },
      "time": {
        "from": "now-5m",
        "to": "now"
      },
      "timepicker": {
        "refresh_intervals": [
          "5s",
          "10s",
          "30s",
          "1m",
          "5m",
          "15m",
          "30m",
          "1h",
          "2h",
          "1d"
        ],
        "time_options": [
          "5m",
          "15m",
          "1h",
          "6h",
          "12h",
          "24h",
          "2d",
          "7d",
          "30d"
        ]
      },
      "timezone": "UTC",
      "title": "Vault KMS",
      "uid": "E6b8xFH7z",
      "version": 5
    }
```
