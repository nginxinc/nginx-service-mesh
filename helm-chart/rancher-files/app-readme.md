# NGINX Service Mesh

[NGINX Service Mesh](https://docs.nginx.com/nginx-service-mesh/) is a fully integrated lightweight service mesh that leverages a data plane powered by NGINX Plus to manage container traffic in Kubernetes environments.

NGINX Service Mesh is supported in Rancher 2.5+ when deploying from the Apps and Marketplace. NGINX Service Mesh is not currently supported on k3s.

## Observability
NGINX Service Mesh can integrate with a number of tracing services using OpenTelemetry or OpenTracing.

### Using OpenTelemetry

Telemetry can only be enabled by editing the configuration YAML directly in the Rancher UI. When installing NGINX Service Mesh, select the `Edit YAML` option. To enable telemetry, set the `tracing` object to `{}` and fill out the `telemetry` object.
The telemetry object expects a `samplerRatio`, and the `host` and `port` of your OTLP gRPC collector.
For example:

```yaml
tracing: {}
telemetry:
  samplerRatio: 0.01
  exporters:
    otlp:
      host: "my-otlp-collector-host"
      port: 4317
```

### Using OpenTracing

Note: OpenTracing is deprecated in favor of OpenTelemetry.

Tracing can only be enabled if telemetry is not enabled. In order to enable tracing, edit the configuration YAML directly in the Rancher UI. When installing NGINX Service Mesh, select the `Edit YAML` option, set the `telemetry` object to `{}`, and fill out the `tracing` object.
The tracing object expects a `sampleRate`, an `address` and a `backend`. The three options for backend are "jaeger", "zipkin", and "datadog".

For example:

```yaml
telemetry: {}
tracing:
  sampleRate: 1
  backend: "jaeger"
  address: "jaeger.my-namespace:6831"
```
### Automatic Sidecar Injection

We recommend deploying the mesh with auto-injection disabled globally. You can then opt-in the namespaces where you would like auto-injection enabled.  This ensures that Pods are not automatically injected without your consent, especially in system namespaces.

To opt-in a namespace you can label it with `injector.nsm.nginx.com/auto-inject=enabled` or provide a list of `enabledNamespaces` in YAML. For example:
```yaml
enabledNamespaces:
- namespace1
- namespace2
```
