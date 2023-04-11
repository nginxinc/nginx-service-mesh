---
title: "Install NGINX Service Mesh using Helm"
draft: false
toc: true
description: "This topic explains how to deploy, upgrade, and remove NGINX Service Mesh using Helm."
weight: 400
categories: ["tasks"]
docs: "DOCS-680"
---

## Overview

This topic contains instructions for installing, upgrading, and uninstalling NGINX Service Mesh using Helm.

### Supported Versions

NGINX Service Mesh supports Helm versions 3.2.0 and newer.

### Prerequisites

{{< important >}} Before installing NGINX Service Mesh, make sure you've completed the following steps. {{< /important >}}

- You have a [supported version](#supported-versions) of Helm installed.
- You reviewed the [Configuration Options](#configuration-options).

#### Kubernetes users

- You have a working and [supported]({{< ref "/about/tech-specs.md#supported-versions" >}}) Kubernetes cluster.
- You followed the [Kubernetes Platform Setup]( {{< ref "/get-started/kubernetes-platform/_index.md" >}} ) guide to **prepare your cluster** to work with NGINX Service Mesh.

#### OpenShift users

- You followed the [OpenShift Platform Setup]( {{< ref "/get-started/openshift-platform/_index.md" >}} ) guide to **prepare your cluster** to work with NGINX Service Mesh.

## Getting the Chart Sources

This step is required if you're installing the chart using its sources. Additionally, this step is required for upgrading/deleting the custom resource definitions (CRDs), which NGINX Service Mesh requires by default.

```bash
git clone https://github.com/nginxinc/nginx-service-mesh
cd nginx-service-mesh/helm-chart
git checkout v2.0.0
```

## Adding the Helm Repository

This step is required if you're installing the chart via the helm repository.

```bash
helm repo add nginx-stable https://helm.nginx.com/stable
helm repo update
```

## Installing the Chart

NGINX Service Mesh requires a dedicated namespace for the control plane.
You can create this namespace yourself, or allow Helm to create it for you via the `--create-namespace` flag when installing.

For information on the container images that NGINX Service Mesh uses, see this section on [Images]( {{< ref "/about/tech-specs.md#images" >}} ).

{{< note >}}
If [Persistent Storage]({{< ref "/get-started/kubernetes-platform/persistent-storage.md" >}}) is not configured in your cluster, set the `mTLS.persistentStorage` field to `off`.
{{< /note >}}

NGINX Service Mesh control plane Pods may take some time to become Ready once installed.
Some Pods may display error logs during the startup process.
This typically occurs as the Pods attempt to connect to each other.

OpenShift user may see error events related to security contexts while the NGINX Service Mesh control plane is installing.
These should resolve themselves as each component becomes ready.

Ensure all control plane Pods are in a Ready state before deploying your applications.
You can opt-in the namespaces where you would like auto-injection enabled by labeling the namespace with `injector.nsm.nginx.com/auto-inject=enabled`.

### Install via Repository

To install the chart with the release name `nsm` and namespace `nginx-mesh`, run:

```bash
helm install nsm nginx-stable/nginx-service-mesh --namespace nginx-mesh --create-namespace --wait
```

### Install via Source

To install the chart with the release name `nsm` and namespace `nginx-mesh`, run:

```bash
helm install nsm . --namespace nginx-mesh --create-namespace --wait
```

## Upgrading the Chart

You can upgrade to the latest Helm chart from the version immediately before it (for example, from v1.6.0 to v1.7.0). NGINX Service Mesh doesn't support skipping versions.

{{< important >}}
OpenShift users: view the [upgrade guide]({{< ref "/guides/upgrade.md#upgrade-to-170-in-openshift" >}}) for instructions on upgrading from v1.6.0 to v1.7.0.
{{< /important >}}

### Upgrading the CRDs

Helm does not upgrade the CRDs during a release upgrade. Before you upgrade a release, run the following command to upgrade the CRDs:

```bash
kubectl apply -f crds/
```

The following warning is expected and can be ignored: `Warning: kubectl apply should be used on resource created by either kubectl create --save-config or kubectl apply.`

### Upgrading the Release

To upgrade the release `nsm` in the `nginx-mesh` namespace:

#### Upgrade via Repository

```bash
helm repo update
helm upgrade nsm nginx-stable/nginx-service-mesh  --namespace nginx-mesh --wait
```

Once the upgrade is complete, if your applications support rolling updates, re-roll using the following command:

```bash
kubectl rollout restart <resource type>/<resource name>
```

Otherwise, the application Pods need to be deleted and re-created.

#### Upgrade via Source

```bash
helm upgrade nsm . --namespace nginx-mesh --wait
```

Once the upgrade is complete, if your applications support rolling updates, re-roll using the following command:

```bash
kubectl rollout restart <resource type>/<resource name>
```

Otherwise, the application Pods need to be deleted and re-created.

## Uninstalling the Chart

{{< important >}}
OpenShift users: Before uninstalling, read through the [OpenShift considerations]({{< ref "/get-started/openshift-platform/considerations.md#remove" >}}) guide to make sure you understand the implications.
{{< /important >}}

To uninstall the `nsm` release in the `nginx-mesh` namespace, run:

```bash
helm uninstall nsm --namespace nginx-mesh
```

This command removes most of the Kubernetes components associated with the NGINX Service Mesh release.
Helm does **not** remove the following components:

- CRDs
- `nginx-mesh` namespace
- Spire PersistentVolumeClaim in the `nginx-mesh` namespace

Run this command to remove the CRDS:

```bash
kubectl delete crd -l app.kubernetes.io/part-of==nginx-service-mesh
```

Deleting the namespace will also delete the PersistentVolumeClaim:

```bash
kubectl delete namespace nginx-mesh
```

After uninstalling, re-roll your injected Deployments, DaemonSets, and StatefulSets to remove the sidecar proxy from Pods.

```bash
kubectl rollout restart <resource type>/<resource name>
```

Example:

```bash
kubectl rollout restart deployment/frontend
```

## Configuration Options

The [values.yaml](https://github.com/nginxinc/nginx-service-mesh/blob/main/helm-chart/values.yaml) file within the `nginx-service-mesh` Helm chart contains the deployment configuration for NGINX Service Mesh.
These configuration fields map directly to the `nginx-meshctl deploy` command-line options mentioned throughout our documentation.
More details about these options can be found in the [Configuration]( {{< ref "/get-started/configuration.md" >}} ) guide.
You can update these fields directly in the `values.yaml` file, or by specifying the `--set` flag when running `helm install`.

The following table lists the configurable parameters of the NGINX Service Mesh chart and their default values.

{{% table %}}
| Parameter | Description | Default |
| --- | --- | --- |
| `registry.server` | Hostname:port (if needed) for registry and path to images. Affects: nginx-mesh-controller, nginx-mesh-cert-reloader, nginx-mesh-init, nginx-mesh-metrics, nginx-mesh-sidecar | docker-registry.nginx.com/nsm |
| `registry.imageTag` | Tag used for pulling images from registry. Affects: nginx-mesh-controller, nginx-mesh-cert-reloader, nginx-mesh-init, nginx-mesh-metrics, nginx-mesh-sidecar | 2.0.0 |
| `registry.key` | Contents of your Google Cloud JSON key file. Can be set via `--set-file registry.key=<your-key-file>.json`. Cannot be used with username/password. | "" |
| `registry.username` | Username for accessing private registry. Cannot be used with key. | "" |
| `registry.password` | Password for accessing private registry. Cannot be used with key. | "" |
| `registry.disablePublicImages` | Do not pull third party images from public repositories. If true, registry.server is used for all images. | false |
| `registry.imagePullPolicy` | Image pull policy. | IfNotPresent |
| `accessControlMode` | Default access control mode for service-to-service communication. | allow |
| `environment` | Environment to deploy the mesh into. Valid values: "kubernetes", "openshift". | kubernetes |
| `enableUDP` | Enable UDP traffic proxying (beta). Linux kernel 4.18 or greater is required. | false |
| `nginxErrorLogLevel` | NGINX error log level. | warn |
| `nginxLogFormat` | NGINX log format. | default |
| `nginxLBMethod` | NGINX load balancing method. | least_time |
| `clientMaxBodySize` | NGINX client max body size. Setting to "0" disables checking of client request body size. | 1m |
| `prometheusAddress` | The address of a Prometheus server deployed in your Kubernetes cluster. Address should be in the format `<service-name>.<namespace>:<service-port>`. | "" |
| `telemetry.samplerRatio` | The percentage of traces that are processed and exported to the telemetry backend. Float between 0 and 1. | 0.01 |
| `telemetry.exporters` | The configuration of exporters to send telemetry data to. | |
| `telemetry.exporters.otlp` | The configuration for an OTLP gRPC exporter. | |
| `telemetry.exporters.otlp.host` | The host of the OpenTelemetry gRPC exporter to connect to. Must be accessible from within the cluster. | |
| `telemetry.exporters.otlp.port` | The port of the OpenTelemetry gRPC exporter to connect to. | 4317 |
| `mtls.mode` | mTLS mode for pod-to-pod communication. | permissive |
| `mtls.caTTL` | The CA/signing key TTL in hours(h). Min value 24h. Max value 999999h. | 720h |
| `mtls.svidTTL` | The trust domain of the NGINX Service Mesh. Max value is 999999. | 1h |
| `mtls.persistentStorage` | Use persistent storage; "on" assumes that a StorageClass exists. | on |
| `mtls.spireServerKeyManager` | Storage logic for Spire Server's private keys. | disk |
| `mtls.caKeyType` | The key type used for the SPIRE Server CA. Valid values: "ec-p256", "ec-p384", "rsa-2048", "rsa-4096". | ec-p256 |
| `mtls.upstreamAuthority` | Upstream authority settings. If left empty, SPIRE is used as the upstream authority. See [values.yaml](https://github.com/nginxinc/nginx-service-mesh/blob/main/helm-chart/values.yaml) for how to configure. | {} |
{{% /table %}}
