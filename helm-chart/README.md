# NGINX Service Mesh

Before deploying NGINX Service Mesh, see the [Platform Guide](https://docs.nginx.com/nginx-service-mesh/get-started/platform-setup/) to ensure your environment is properly configured.
If [Persistent Storage](https://docs.nginx.com/nginx-service-mesh/get-started/platform-setup/persistent-storage/) is not configured in your cluster, set the `mTLS.persistentStorage` field to `off`.
Verify that no other service meshes exist in your Kubernetes cluster. It is advised to install NGINX Service Mesh in a dedicated namespace.

## Helm Installation and Configuration

For information on the configuration options and installation process when using Helm with NGINX Service Mesh, see the [Installation Guide](https://docs.nginx.com/nginx-service-mesh/get-started/install-with-helm/).

To enable automatic sidecar injection, add the following label to a namespace: `injector.nsm.nginx.com/auto-inject=enabled`.
