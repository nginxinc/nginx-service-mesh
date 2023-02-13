---
title: "Sidecar Proxy Injection"
date: 2020-08-24T11:46:19-06:00
toc: true
description: "Learn about the configuration options for NGINX Service Mesh sidecar proxy injection."
weight: 10
categories: ["tasks"]
docs: "DOCS-692"
---

## Overview

NGINX Service Mesh works by injecting a sidecar proxy into Kubernetes resources. 
You can choose to inject the sidecar proxy into the YAML or JSON definitions for your Kubernetes resources in the following ways:

- [Automatic Injection](#automatic-proxy-injection)
- [Manual Injection](#manual-proxy-injection)

{{< note >}}
When you inject the sidecar proxy into a Kubernetes resource, the injected config uses the global mTLS setting. 
You can define the global setting when you deploy NGINX Service Mesh, or use the default setting.

Refer to [Secure Mesh Traffic using mTLS]({{< ref "/guides/secure-traffic-mtls.md" >}}) for more information.
{{< /note >}}

{{< important >}}
The sidecar proxy will not be injected into Pods that define multiple container ports with the same port number or for container ports with the SCTP protocol.

UDP and TCP is an exception to this, and may be specified on the same port.
{{< /important >}}

The mesh supports the following Kubernetes resources and API versions for injection:

{{% table %}}
|  Resource Type        | API Version |
|-----------------------|-------------|
| Deployment            | apps/v1     |
| DaemonSet             | apps/v1     |
| StatefulSet           | apps/v1     |
| ReplicaSet            | apps/v1     |
| ReplicationController | v1          |
| Pod                   | v1          |
| Job                   | batch/v1    |
{{% /table %}}

## Automatic Proxy Injection

NGINX Service Mesh uses automatic injection by default. This means that any time a user creates a Kubernetes Pod resource, the NGINX Service Mesh automatically injects the sidecar proxy into the Pod. Automatic injection applies to all namespaces in your Kubernetes cluster.

### Enable or Disable Automatic Proxy Injection by Namespace

By default, NGINX Service Mesh can access resources in all Kubernetes namespaces.

To disable this setting, deploy the mesh using the `--disable-auto-inject` flag:

```bash
nginx-meshctl deploy ... --disable-auto-inject
```

To enable injection for a specific namespace, add the `injector.nsm.nginx.com/auto-inject=enabled` label. This will only work if the mesh is deployed with global automatic sidecar injection disabled.

{{< note >}}
If you add this label to a namespace where Pods already exist you will need to restart those Pods for the sidecar to be injected.
By the same token if you remove this label from a namespace where Pods exist and have the sidecar injected, you will need to restart them to remove the sidecar.
{{< /note >}}

You can also enable injection by adding the `--enabled-namespaces` flag to your deploy command.

For example, to disable automatic injection in all namespaces and enable it only in the namespaces "prod" and "staging", you would run the following command:

```bash
nginx-meshctl deploy ... --disable-auto-inject --enabled-namespaces="prod,staging"
```

{{< note >}}
If you need to [modify the auto injection settings]( {{< ref "api-usage.md#modify-the-mesh-state-by-using-the-rest-api" >}} ) after you've deployed NGINX Service Mesh, you can do so by using the REST API.
{{< /note >}}

### Enable or Disable Automatic Proxy Injection on a Resource

For more granular control, you can override the global automatic injection setting on a per-resource basis. To do so, add the following label to the resource's **PodTemplateSpec**:

`injector.nsm.nginx.com/auto-inject: "enabled|disabled"`

### Injecting Sidecar into Existing Resources

To inject the sidecar into existing resources you must re-roll those resources after installing NGINX Service Mesh.

```bash
kubectl rollout restart <resource type>/<resource name>
```

For example:

```bash
kubectl rollout restart deployment/frontend
```

{{< note >}}
Refer to [NGINX Service Mesh Labels and Annotations]( {{< ref "/get-started/configuration.md#supported-labels-and-annotations" >}}) for more information.
{{< /note >}}

{{< see-also >}}
Refer to the Kubernetes [`kubectl` Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#updating-resources) documentation for more information about rolling resources.
{{< /see-also >}}

## Manual Proxy Injection

Before running the `inject` command, make sure your Kubernetes user has the permission to `create` the resource `inject` in APIGroup `nsm.nginx.com`. Below is an example of a `ClusterRole` with this permission:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nsm-inject
rules:
- apiGroups:
  - nsm.nginx.com
  resources:
  - inject
  verbs:
  - create
```

{{< note >}}
If your Kubernetes user account has the `ClusterAdmin` role, then no additional permissions are necessary to run the inject command.
{{< /note >}}

To inject the sidecar proxy into a resource manually, use the `nginx-meshctl inject` command. Provide the path to the resource definition file and your desired output filename.

```bash
nginx-meshctl inject < <resource-file> > <new-resource-file>
```

For example, the following command will write the updated config for "resource.yaml" to a new file, "resource-injected.yaml":

```bash
nginx-meshctl inject < resource.yaml > resource-injected.yaml
```

Depending on the network connection and the size of the file you're injecting, timeouts may occur while running the inject command. If this happens, you can use the `--timeout` flag to increase the timeout. The default timeout is 5 seconds.

```bash
nginx-meshctl --timeout 10s inject < resource.yaml > resource-injected.yaml
```

## Ignore Specific Ports

You can set the proxy to ignore ports for either incoming or outgoing traffic. The NGINX Service Mesh applies the configurations at injection time.

- For automatic injection, add the following annotations to the **PodTemplateSpec** in your resource definition:

  ```yaml
  config.nsm.nginx.com/ignore-incoming-ports: "port1, port2, ..., portN"
  config.nsm.nginx.com/ignore-outgoing-ports: "port1, port2, ..., portN"
  ```

- For manual injection, you can use the annotations above or specify the ports when running the `nginx-meshctl inject` command.

    ```bash
    nginx-meshctl inject --ignore-incoming-ports "port1,port2,...,portN", --ignore-outgoing-ports "port1,port2,...,portN" < resource.yaml > resource-injected.yaml
    ```
  
{{< note >}}
Refer to [NGINX Service Mesh Annotations]( {{< ref "/get-started/configuration.md#pod-annotations" >}}) for more information around annotations.
{{< /note >}}

## Default Ignored Ports

By default, the following ports are ignored by the proxy:

- 53 (DNS)

## HTTPGet Health Probe Rewrite

If mTLS mode is set to `strict`, then readiness, liveness, and startup probes using HTTP GET do not work. This is
due to `kubelet` not having the correct client certificates. To remedy this, application HTTP/S health probes are
rewritten at injection time. The new probes point to an endpoint on the sidecar proxy, which
redirects the health check to the original destination on the application. This allows the health check to bypass 
the SSL verification, while still sending the health check to the intended destination.

## UDP MTU Sizing

NGINX Service Mesh automatically detects and adjusts the `eth0` interface to support the 32 bytes of space required for PROXY Protocol V2. See the [UDP and eBPF architecture]( {{< ref "architecture.md#udp-and-ebpf" >}} ) section for more information.
