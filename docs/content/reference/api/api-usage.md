---
title: "Use the NGINX Service Mesh API"
date: 2020-02-20T19:44:09Z
toc: true
description: "Instructions for using the NGINX Service Mesh API."
weight: 100
categories: ["tasks"]
doctypes: ["beta"]
docs: "DOCS-702"
---

{{< warning >}}
The NGINX Service Mesh API is in beta release status and is not recommended for use in client-side automation.
The objects described here are subject to change as the API evolves. No guarantee of backwards compatibility is made.
{{< /warning >}}

## Overview

The Service Mesh API is a Kubernetes [APIService](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/api-service-v1/) that offers a way to view and manage resources within the mesh. It can be accessed through the Kubernetes API server or via the command line tool. 

{{< note >}}Refer to the [API documentation]({{< ref "nginx-mesh-api.md" >}}) for endpoint descriptions, request payloads, and response bodies.{{< /note >}}

## How it works

The `v1alpha1.nsm.nginx.com` `APIService` extends the Kubernetes API and claims the URL path `/apis/nsm.nginx.com/v1alpha1` in the Kubernetes API.
When a request is sent to the path `/apis/nsm.nginx.com/v1alpha1`, the Kubernetes [aggregation layer](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/) proxies the request to the `nginx-mesh-api` Service running in the NGINX Service Mesh namespace.
The `nginx-mesh-api` Pod runs an extension API server that implements the `APIService`, and handles the requests proxied from the Kubernetes aggregation layer. All authentication and authorization decisions are delegated to the Kubernetes API server.
For more information on how authentication and authorization works, see the [Authentication Flow](https://kubernetes.io/docs/tasks/extend-kubernetes/configure-aggregation-layer/#authentication-flow) section in the Kubernetes docs.

## NGINX Service Mesh API Resources

The following `APIResourceList` describes the resources and actions that are available through the NGINX Service Mesh API:

```json
{
  "kind": "APIResourceList",
  "apiVersion": "v1",
  "groupVersion": "nsm.nginx.com/v1alpha1",
  "resources": [
    {
      "name": "services",
      "singularName": "",
      "namespaced": false,
      "kind": "",
      "verbs": [
        "list"
      ]
    },
    {
      "name": "config",
      "singularName": "",
      "namespaced": false,
      "kind": "",
      "verbs": [
        "list",
        "patch"
      ]
    },
    {
      "name": "inject",
      "singularName": "",
      "namespaced": false,
      "kind": "",
      "verbs": [
        "create"
      ]
    }
  ]
}
```

This `APIResourceList` can be retrieved from the `/apis/nsm.nginx.com/v1alpha` endpoint. 

{{< important >}}
The Kubernetes API discovery role `system:discovery` allows read-only access to this endpoint. By default, the `system:discovery` role is bound to the Kubernetes group `system:authenticated`, which allows all authenticated users access to this endpoint. Some managed Kubernetes environments may also bind the `system:discovery` role to the Kubernetes group `system:unauthenticated`, which allows unauthenticated users access to this endpoint as well.
For more information on API discovery roles and how to check your cluster's configuration, see the Kubernetes [API discovery roles doc](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#discovery-roles).
{{< /important >}}

## Authentication and Authorization

In order to access the NGINX Service Mesh API, you must be authenticated with the Kubernetes API server and authorized to perform the action (for example, `list`, `create`, `patch`) on the `nsm.nginx.com` resource (for example, `config`, `services`, `inject`). 

You can find information about authenticating with the Kubernetes API server in the Kubernetes [Authenticating](https://kubernetes.io/docs/reference/access-authn-authz/authentication/) documentation.

In addition to authenticating, you must be authorized to access an `nsm.nginx.com` resource. The following `ClusterRole` contains all the permissions needed to perform the available actions on each resource in the `nsm.nginx.com` API group:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nsm-api-full-access
rules:
- apiGroups:
  - nsm.nginx.com
  resources:
  - services
  - config
  verbs:
  - list
- apiGroups:
  - nsm.nginx.com
  resources:
  - config
  verbs:
  - patch
- apiGroups:
  - nsm.nginx.com
  resources:
  - inject
  verbs:
  - create
```

You can also create a `ClusterRole` with a subset of these permissions if you do not want to grant a user or Pod full access to the NGINX Service Mesh API. For example, if you would like to allow a user to only list NGINX Service Mesh services, you can define the following `ClusterRole`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nsm-api-list-services
rules:
- apiGroups:
  - nsm.nginx.com
  resources:
  - services
  verbs:
  - list
```

For more information on how to configure authorization in Kubernetes, see their [Authorization Overview](https://kubernetes.io/docs/reference/access-authn-authz/authorization/) doc.

## Access the NGINX Service Mesh REST API

Before you can access the NGINX Service Mesh API in any of the manners listed below, you must be authenticated to the Kubernetes API server and authorized to access the resource. 
See the [Authentication and Authorization](#authentication-and-authorization) section for more details. 

### Kubectl proxy

You can access the NGINX Service Mesh REST API by running `kubectl` in proxy mode. This method is recommended by Kubernetes because it protects against man-in-the-middle attacks by locating and authenticating to the API server on behalf of the user.
For more information and directions on how to use `kubectl proxy`, see the [Using kubectl proxy](https://kubernetes.io/docs/tasks/administer-cluster/access-cluster-api/#using-kubectl-proxy) section in the Kubernetes docs.

Example:

```bash
kubectl proxy --port=8080 &

curl http://localhost:8080/apis/nsm.nginx.com/v1alpha1
```

### Direct access to Kubernetes API server

To access the NGINX Service Mesh REST API directly though the Kubernetes API server, you can provide the location and credentials of the Kubernetes API server directly to the HTTP client.
For more information and directions, see the [Without kubectl proxy](https://kubernetes.io/docs/tasks/administer-cluster/access-cluster-api/#without-kubectl-proxy) section in the Kubernetes docs.

Example:

```bash
APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
## If using kubectl version 1.24 or greater:
TOKEN=$(kubectl create token default)
## Otherwise:
TOKEN=$(kubectl get secret $(kubectl get serviceaccount default -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 --decode)

curl ${APISERVER}/apis/nsm.nginx.com/v1alpha1 --cacert {PATH_TO_CLUSTER_CA_CERT} -H "Authorization: Bearer $TOKEN"
```

### Use kubectl to send requests to the API

In some cases, shown below, you can use `kubectl` to get NGINX Service Mesh API resources.

To get the `APIResourceList`:

```bash
kubectl get --raw /apis/nsm.nginx.com/v1alpha1
```

To get the NGINX Service Mesh deployment configuration:

```bash
kubectl get --raw /apis/nsm.nginx.com/v1alpha1/config
```

To get the list of NGINX Service Mesh services:

```bash
kubectl get --raw /apis/nsm.nginx.com/v1alpha1/services
```

### Command Line Access
The `nginx-meshctl` command line tool acts as a wrapper around the Service Mesh API. You can use the `nginx-meshctl` CLI to access the API endpoints, which simplifies human interactions with the REST API. For automation purposes, you can also access the REST API programmatically.

Refer to the [CLI documentation]( {{< ref "nginx-meshctl.md">}} ) for more information on how to use `nginx-meshctl`.

### Programmatic Access

For programmatic access, we recommend using a [Kubernetes client SDK](https://kubernetes.io/docs/tasks/administer-cluster/access-cluster-api/#programmatic-access-to-the-api). 

## Example Use Cases

### View the Mesh State by using the REST API

You can use the REST API, `nginx-meshctl` command line tool, or `kubectl` to view the current state of the mesh.

- View the configuration of the mesh:

  - REST API endpoint: `/apis/nsm.nginx.com/v1alpha1/config`
  - kubectl command: `kubectl get --raw /apis/nsm.nginx.com/v1alpha1/config`
  - CLI command: `nginx-meshctl config`

- View the services participating in the mesh:

  - REST API endpoint: `/apis/nsm.nginx.com/v1alpha1/services`
  - kubectl command: `kubectl get --raw /apis/nsm.nginx.com/v1alpha1/services`
  - CLI command: `nginx-meshctl services`

### Modify the Mesh State by using the REST API

NGINX Service Mesh API is under active development, but provides an endpoint and PATCH method to update a subset of the deployment configuration. The API schema is described in the [patchConfig section](https://docs.nginx.com/nginx-service-mesh/reference/api/nginx-mesh-api/#operation/patchConfig).

You can PATCH the configuration of NGINX Service Mesh by sending a request to the REST API endpoint:

- REST API endpoint: `/apis/nsm.nginx.com/v1alpha1/config`

The supported patch operations are:

- `add`
- `remove`
- `replace`

There are a subset of fields and objects supported for add, remove, and replace. Refer to the PATCH config schema described above for the full reference, the following examples can be referred to for a quick start.

#### Example: Disable Automatic injection in All Namespaces

The payload shown below disables automatic injection of the sidecar proxy for all namespaces and enables it for only the "prod" and "staging" namespaces.

```json
[
    {
        "op": "replace",
        "field": {
            "isAutoInjectEnabled": false
        }
    },
    {
        "op": "add",
        "field": {
            "enabledNamespaces": ["prod", "staging"]
        }
    }
]
```

To `remove` all values from a list of strings, define the value as an empty list (using `replace` with an empty list will have the same effect). For example:

```json
{
    "op": "remove",
    "field": {
        "enabledNamespaces": []
    }
}
```

## Inject the Sidecar Proxy into Kubernetes Resources

You can use the CLI or the REST API to manually inject the sidecar proxy into a Kubernetes resource definition.

- CLI command: `nginx-meshctl inject`
- API endpoint: `/apis/nsm.nginx.com/v1alpha1/inject`

The NGINX Service Mesh supports injection for the following Kubernetes resources:

- Deployment
- DaemonSet
- StatefulSet
- ReplicaSet
- ReplicationController
- Job
- Pod

Requests to the `/apis/nsm.nginx.com/v1alpha1/inject` endpoint must include the following:

- `Content-Type: multipart/form-data` header
- a JSON or YAML file sent as a form field with a key name of `file` and the `Content-Type: octet-stream` header.
  
  Usage: `-F file=@my-app.json`

The endpoint also supports the following optional form fields:

- `ignoreIncomingPorts`: a string list of ports for the proxy to ignore for incoming traffic; with `Content-Type: text/plain`.
  
  Usage: `-F "ignoreIncomingPorts=80;type=text/plain"`

- `ignoreOutgoingPorts`: a string list of ports for the proxy to ignore for outgoing traffic; with `Content-Type: text/plain`.

  Usage: `-F "ignoreOutgoingPorts=90;type=text/plain"`

### Example cURL Requests for Sidecar Proxy Injection

{{< important >}}
Read the [Direct access to Kubernetes API server](#direct-access-to-kubernetes-api-server) before trying any of the following examples. 
{{< /important >}}

- Provide a JSON file for injection:

    ```bash
    curl https://{APISERVER}/apis/nsm.nginx.com/v1alpha1/inject -X POST -H "Authorization: Bearer $TOKEN" --insecure -H "Content-Type:multipart/form-data"  -F file=@my-app.json
    ```

- Provide a YAML file, ignore incoming requests for port 80, and ignore outgoing traffic on port 90:

    ```bash
    curl https://{KUBERNETES_APISERVER}/apis/nsm.nginx.com/v1alpha1/inject -X POST -H "Authorization: Bearer $TOKEN" --insecure -H "Content-Type:multipart/form-data"  -F file=@my-app.yaml
    -F "ignoreIncomingPorts=80;type=text/plain" -F "ignoreOutgoingPorts=90;type=text/plain"
    ```

- Ignore incoming requests on multiple ports (80, 90):

    ```bash
    curl https://{APISERVER}/apis/nsm.nginx.com/v1alpha1/inject -X POST -H "Authorization: Bearer $TOKEN" --insecure -H "Content-Type:multipart/form-data"  -F file=@my-app.yaml
    -F "ignoreIncomingPorts=80;type=text/plain" -F "ignoreIncomingPorts=90;type=text/plain"
    ```
  
### Internal Configuration API Endpoints

There are a few API endpoints that are used by the NGINX Service Mesh CLI and Helm jobs to communicate with the control plane.
The list below describes each endpoint, the CLI command and Helm job that calls them, and the Kubernetes permissions needed.

{{< important >}}
These endpoints are for internal use only. 
{{< /important >}}

{{< bootstrap-table "table table-striped table-bordered">}}
|Resource Name|REST API Endpoint                       |Description                                                                                                      |CLI Command|Helm Job                |Permissions                             |
|-------------|----------------------------------------|-----------------------------------------------------------------------------------------------------------------|-----------|------------------------|----------------------------------------|
|clear        |`/apis/nsm.nginx.com/v1alpha1/clear`    |a POST request to `/clear` turns all NGINX Service Mesh sidecars transparent                                     |remove     |turn-proxies-transparent|create clear in APIGroup nsm.nginx.com  |
|version      |`/apis/nsm.nginx.com/v1alpha1/version`  |a GET request to `/version` returns the versions of the control plane components and the sidecars                |version    |N/A                     |list version in APIGroup nsm.nginx.com  |

{{< /bootstrap-table >}}
