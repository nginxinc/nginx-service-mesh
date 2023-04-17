---
title: "Upgrade NGINX Service Mesh"
draft: false
toc: true
description: "This topic explains how to upgrade NGINX Service Mesh."
weight: 80
categories: ["tasks"]
docs: "DOCS-700"
---

## Overview

This document contains instructions for upgrading NGINX Service Mesh.

For Helm users, see [how to upgrade using Helm]( {{< ref "/get-started/install-with-helm.md#upgrading-the-chart" >}} ).

Incompatibilities between previous and current versions may require specific steps in order to upgrade properly. Be sure to read through the specific details for your target version.

- [In-Place Upgrade](#in-place-upgrade)
- [Manual Upgrade](#manual-upgrade)
- [Upgrade to 1.7.0 in OpenShift](#upgrade-to-170-in-openshift)

{{< important >}}
When upgrading from NGINX Service Mesh prior to v1.7, any tracing settings that do not correspond to an OpenTelemetry configuration will be removed. To deploy OpenTelemetry services and configure your mesh for OpenTelemetry tracing refer to the [Monitoring and Tracing]( {{< ref "/guides/monitoring-and-tracing.md" >}} ) guide.
{{< /important >}}

## In-Place Upgrade

{{< important >}}
Each release may have version-specific steps for performing a successful upgrade. If there is a dedicated entry in this document for your target version, you should follow those steps instead.
{{< /important >}}

{{< important >}}
If you wish to change the [deployment configuration]( {{< ref "nginx-meshctl.md#deploy" >}} ) (for example, setting a new flag or changing the value of an existing flag), then you will need to follow the [Manual Upgrade](#manual-upgrade) steps.
{{< /important >}}

You can upgrade to the latest `nginx-meshctl` release from the version immediately before it (for example, from v1.4.0 to v1.5.0). After you have [installed]({{< ref "/get-started/install.md#install-the-cli" >}}) the latest version of `nginx-meshctl`, you can run:

```bash
nginx-meshctl upgrade
```

This will upgrade the NGINX Service Mesh control plane to the latest version. All user configuration (traffic policies, mesh configuration, deploy configuration) is preserved through the upgrade. New applications will not be injected during the upgrade and existing applications will not receive configuration updates. Existing applications may not function properly until updated. It is recommended that upgrades are only performed during a maintenance window due to the brief downtime.

By default, the mesh will pull images from the registry that it was originally deployed with. If you would like to pull from a different registry, you can use the`--registry-server` flag.

Additionally, if you would like to upgrade to a custom image tag you can use the `--image-tag` flag.

Once the upgrade is complete, if your applications support rolling updates, re-roll using the following command:

```bash
kubectl rollout restart <resource type>/<resource name>
```

Otherwise, the application Pods need to be deleted and re-created.

## Manual Upgrade

{{< important >}}
This upgrade method is the most disruptive, as it involves fully removing the existing mesh and deploying the newer version.
{{< /important >}}

If breaking changes are introduced between versions, or you wish to change the [deployment configuration]( {{< ref "nginx-meshctl.md#deploy" >}} ), then a manual upgrade strategy is necessary.

{{< note >}}
Some deployment configuration fields can be updated after the mesh has already been deployed, avoiding the need for manual upgrades. Those fields are discussed in the [API reference]( {{< ref "api-usage.md#modifying-the-global-mesh-configuration" >}} ) guide.
{{< /note >}}

Before downloading the newer version of `nginx-meshctl`, you will need to remove NGINX Service Mesh using your existing version of `nginx-meshctl`. See the following steps.

### Save Custom Resources
{{< warning >}}
When you manually upgrade NGINX Service Mesh, all of your Custom Resources will be deleted. This includes TrafficSplits, TrafficTargets, RateLimits, and so on.
{{< /warning>}}

Before you proceed with the upgrade, run the commands shown below to back up your Custom Resources.

```bash
kubectl get trafficsplits.split.smi-spec.io -A -o yaml > trafficsplits.yaml
kubectl get traffictargets.access.smi-spec.io -A -o yaml > traffictargets.yaml
kubectl get httproutegroups.specs.smi-spec.io -A -o yaml > httproutegroups.yaml
kubectl get tcproutes.specs.smi-spec.io -A -o yaml > tcproutes.yaml
kubectl get ratelimits.specs.smi.nginx.com -A -o yaml > ratelimits.yaml
kubectl get circuitbreakers.specs.smi.nginx.com -A -o yaml > circuitbreakers.yaml
```

### Remove NGINX Service Mesh
Remove NGINX Service Mesh using your existing version of `nginx-meshctl`.

```bash
nginx-meshctl remove
```

At this point, you can discard your old version of `nginx-meshctl`.

### Redeploy NGINX Service Mesh
[Download and install]({{< ref "/get-started/install.md" >}}) the newer version of `nginx-meshctl`.

[Redeploy]({{< ref "/get-started/install.md#install-the-nginx-service-mesh-control-plane" >}}) NGINX Service Mesh.

Use the backups you made earlier to recreate your custom resources:

```bash
kubectl create -f trafficsplits.yaml -f traffictargets.yaml -f httproutegroups.yaml -f tcproutes.yaml -f ratelimits.yaml -f circuitbreakers.yaml
```

If your applications support rolling updates, re-roll using the following command:

```bash
kubectl rollout restart <resource type>/<resource name>
```

Otherwise, the application Pods need to be deleted and re-created.

## Upgrade to 1.7.0 in OpenShift

Due to changes in the CSI Driver in version 1.7.0 of NGINX Service Mesh for OpenShift, a manual upgrade is necessary to ensure volumes are properly unmounted and remounted.

{{< see-also >}}
For more context on OpenShift and the CSI Driver, see the [OpenShift Considerations]({{< ref "/get-started/openshift-platform/considerations" >}}).
{{< /see-also >}}

### 1. Save Custom Resources
{{< warning >}}
When you manually upgrade NGINX Service Mesh, all of your Custom Resources will be deleted. This includes TrafficSplits, TrafficTargets, RateLimits, and so on.
{{< /warning>}}

Before you proceed with the upgrade, run the commands shown below to back up your Custom Resources.

```bash
kubectl get trafficsplits.split.smi-spec.io -A -o yaml > trafficsplits.yaml
kubectl get traffictargets.access.smi-spec.io -A -o yaml > traffictargets.yaml
kubectl get httproutegroups.specs.smi-spec.io -A -o yaml > httproutegroups.yaml
kubectl get tcproutes.specs.smi-spec.io -A -o yaml > tcproutes.yaml
kubectl get ratelimits.specs.smi.nginx.com -A -o yaml > ratelimits.yaml
kubectl get circuitbreakers.specs.smi.nginx.com -A -o yaml > circuitbreakers.yaml
```

### 2. Remove NGINX Service Mesh

#### nginx-meshctl users

Remove NGINX Service Mesh using your existing version of `nginx-meshctl`.

```bash
nginx-meshctl remove
```

At this point, you can discard your old version of `nginx-meshctl`.

#### helm users

Uninstall/delete the release `nsm` in the `nginx-mesh` namespace:

```bash
helm uninstall nsm --namespace nginx-mesh
```

Change the release or namespace names as necessary for your deployment.

### 3. Remove sidecars from applications

The CSI volumes must be unmounted from existing applications before redeploying the mesh.

If your applications support rolling updates, re-roll using the following command:

```bash
kubectl rollout restart <resource type>/<resource name>
```

Otherwise, the application Pods need to be deleted. The list of resources that need to be re-rolled are printed out when removing the mesh.

Once all applications have their sidecars removed, and the `csi-driver-sentinel` Job in the `nginx-mesh` namespace has been automatically deleted, you can deploy the new version of the mesh.

### 4. Redeploy NGINX Service Mesh

#### nginx-meshctl users

[Download and install]({{< ref "/get-started/install.md" >}}) the newer version of `nginx-meshctl`.

[Redeploy]({{< ref "/get-started/install.md#install-the-nginx-service-mesh-control-plane" >}}) NGINX Service Mesh.

#### helm users

```bash
helm repo update
helm install nsm nginx-stable/nginx-service-mesh  --namespace nginx-mesh --wait
```

### 5. Redeploy policies and sidecars

Use the backups you made earlier to recreate your custom resources:

```bash
kubectl create -f trafficsplits.yaml -f traffictargets.yaml -f httproutegroups.yaml -f tcproutes.yaml -f ratelimits.yaml -f circuitbreakers.yaml
```

If your applications support rolling updates, re-roll using the following command:

```bash
kubectl rollout restart <resource type>/<resource name>
```

Otherwise, the application Pods need to be re-created.
