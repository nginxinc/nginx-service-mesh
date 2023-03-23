---
title: "Uninstall NGINX Service Mesh"
draft: false
toc: true
description: "This topic explains how to uninstall NGINX Service Mesh."
weight: 200
categories: ["tasks"]
docs: "DOCS-699"
---

## Overview

This document contains instructions for uninstalling NGINX Service Mesh.

For Helm users, see [how to uninstall using Helm]( {{< ref "/get-started/install-with-helm.md#uninstalling-the-chart" >}} ).

For OpenShfit users, removal behaves differently in order to ensure all injected Pods are serviced. See the [Remove]({{< ref "/get-started/openshift-platform/considerations.md#remove" >}}) section of the OpenShift Considerations document for more information.

Uninstalling does the following:

1. Removes the control plane and its contents from Kubernetes.
2. Deletes all NGINX Service Mesh traffic policies.

The `nginx-meshctl` command-line utility prints a list of resources that contain the sidecar proxies when the uninstall completes. You must re-roll the Deployments in Kubernetes to remove the sidecars. Until you re-roll the resources, the sidecar proxies still exist, but they don't apply any rules to the traffic.

## Uninstall the Control Plane

To uninstall the Service Mesh control plane using the `nginx-meshctl` command-line utility, run the command shown below.

```bash
nginx-meshctl remove
```

When prompted for confirmation, specify `y` or `n`.

{{< tip >}}
If you want to skip the confirmation prompt, add the `-y` flag as shown in the example below.

```bash
nginx-meshctl remove -y
```

{{< /tip >}}

{{< note >}}
If the removal process gets stuck or fails to clean up all resources, you can manually view all NGINX Service Mesh resources that were left over using the following command:

```bash
kubectl api-resources --verbs=list -o name | xargs -n 1 kubectl get --show-kind --ignore-not-found -l app.kubernetes.io/part-of=nginx-service-mesh -A
```

These resources can be manually removed if necessary.
{{< /note >}}

## Remove the Sidecar Proxy from Deployments

If your resources support Rolling Updates (Deployments, DaemonSets, and StatefulSets), run the following `kubectl` command for each resource to complete the uninstall.

```bash
kubectl rollout restart <resource type>/<resource name>
```

For example:

```bash
kubectl rollout restart deployment/frontend
```

{{< note >}}
If you want to redeploy NGINX Service Mesh after removing it, you need to re-roll the resources after the new NGINX Service Mesh is installed. Sidecars from an earlier NGINX Service Mesh installation won't work with a new installation.
{{< /note >}}

{{< see-also >}}
Refer to the Kubernetes [`kubectl` Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#updating-resources) documentation for more information about rolling resources.
{{< /see-also >}}
