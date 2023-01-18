# NGINX Service Mesh

Before deploying NGINX Service Mesh, see the [Platform Guide](https://docs.nginx.com/nginx-service-mesh/get-started/kubernetes-platform/) to ensure your environment is properly configured. If [Persistent Storage](https://docs.nginx.com/nginx-service-mesh/get-started/kubernetes-platform/persistent-storage/) is not configured in your cluster, set the `mTLS.persistentStorage` field to `off`. Verify that no other service meshes exist in your Kubernetes cluster. It is advised to install NGINX Service Mesh in a dedicated namespace.

## Helm Installation and Configuration

For information on the configuration options and installation process when using Helm with NGINX Service Mesh, see the [Installation Guide](https://docs.nginx.com/nginx-service-mesh/get-started/install-with-helm/).

We recommend deploying the mesh with auto-injection disabled globally, using the `--set disableAutoInjection=true` flag. This ensures that Pods are not automatically injected without your consent, especially in system namespaces.

To opt-in a namespace you can label it with `injector.nsm.nginx.com/auto-inject=enabled` or use the flag `--set autoInjection.enabledNamespaces={namespace-1, namespace-2}`.
