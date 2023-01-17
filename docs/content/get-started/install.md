---
title: "Install NGINX Service Mesh using nginx-meshctl"
date: 2020-02-20T19:43:59Z
draft: false
toc: true
description: "This topic explains how to download, install, and deploy NGINX Service Mesh."
weight: 300
categories: ["tasks"]
docs: "DOCS-681"
---

## Overview

This guide contains instructions for downloading and installing NGINX Service Mesh using the `nginx-meshctl` command line tool.

For Helm users, see how to [Install NGINX Service Mesh using Helm]( {{< ref "/get-started/install-with-helm.md" >}} ).

### Prerequisites

{{< important >}} Before installing NGINX Service Mesh, make sure you've completed the following steps. {{< /important >}}

- You have a working and [supported]({{< ref "/about/tech-specs.md#supported-versions" >}}) Kubernetes cluster.
- You followed the [Kubernetes]( {{< ref "/get-started/kubernetes-platform/_index.md" >}} ) or [OpenShift]( {{< ref "/get-started/openshift-platform/_index.md" >}} ) Platform Setup guide to **prepare your cluster** to work with NGINX Service Mesh.
- You have the Kubernetes `kubectl` command-line utility configured on the machine where you want to install NGINX Service Mesh.
- You reviewed the [Configuration Options for NGINX Service Mesh]( {{< ref "/get-started/configuration.md" >}} ).

### Download nginx-meshctl

The NGINX Service Mesh command-line tool -- `nginx-meshctl` -- allows you to deploy, remove, and interact with the NGINX Service Mesh control plane.

