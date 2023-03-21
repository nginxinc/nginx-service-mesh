# NGINX Service Mesh

[NGINX Service Mesh](https://docs.nginx.com/nginx-service-mesh/) is a fully integrated lightweight service mesh that leverages a data plane powered by NGINX Plus to manage container traffic in Kubernetes environments.

NGINX Service Mesh is supported in Rancher 2.5+ when deploying from the Apps and Marketplace. NGINX Service Mesh is not currently supported on k3s.

## Observability
NGINX Service Mesh can integrate with a number of tracing services using OpenTelemetry or OpenTracing.

### Using OpenTelemetry

Telemetry can only be enabled by editing the configuration YAML directly in the Rancher UI. When installing NGINX Service Mesh, select the `Edit YAML` option.
To enable telemetry, fill out the `telemetry` object.
The telemetry object expects a `samplerRatio`, and the `host` and `port` of your OTLP gRPC collector.

For example:

```yaml
telemetry:
  samplerRatio: 0.01
  exporters:
    otlp:
      host: "my-otlp-collector-host"
      port: 4317
```

### Automatic Sidecar Injection

To enable automatic sidecar injection for all Pods in a namespace, label the namespace with `injector.nsm.nginx.com/auto-inject=enabled`.
