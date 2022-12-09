---
title: Persistent Storage
description: Learn how to set up persistent storage for use with NGINX Service Mesh.
categories: ["tasks"]
weight: 101
toc: true
docs: "DOCS-686"
---

NGINX Service Mesh generates data that needs to persist across restarts and failures to ensure uninterrupted operations. For example, if the SPIRE Server restarts, the new instance can pick up the the existing database of identities without having to rebuild everything. Depending on the environment, persistent storage may already be set up and ready for use by NGINX Service Mesh.

The big three hosted Kubernetes environments (Elastic Kubernetes Service (EKS), Azure Kubernetes Service (AKS), and Google Kubernetes Engine (GKE)) all have built-in persistent storage that NGINX Service Mesh will automatically pick up and use.

## Determining Persistent Storage on your Cluster

NGINX Service Mesh will automatically use the default Kubernetes `StorageClass` if it's configured.

```bash
$ kubectl get storageclass
NAME                 PROVISIONER            AGE
standard (default)   kubernetes.io/gce-pd   153d
```

In the above output from GKE, NGINX Service Mesh will use the `standard` `StorageClass` to persist data. It's possible to have multiple Storage Classes configured, in that case NGINX Service Mesh will use the one configured as default. Specifying a specific `StorageClass` to use isn't supported at this time.

## Deploying Without Persistent Storage

If there is no default `StorageClass` set up, NGINX Service Mesh will still work, but will print the below warning during installation:

```text
Warning: Deploying without persistent storage, not suitable for production environments.
         For production environments ensure a default StorageClass is set.
```

Without persistent storage, if SPIRE Server restarts for any reason, the entire identity database will need to be rebuilt, which will significantly increase time to recovery.

## Setting up Persistent Storage 

Kubernetes has an extensive ecosystem of plugins for persistent storage. These range from vSphere Volumes to Amazon Web Services (AWS) Elastic Block Store. For more details refer to the [Kubernetes Storage Classes documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/).

{{< important >}}
Based on our testing, NFS Storage Classes introduce too much latency and aren't recommended for use with NGINX Service Mesh.
{{< /important >}}


## Troubleshooting

By default NGINX Service Mesh detects if a default StorageClass is configured and uses it if present. If the StorageClass is misconfigured NGINX Service Mesh will fail to start. If you suspect persistent storage is misconfigured, try to deploy NGINX Service Mesh with `--persistent-storage off`.