To install NGINX Service Mesh, you need to download the `nginx-meshctl` binary for your architecture. The latest version of `nginx-meshctl` is available on our [Github releases](https://github.com/nginxinc/nginx-service-mesh/releases/latest) page.

### Install the CLI

The following sections describe how to install the CLI on Linux, macOS, and Windows.

#### Install on Linux

1. Download the appropriate binary for your architecture, either `nginx-meshctl_<version>_linux_amd64.tar.gz` or `nginx-meshctl_<version>_linux_arm64.tar.gz`.

1. Unzip the binary.

    ```bash
    tar -xvf nginx-meshctl_<version>_linux_amd64.tar.gz nginx-meshctl
    ```

1. Move the `nginx-meshctl` executable in to your PATH.

    ```bash
    sudo mv nginx-meshctl /usr/local/bin/nginx-meshctl
    ```

1. Ensure the `nginx-meshctl` is executable.

    ```bash
    sudo chmod +x /usr/local/bin/nginx-meshctl
    ```

1. Test the installation.

    ```bash
    nginx-meshctl version
    ```

#### Install on macOS

1. Download the appropriate binary for your architecture, either `nginx-meshctl_<version>_darwin_arm64.tar.gz` for M1 Macs or `nginx-meshctl_<version>_darwin_amd64.tar.gz` for Intel based Macs.

1. Unzip the binary.

    ```bash
    tar -xvf nginx-meshctl_<version>_darwin_amd64.tar.gz nginx-meshctl
    ```

1. Move the `nginx-meshctl` executable in to your PATH.

    ```bash
    sudo mv nginx-meshctl /usr/local/bin/nginx-meshctl
    ```

1. Ensure the `nginx-meshctl` is executable.

    ```bash
    sudo chmod +x /usr/local/bin/nginx-meshctl
    ```

1. Test the installation.

    ```bash
    nginx-meshctl version
    ```

#### Install on Windows

1. Download the appropriate binary for your architecture, either `nginx-meshctl_<version>_windows_amd64.zip` or `nginx-meshctl_<version>_windows_arm64.zip`.

1. Extract the binary, `nginx-meshctl.exe`, from the zip file.

1. Add the binary to your PATH and rename.

1. Test the installation.

    ```bash
    nginx-meshctl.exe version
    ```

### Images

NGINX Service Mesh distributes a number of images and pulls additional publicly-accessible third-party container images into your Kubernetes cluster in order to function. For a full list refer to the [Technical Specifications]( {{< ref "/about/tech-specs.md#images" >}} ).

#### Manually Download and Push Images to Container Registry

NGINX Service Mesh images are pulled in automatically when deploying the mesh. However, if desired, you can manually download and push them to your own container registry that your cluster can access.

{{< important >}}
To ensure compatibility, the image versions need to match the version of `nginx-meshctl` that you downloaded. If using Helm, the image versions need to match the chart's `appVersion`.
{{< /important >}}

When deploying NGINX Service Mesh using `nginx-meshctl`, set the `--registry-server` flag to your registry. If using Helm, set the `registry.server` field to your registry.

#### Air Gap Environment

If your environment does not have access to public image repositories, then you will need to manually pull the images from this [list]( {{< ref "/about/tech-specs.md#images" >}} ), and push them to your [private registry]( {{< ref "/guides/private-registry.md" >}} ). The image names and tags must remain the same. For example:

 `gcr.io/spiffe-io/spire-agent:1.5.4` would become `your-registry/spire-agent:1.5.4`
 
 `nats:2.9-alpine` would become `your-registry/nats:2.9-alpine`

When running `nginx-meshctl deploy`, use the `--disable-public-images` flag to instruct the mesh to use your `--registry-server` for all images. 
For example:

```bash
nginx-meshctl deploy --registry-server your-registry --disable-public-images ...
```

## Install the NGINX Service Mesh Control Plane

{{< see-also >}}
Check out [Getting Started with NGINX Service Mesh]({{< ref "/get-started/configuration.md" >}}) to learn about the deployment options before proceeding.  
You can find the full list of options in the [`nginx-meshctl` Reference]( {{< ref "nginx-meshctl.md" >}} ).
{{< /see-also >}}

{{< important >}}
`nginx-meshctl` creates the namespace for the NGINX Service Mesh control plane.  
This namespace is dedicated to the NGINX Service Mesh control plane and **should not be used for anything else**.  
If desired, you can specify any name for the namespace via the `--namespace` argument, but do not create this namespace yourself.
{{< /important >}}

Take the steps below to install the NGINX Service Mesh control plane.

1. Run the `nginx-meshctl deploy` command using the desired [options]({{< ref "nginx-meshctl.md#deploy" >}}).

   For example, the following command will deploy NGINX Service Mesh using all of the default settings for the latest release:

    ```bash
    nginx-meshctl deploy
    ```

   {{< note >}}
   We recommend deploying the mesh with auto-injection disabled globally, using the `--disable-auto-inject` flag. This ensures that Pods are not automatically injected without your consent, especially in system namespaces.
   You can opt-in the namespaces where you would like auto-injection enabled using `--enabled-namespaces "namespace-1,namespace-2"` or by labeling a namespace with `injector.nsm.nginx.com/auto-inject=enabled`.
   {{< /note >}}

    If you are using a private registry to store the NGINX Service Mesh images see the [Private Registry]( {{< ref "/guides/private-registry.md" >}} ) guide for instructions. 

    For example, `nginx-meshctl deploy --registry-server "registry:5000/images" --image-tag X.Y.Z` will look for containers `registry:5000/images/nginx-mesh-api:X.Y.Z`, `registry:5000/images/nginx-mesh-sidecar:X.Y.Z`, and so on.



2. Run the `kubectl get pods` command to verify that the Pods are up and running.

    Be sure to specify the `nginx-mesh` namespace when running the command.

    ```bash
    $ kubectl -n nginx-mesh get pods
    NAME                                  READY   STATUS    RESTARTS   AGE
    nats-server-84f8b6f669-xszkc          1/1     Running   0          14m
    nginx-mesh-api-954467945-sc7qh        1/1     Running   0          14m
    nginx-mesh-metrics-57464df46d-qskd2   1/1     Running   0          14m
    spire-agent-92ktv                     1/1     Running   0          15m
    spire-agent-9dbn6                     1/1     Running   0          15m
    spire-agent-z5cq6                     1/1     Running   0          15m
    spire-server-0                        2/2     Running   0          15m
    ```

    {{< note >}} If running in OpenShift, you will see two pods per Spire Agent container. {{< /note >}}

## UDP MTU Sizing

{{< note >}}
UDP traffic proxying is turned off by default. You can activate it at deploy time using the `--enable-udp` flag. Linux kernel 4.18 or greater is required.
{{< /note >}}

NGINX Service Mesh automatically detects and adjusts the `eth0` interface to support the 32 bytes of space required for PROXY Protocol V2. See the [UDP and eBPF architecture]({{< ref "architecture.md#udp-and-ebpf" >}}) section for more information.

NGINX Service Mesh does not detect changes made to the MTU in the pod at runtime. If adding a CNI changes the MTU of the `eth0` interface of running pods, you should re-roll the affected pods to ensure those changes take place.
