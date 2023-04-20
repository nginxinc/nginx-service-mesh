---
	title: CLI Reference
	description: "Man page and instructions for using the NGINX Service Mesh CLI"
	draft: false
	weight: 300
	toc: true
	categories: ["reference"]
	docs: "DOCS-704"
	---
## nginx-meshctl

nginx-meshctl is the CLI utility for the NGINX Service Mesh control plane

### Synopsis

nginx-meshctl is the CLI utility for the NGINX Service Mesh control plane. 
Requires a connection to a Kubernetes cluster via a kubeconfig.

```txt
nginx-meshctl [flags]
```

### Options

```txt
  -h, --help                help for nginx-meshctl
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Completion

```txt
  nginx-meshctl completion [command]
```

### Available Commands

```txt
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh
```

### Options

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```

## Config


Display the NGINX Service Mesh configuration

### Synopsis

Display the NGINX Service Mesh configuration.

```txt
nginx-meshctl config [flags]
```

### Options

```txt
  -h, --help   help for config
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Deploy


Deploys NGINX Service Mesh into your Kubernetes cluster

### Synopsis

Deploy NGINX Service Mesh into your Kubernetes cluster.
This command installs the following resources into your Kubernetes cluster by default:

- NGINX Mesh Controller: The Control Plane for the Service Mesh.
- NGINX Metrics API: SMI-formatted metrics.
- SPIRE: mTLS service-to-service communication.
- NATS: Message bus.



```txt
nginx-meshctl deploy [flags]
```

### Examples

```txt

  Most of the examples below are abbreviated for convenience. The '...' in these 
  examples represents the image references. Be sure to include the image references 
  when running the deploy command.  
	
  - Deploy the latest version of NGINX Service Mesh, using default values, from your container registry:
		
      nginx-meshctl deploy --registry-server "registry:5000"
	
  - Deploy the Service Mesh in namespace "my-namespace":

      nginx-meshctl deploy ... --namespace my-namespace

  - Deploy the Service Mesh with mTLS turned off:

      nginx-meshctl deploy ... --mtls-mode off

  - Deploy the Service Mesh and enable telemetry traces to be exported to your OTLP gRPC collector running in your Kubernetes cluster:
      
      nginx-meshctl deploy ... --telemetry-exporters "type=otlp,host=otel-collector.my-namespace.svc.cluster.local,port=4317"

  - Deploy the Service Mesh with upstream certificates and keys for mTLS:

      nginx-meshctl deploy ... --mtls-upstream-ca-conf="disk.yaml"

```

### Options

```txt
      --access-control-mode string        default access control mode for service-to-service communication
                                          		Valid values: allow, deny (default "allow")
      --client-max-body-size string       NGINX client max body size (default "1m")
      --disable-public-images             don't pull third party images from public repositories
      --enable-udp                        enable UDP traffic proxying (beta); Linux kernel 4.18 or greater is required
      --environment string                environment to deploy the mesh into
                                          		Valid values: kubernetes, openshift (default "kubernetes")
  -h, --help                              help for deploy
      --image-tag string                  tag used for pulling images from registry
                                          		Affects: nginx-mesh-cert-reloader, nginx-mesh-controller, nginx-mesh-init, nginx-mesh-metrics, nginx-mesh-sidecar (default "2.0.0")
      --mtls-ca-key-type string           the key type used for the SPIRE Server CA
                                          		Valid values: ec-p256, ec-p384, rsa-2048, rsa-4096 (default "ec-p256")
      --mtls-ca-ttl string                the CA/signing key TTL in hours(h). Min value 24h. Max value 999999h. (default "720h")
      --mtls-mode string                  mTLS mode for pod-to-pod communication
                                          		Valid values: off, permissive, strict (default "permissive")
      --mtls-svid-ttl string              the TTL of certificates issued to workloads in hours(h) or minutes(m). Max value is 999999. (default "1h")
      --mtls-trust-domain string          the trust domain of the NGINX Service Mesh (default "example.org")
      --mtls-upstream-ca-conf string      the upstream certificate authority configuration file
      --nginx-error-log-level string      NGINX error log level
                                          		Valid values: debug, info, notice, warn, error, crit, alert, emerg (default "warn")
      --nginx-lb-method string            NGINX load balancing method
                                          		Valid values: least_conn, least_time, least_time last_byte, least_time last_byte inflight, random, random two, random two least_conn, random two least_time, random two least_time=last_byte, round_robin (default "least_time")
      --nginx-log-format string           NGINX log format
                                          		Valid values: default, json (default "default")
      --persistent-storage string         use persistent storage. "auto" will enable persistent storage if a default StorageClass exists
                                          		Valid values: auto, off, on (default "auto")
      --prometheus-address string         the address of a Prometheus server deployed in your Kubernetes cluster
                                          		Address should be in the format <service-name>.<namespace>:<service-port>
      --registry-key string               path to JSON Key file for accessing private GKE registry
                                          		Cannot be used with --registry-username or --registry-password
      --registry-password string          password for accessing private registry
                                          		Requires --registry-username to be set. Cannot be used with --registry-key
      --registry-server string            hostname:port (if needed) for registry and path to images
                                          		Affects: nginx-mesh-cert-reloader, nginx-mesh-controller, nginx-mesh-init, nginx-mesh-metrics, nginx-mesh-sidecar (default "docker-registry.nginx.com/nsm")
      --registry-username string          username for accessing private registry
                                          		Requires --registry-password to be set. Cannot be used with --registry-key
      --spire-server-key-manager string   storage logic for SPIRE Server's private keys
                                          		Valid values: disk, memory (default "disk")
      --telemetry-exporters stringArray   list of telemetry exporter key-value configurations
                                          		Format: "type=<exporter_type>,host=<exporter_host>,port=<exporter_port>".
                                          		Type, host, and port are required. Only type "otlp" exporter is supported.
      --telemetry-sampler-ratio float32   the percentage of traces that are processed and exported to the telemetry backend.
                                          		Float between 0 and 1 (default 0.01)
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Inject


