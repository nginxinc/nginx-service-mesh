# NGINX Service Mesh

Before deploying NGINX Service Mesh, see the [Platform Guide](https://docs.nginx.com/nginx-service-mesh/get-started/kubernetes-platform/) to ensure your environment is properly configured. If [Persistent Storage](https://docs.nginx.com/nginx-service-mesh/get-started/kubernetes-platform/persistent-storage/) is not configured in your cluster, set the `mTLS.persistentStorage` field to `off`. Verify that no other service meshes exist in your Kubernetes cluster. It is advised to install NGINX Service Mesh in a dedicated namespace.

## Helm Installation and Configuration

For information on the configuration options and installation process when using Helm with NGINX Service Mesh, see the [Installation Guide](https://docs.nginx.com/nginx-service-mesh/get-started/install-with-helm/).

## Rancher users

When deploying NGINX Service Mesh via the Rancher Apps and Marketplace, the Helm value `rancher` is set to `true` by default. This value causes Pods in the `cattle-*`, `ingress-nginx`, and `cert-manager` namespaces to be ignored by the automatic sidecar injection webhook. If this behavior is not desired, the `rancher` value can be set to `false`, or the `injector.nsm.nginx.com/auto-inject` label can be manually removed from these namespaces.
