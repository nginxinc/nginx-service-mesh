---
title: "Monitoring and Tracing"
date: 2020-08-24T10:00:30-06:00
toc: true
description: "Learn about the monitoring and tracing features available in NGINX Service Mesh."
weight: 20
categories: ["tasks"]
docs: "DOCS-693"
---

## Overview

NGINX Service Mesh can integrate with your Prometheus, Grafana, and tracing backends for exporting metrics and tracing data.

{{< see-also >}}
If you do not have your own Prometheus, Grafana, or tracing backends deployed, the [Observability Tutorial]( {{< ref "/tutorials/observability.md" >}}) will deploy and integrate a basic demo setup for you.
{{< /see-also >}}

### Prometheus

{{< important >}}
In order to prevent automatic sidecar injection into your Prometheus deployment, it should be deployed in a namespace where auto-injection is disabled, or the `injector.nsm.nginx.com/auto-inject: disabled` label must be added to the *PodTemplateSpec* of the Prometheus deployment.

Another option is to disable auto injection globally when deploying the mesh.

Refer to [NGINX Service Mesh Labels and Annotations]( {{< ref "/get-started/configuration.md#supported-labels-and-annotations" >}}) for more information.
{{< /important >}}

To use NGINX Service Mesh with your Prometheus deployment:

{{< warning >}}
We do not currently support Prometheus deployments running with TLS encryption.
{{< /warning>}}

1. Connect your existing Prometheus Deployment to NGINX Service Mesh:

   The expected address format is `<service-name>.<namespace>:<port>`

   - *At deployment*:
   
      Run the `nginx-meshctl deploy` command with the `--prometheus-address` flag.

      For example:

      ```bash
      nginx-meshctl deploy ...  --prometheus-address "my-prometheus.example-namespace:9090"
      ```

   - *At runtime*:

      You can use the [NGINX Service Mesh API]({{< ref "reference/api/api-usage.md" >}})
      to update the Prometheus address that the control plane uses to get metrics.

      For example, send the following payload via `PATCH` to the NGINX Service Mesh API:

      ```json
      {
         "op": "replace",
         "field": {
            "prometheusAddress": "my-prometheus.example-namespace:9090"
         }
      }
      ```

1. Add the `nginx-mesh-sidecars` scrape config to your Prometheus configuration.

   {{< note >}}If you are deploying NGINX Plus Ingress Controller with the NGINX Service Mesh, add the `nginx-plus-ingress` scrape config as well. Consult the [Metrics]( {{< ref "/tutorials/kic/deploy-with-kic.md#nginx-plus-ingress-controller-metrics" >}} ) section of the NGINX Ingress Controller Deployment tutorial for more information about the metrics collected.{{< /note >}}

   - {{< fa "download" >}} {{< link "/examples/nginx-mesh-sidecars-scrape-config.yaml" "`nginx-mesh-sidecars-scrape-config.yaml`" >}}
   - {{< fa "download" >}} {{< link "/examples/nginx-plus-ingress-scrape-config.yaml" "`nginx-plus-ingress-scrape-config.yaml`" >}}

{{< see-also >}}
For more information on how to view and understand the metrics that we track, see our [Prometheus Metrics]({{< ref "prometheus-metrics.md" >}}) guide.
{{< /see-also >}}

### Grafana
The custom NGINX Service Mesh Grafana dashboard `NGINX Mesh Top` can be imported into your Grafana instance. 
For instructions and a list of features, see the [Grafana example](https://github.com/nginxinc/nginx-service-mesh/tree/main/examples/grafana) in the `nginx-service-mesh` GitHub repo.

{{< important >}}
In order to prevent automatic sidecar injection into your Grafana deployment, it should be deployed in a namespace where auto-injection is disabled, or the `injector.nsm.nginx.com/auto-inject: disabled` label must be added to the *PodTemplateSpec* of the Grafana deployment.

Another option is to disable auto injection globally when deploying the mesh.

Refer to [NGINX Service Mesh Labels and Annotations]( {{< ref "/get-started/configuration.md#supported-labels-and-annotations" >}}) for more information.
{{< /important >}}

### Tracing with OpenTelemetry

NGINX Service Mesh can export tracing data using `OpenTelemetry`. Tracing data can be exported to an OpenTelemetry Protocol (OTLP) gRPC Exporter, such as the [OpenTelemetry (OTEL) Collector](https://opentelemetry.io/docs/collector/), which can then export data to one or more upstream collectors like [Jaeger](https://www.jaegertracing.io/), [DataDog](https://docs.datadoghq.com/tracing/), [LightStep](https://lightstep.com/), or many others. Before installing the mesh, deploy an OTEL Collector that is [configured to export data](https://opentelemetry.io/docs/collector/configuration/#exporters) to an upstream collector that you have already deployed or have access to.

Tracing relies on the trace headers passed through each microservice in an application in order to build a full trace of a request. If you don't configure your app to pass trace headers, you'll get disjointed traces that are more difficult to understand. See the [OpenTelemetry Instrumentation](https://opentelemetry.io/docs/instrumentation/) guide for information on how to instrument your application to pass trace headers.

{{< important >}}
In order to prevent automatic sidecar injection into your tracing deployments, it should be deployed in a namespace where auto-injection is disabled, or the `injector.nsm.nginx.com/auto-inject: disabled` label must be added to the *PodTemplateSpec* of the deployments.

Another option is to disable auto injection globally when deploying the mesh.

Refer to [NGINX Service Mesh Labels and Annotations]( {{< ref "/get-started/configuration.md#supported-labels-and-annotations" >}}) for more information.
{{< /important >}}

- *At deployment*:

   Use the `--telemetry-exporters` flag to point the mesh to your OTLP exporter:

   ```bash
   nginx-meshctl deploy ... --telemetry-exporters "type=otlp,host=otel-collector.example-namespace.svc,port=4317"
   ```

   You can set the desired sampler ratio to use for tracing by adding the `--telemetry-sampler-ratio` flag to your deploy command. 
   The sampler ratio must be a float between `0` and `1`. The sampler ratio sets the probability of sampling a span; this means that a sampler ratio of `0.1` sets a 10% probability the span is sampled. For example:

   ```bash
   nginx-meshctl deploy ... --telemetry-sampler-ratio 0.1
   ```

- *At runtime*:

   You can use the [NGINX Service Mesh API]({{< ref "reference/api/api-usage.md" >}}) to update the telemetry configuration.

   For example, send the following payload via `PATCH` to the NGINX Service Mesh API:

   ```json
   {
      "op": "replace",
      "field": {
         "telemetry": {
            "exporters": {
               "otlp": {
                  "host": "otel-collector.example-namespace.svc",
                  "port": 4317
               }
            },
            "samplerRatio": 0.1
         }
      }
   }
   ```

If configured correctly, tracing data that is generated or propagated by the NGINX Service Mesh sidecar will be exported to the OTEL Collector, and then exported to the upstream collector(s), as shown in the following example diagram:

{{< img src="/img/opentelemetry.png" alt="OpenTelemetry Data Flow" >}}
*Tracing data flow using the OpenTelemetry Collector*
