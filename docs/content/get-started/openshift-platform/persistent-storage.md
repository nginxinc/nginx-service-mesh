---
title: Persistent Storage
description: Learn how to set up persistent storage for use with NGINX Service Mesh.
categories: ["tasks"]
weight: 101
toc: true
docs: "DOCS-690"
---

OpenShift's persistent storage mechanisms work exactly the same as in Kubernetes. See the Kubernetes Platform [Persistent Storage]({{< ref "/get-started/kubernetes-platform/persistent-storage.md" >}}) page for more information on setting up persistent storage in your environment.

When using a managed environment such as Azure Red Hat OpenShift, a default StorageClass can be utilized. However, for something local such as CodeReady Containers, you will need to either provision your own persistent storage or disable it with `--persistent-storage off` when deploying NGINX Service Mesh.