Inject the NGINX Service Mesh sidecars into Kubernetes resources

### Synopsis

Inject the NGINX Service Mesh sidecar into Kubernetes resources.

- Accepts JSON and YAML formats.
- Outputs JSON or YAML resources with injected sidecars to stdout.

```txt
nginx-meshctl inject [flags]
```

### Examples

```txt

  - Inject the resources in my-app.yaml and create in Kubernetes:

      nginx-meshctl inject -f ./my-app.yaml | kubectl apply -f -

  - Inject the resources passed into stdin and write the changes to the same file:

      nginx-meshctl inject < ./my-app.json > ./my-injected-app.json

  - Inject the resources in my-app.yaml and configure proxies to ignore ports 1433 and 1434 for outgoing traffic:

      nginx-meshctl inject --ignore-outgoing-ports 1433,1434 -f ./my-app.yaml

  - Inject the resources passed into stdin and configure proxies to ignore port 1433 for incoming traffic:

      nginx-meshctl inject --ignore-incoming-ports 1433 < ./my-app.json 

```

### Options

```txt
  -f, --file string                  the filename that contains the resources you want to inject
                                     		If no filename is provided, input will be taken from stdin
  -h, --help                         help for inject
      --ignore-incoming-ports ints   ports to ignore for incoming traffic
      --ignore-outgoing-ports ints   ports to ignore for outgoing traffic
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Remove


Remove NGINX Service Mesh from your Kubernetes cluster

### Synopsis

Remove the NGINX Service Mesh from your Kubernetes cluster.

- Removes the resources created by the deploy command from the Service Mesh namespace (default: "nginx-mesh").
- You will need to clean up all resources containing injected sidecar proxies manually.


```txt
nginx-meshctl remove [flags]
```

### Examples

```txt

  - Remove the NGINX Service Mesh from the default namespace ('nginx-mesh'):
		
      nginx-meshctl remove
	
  - Remove the NGINX Service Mesh from namespace 'my-namespace':
		
      nginx-meshctl remove --namespace my-namespace
		
  - Remove the NGINX Service Mesh without prompting the user to confirm removal:
	
      nginx-meshctl remove -y

```

### Options

```txt
  -h, --help   help for remove
  -y, --yes    answer yes for confirmation of removal
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Services


List the Services registered with NGINX Service Mesh

### Synopsis

List the Services registered with NGINX Service Mesh.

- Outputs the Services and their upstream addresses and ports.
- The list contains only those Services whose Pods contain the NGINX Service Mesh sidecar.


```txt
nginx-meshctl services [flags]
```

### Options

```txt
  -h, --help   help for services
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Status


Check connection to NGINX Service Mesh

### Synopsis

Check connection to NGINX Service Mesh.

```txt
nginx-meshctl status [flags]
```

### Options

```txt
  -h, --help   help for status
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Supportpkg


Create an NGINX Service Mesh support package

### Synopsis

Create an NGINX Service Mesh support package.

```txt
nginx-meshctl supportpkg [flags]
```

### Options

```txt
      --disable-sidecar-logs   disable the collection of sidecar logs
  -h, --help                   help for supportpkg
  -o, --output string          output directory for supportpkg tarball (default "/Users/<user>/NGINX/nginx-service-mesh")
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Top


Display traffic statistics

### Synopsis

Display traffic statistics.
Top provides information about the incoming and outgoing requests to and from a resource type or name.
Supported resource types are: Pods, Deployments, StatefulSets, DaemonSets, and Namespaces.

```txt
nginx-meshctl top [resource-type/resource] [flags]
```

### Examples

```txt

  - Display traffic statistics for all Deployments:
		
      nginx-meshctl top
  
  - Display traffic statistics for all Pods:
		
      nginx-meshctl top pods
  
  - Display traffic statistics for Deployment "my-app":
	 
      nginx-meshctl top deployments/my-app
```

### Options

```txt
  -h, --help               help for top
  -n, --namespace string   namespace where the resource(s) resides (default "default")
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




## Upgrade


Upgrade NGINX Service Mesh

### Synopsis

Upgrade NGINX Service Mesh to the latest version.
This command removes the existing NGINX Service Mesh while preserving user configuration data.
The latest version of NGINX Service Mesh is then deployed using that data.


```txt
nginx-meshctl upgrade [flags]
```

### Options

```txt
  -h, --help                     help for upgrade
      --image-tag string         tag used for pulling images from registry
                                 		Affects: nginx-mesh-cert-reloader, nginx-mesh-controller, nginx-mesh-init, nginx-mesh-metrics, nginx-mesh-sidecar
      --registry-server string   hostname:port (if needed) for registry and path to images
                                 		Affects: nginx-mesh-cert-reloader, nginx-mesh-controller, nginx-mesh-init, nginx-mesh-metrics, nginx-mesh-sidecar
  -t, --timeout duration         timeout when waiting for an upgrade to finish (default 5m0s)
  -y, --yes                      answer yes for confirmation of upgrade
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
```




## Version


Display NGINX Service Mesh version

### Synopsis

Display NGINX Service Mesh version.
Will contact the mesh for version and timeout if unable to connect.

```txt
nginx-meshctl version [flags]
```

### Options

```txt
  -h, --help   help for version
```

### Options inherited from parent commands

```txt
  -k, --kubeconfig string   path to kubectl config file (default "/Users/<user>/.kube/config")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "nginx-mesh")
  -t, --timeout duration    timeout when communicating with NGINX Service Mesh (default 5s)
```




