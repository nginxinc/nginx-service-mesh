---
title: "Deploy with NGINX Plus Ingress Controller"
description: "This topic describes how to install and use the NGINX Plus Ingress Controller with NGINX Service Mesh"
weight: 200
draft: false
toc: true
categories: ["tutorials"]
docs: "DOCS-721"
---

## Overview

You can deploy NGINX Ingress Controller for Kubernetes with NGINX Service Mesh to control both ingress and [egress](#enable-egress) traffic.

{{< important >}}
There are two versions of NGINX Ingress Controller for Kubernetes: NGINX Open Source and NGINX Plus.
To deploy NGINX Ingress Controller with NGINX Service Mesh, you must use the NGINX Plus version.
Visit the [NGINX Ingress Controller](https://www.nginx.com/products/nginx-ingress-controller/) product page for more information.
{{< /important >}}

## Supported Versions

The supported NGINX Plus Ingress Controller versions for each release are listed in the [technical specifications]({{< ref "/about/tech-specs.md#supported-versions" >}}) doc.

The documentation for the latest stable release of NGINX Ingress Controller is available at [docs.nginx.com/nginx-ingress-controller](https://docs.nginx.com/nginx-ingress-controller/).
For version specific documentation, deployment configs, and configuration examples, select the tag corresponding to your desired version in [GitHub](https://github.com/nginxinc/kubernetes-ingress/tags). 

## Secure Communication Between NGINX Plus Ingress Controller and NGINX Service Mesh

The NGINX Plus Ingress Controller can participate in the mTLS cert exchange with services in the mesh without being injected with the sidecar proxy. The SPIRE server - the certificate authority of the mesh - issues certs and keys for NGINX Plus Ingress Controller and pushes them to the SPIRE agents running on each node in the cluster. NGINX Plus Ingress Controller fetches these certs and keys from the SPIRE agent via a unix socket and uses them to communicate with services in the mesh.

## Cert Rotation with NGINX Plus Ingress Controller

The `ttl` of the SVID certificates issued by SPIRE is set to `1hr` by default. You can change this when deploying the mesh; refer to the [nginx-meshctl]({{< ref "nginx-meshctl.md" >}}) documentation for more information.

When using NGINX Plus Ingress Controller with mTLS enabled, it is best practice to keep the `ttl` at 1 hour or greater.

## Install NGINX Plus Ingress Controller with mTLS enabled

To configure NGINX Plus Ingress Controller to communicate with mesh workloads over mTLS you need to make a few modifications to the Ingress Controller's Pod spec. This section describes each modification that is required, but if you'd like to jump to installation, go to the [Install with Manifests](#install-with-manifests) or [Install with Helm](#install-with-helm) sections. 

1. Mount the SPIRE agent socket
    
    The SPIRE agent socket needs to be mounted to the Ingress Controller Pod so the Ingress Controller can fetch its certificates and keys from the SPIRE agent. This allows the Ingress Controller to authenticate with workloads in the mesh. For more information on how SPIRE distributes certificates see the [SPIRE]({{< ref "/about/architecture#spire" >}}) section in the architecture doc. 

    - *Kubernetes*
       
        To mount the SPIRE agent socket in Kubernetes, add the following `hostPath` as a volume to the Ingress Controller's Pod spec:

        ```yaml
        volumes:
        - hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
          name: spire-agent-socket
        ```

        and mount the socket to the Ingress Controller's container spec:

        ```yaml
        volumeMounts:
        - mountPath: /run/spire/sockets
          name: spire-agent-socket
        ```

    - *OpenShift*

        To mount the SPIRE agent socket in OpenShift, add the following `csi` driver to the Ingress Controller's Pod spec:
  
        ```yaml
        volumes:
        - csi:
          driver: csi.spiffe.io
          readOnly: true
        name: spire-agent-socket
        ```

        and mount the socket to the Ingress Controller's container spec:

        ```yaml
        volumeMounts:
        - mountPath: /run/spire/sockets
          name: spire-agent-socket
        ```

        For more information as to why a CSI Driver is needed for loading the agent socket in OpenShift, see [Introduction]({{< ref "/get-started/openshift-platform/considerations#introduction" >}}) in the OpenShift Considerations doc.

1. Add command line arguments

    The following arguments must be added to the Ingress Controller's container args:
   
    ```yaml
    args:
      - -nginx-plus
      - -spire-agent-address=/run/spire/sockets/agent.sock
      ...
    - 
    ```

    - The `nginx-plus` argument is required since this feature is only available with NGINX Plus. If you do not specify this flag, the Ingress Controller will fail to start.
    - The `spire-agent-address` passes the address of the SPIRE agent `/run/spire/sockets/agent.sock` to the Ingress Controller.

1. Add NGINX Service Mesh annotation

    The following annotation must be added to the Ingress Controller's Pod spec:

    ```yaml
    annotations:
      nsm.nginx.com/enable-ingress: "true"
      ...
    ```

    This annotation prevents NGINX Service Mesh from automatically injecting the sidecar into the Ingress Controller Pod.


1. Add SPIFFE label

    ```yaml
    labels:
      spiffe.io/spiffeid: "true"
      ...
    ```

    This label tells SPIRE to generate a certificate for the Ingress Controller Pod(s).

{{< note >}}
All communication between NGINX Plus Ingress Controller and the upstream Services occurs over mTLS, using the certificates and keys generated by the SPIRE server.
Therefore, NGINX Plus Ingress Controller can only route traffic to Services in the mesh that have an `mtls-mode` of `permissive` or `strict`.
In cases where you need to route traffic to both mTLS and non-mTLS Services, you may need another Ingress Controller that does not participate in the mTLS fabric.

Refer to the NGINX Ingress Controller's [Running Multiple Ingress Controllers](https://docs.nginx.com/nginx-ingress-controller/installation/running-multiple-ingress-controllers/) guide for instructions on how to configure multiple Ingress Controllers.
{{< /note >}}


If you would like to enable egress traffic, refer to the [Enable Egress](#enable-egress) section of this guide.

### Install with Manifests

Before installing NGINX Plus Ingress Controller, you must install NGINX Service Mesh with an [mTLS mode]({{< ref "/guides/secure-traffic-mtls.md" >}}) of `permissive`, or `strict`.
NGINX Plus Ingress Controller will try to fetch certs from the SPIRE agent on startup. If it cannot reach the SPIRE agent, startup will fail, and NGINX Plus Ingress Controller will go into CrashLoopBackoff state. The state will resolve once NGINX Plus Ingress Controller connects to the SPIRE agent.
For instructions on how to install NGINX Service Mesh, see the [Installation]({{< ref "/get-started/install.md" >}}) guide.

{{< note >}}
Before continuing, check the NGINX Plus Ingress Controller [supported versions](#supported-versions) section and make sure you are working off the correct release tag for all NGINX Plus Ingress Controller instructions.
{{< /note >}}

1. Build or Pull the NGINX Plus Ingress Controller image:
    - [Create and push your NGINX Plus Docker image](https://docs.nginx.com/nginx-ingress-controller/installation/building-ingress-controller-image/).
    - For NGINX Plus Ingress releases >= `1.12.0` you can also [pull the NGINX Plus Docker image](https://docs.nginx.com/nginx-ingress-controller/installation/pulling-ingress-controller-image/).
1. Set up Kubernetes Resources for [NGINX Plus Ingress Controller](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) using Kubernetes manifests:
    - [Configure role-based access control (RBAC)](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/#1-configure-rbac)
    - [Create Common Resources](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/#2-create-common-resources)
1. Create the NGINX Plus Ingress Controller as a **Deployment** or **DaemonSet** in Kubernetes using one of the following example manifests:
    - Kubernetes Deployment: {{< fa "download" >}} {{< link "/examples/nginx-ingress-controller/nginx-plus-ingress.yaml" "`nginx-ingress-controller/nginx-plus-ingress.yaml`" >}}
    - Kubernetes DaemonSet: {{< fa "download" >}} {{< link "/examples/nginx-ingress-controller/nginx-plus-ingress-daemonset.yaml" "`nginx-ingress-controller/nginx-plus-ingress-daemonset.yaml`" >}}
    - OpenShift Deployment: {{< fa "download" >}} {{< link "/examples/nginx-ingress-controller/openshift/nginx-plus-ingress.yaml" "`nginx-ingress-controller/openshift/nginx-plus-ingress.yaml`" >}}
    - Openshift DaemonSet:  {{< fa "download" >}} {{< link "/examples/nginx-ingress-controller/openshift/nginx-plus-ingress-daemonset.yaml" "`nginx-ingress-controller/openshift/nginx-plus-ingress-daemonset.yaml`" >}}
      {{< note >}} The provided manifests configure NGINX Plus Ingress Controller for ingress traffic only. If you would like to enable egress traffic, refer to the [Enable Egress](#enable-with-manifests) section of this guide. {{< /note >}}
      {{< important >}} Be sure to replace the `nginx-plus-ingress:version` image used in the manifest with the chosen image from the F5 Container registry; or the container image that you have built. {{< /important >}}
      
    - *OpenShift only*:

      Download the SecurityContextConstraint necessary to run NGINX Plus Ingress Controller in an OpenShift environment.

      - {{< fa "download" >}} {{< link "/examples/nginx-ingress-controller/openshift/nic-scc.yaml" "`nginx-ingress-controller/openshift/nic-scc.yaml`" >}}

      - Apply the `nginx-ingress-permissions` SecurityContextConstraint:

        ```bash
        kubectl apply -f nic-scc.yaml
        ```

      - Install the OpenShift CLI by following the steps in their [documentation](https://docs.openshift.com/container-platform/4.7/cli_reference/openshift_cli/getting-started-cli.html#installing-openshift-cli).

      - Add the `nginx-ingress-permissions` to the ServiceAccount of the NGINX Plus Ingress Controller.

        ```bash
        oc adm policy add-scc-to-user nginx-ingress-permissions -z nginx-ingress -n nginx-ingress
        ```

### Install with Helm 

Before installing NGINX Plus Ingress Controller, you must install NGINX Service Mesh with an [mTLS mode]({{< ref "/guides/secure-traffic-mtls.md" >}}) of `permissive`, or `strict`.
NGINX Plus Ingress Controller will try to fetch certs from the SPIRE agent on startup. If it cannot reach the SPIRE agent, startup will fail, and NGINX Plus Ingress Controller will go into CrashLoopBackoff state. The state will resolve once NGINX Plus Ingress Controller connects to the SPIRE agent.
For instructions on how to install NGINX Service Mesh, see the [Installation]({{< ref "/get-started/install.md" >}}) guide.
   
{{< note >}} NGINX Plus Ingress Controller v2.2+ is required to deploy via Helm and integrate with NGINX Service Mesh. {{< /note >}}

Follow the [instructions](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/) to install the NGINX Plus version of the Ingress Controller with Helm.
Set the `nginxServiceMesh.enable` parameter to `true`.
{{< note >}} This will configure NGINX Plus Ingress Controller to route ingress traffic to NGINX Service Mesh workloads. If you would like to enable egress traffic, refer to the [Enable Egress](#enable-with-helm) section of this guide. {{< /note >}}

The [`values-nsm.yaml`](https://github.com/nginxinc/kubernetes-ingress/blob/master/deployments/helm-chart/values-nsm.yaml) file contains all the configuration parameters that are relevant for integration with NGINX Service Mesh. You can use this file if you are installing NGINX Plus Ingress Controller via chart sources.

## Expose your applications

With mTLS enabled, you can use Kubernetes [Ingress](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/), [VirtualServer, and VirtualServerRoutes](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/) resources to configure load balancing for HTTP and gRPC applications.
TCP load balancing via TransportServer resources is not supported.

{{< note >}}
The NGINX Plus Ingress Controller's custom resource [TransportServer](https://docs.nginx.com/nginx-ingress-controller/configuration/transportserver-resource/) and the SMI Spec's custom resource [TrafficSplit](https://github.com/servicemeshinterface/smi-spec/blob/main/apis/traffic-split/v1alpha3/traffic-split.md) share the same Kubernetes short name `ts`. 
To avoid conflicts, use the full names `transportserver(s)` and `trafficsplit(s)` when managing these resources with `kubectl`.
{{< /note >}}

To learn how to expose your applications using NGINX Plus Ingress Controller, refer to the [Expose an Application with NGINX Plus Ingress Controller]( {{< ref "/tutorials/kic/ingress-walkthrough.md" >}} ) tutorial.

## Enable Egress

You can configure NGINX Plus Ingress Controller to act as the egress endpoint of the mesh, enabling your meshed services to communicate securely with external, non-meshed services.

{{<note>}} Multiple endpoints for a single egress deployment are supported, but multiple egress
deployments are not supported. {{</note>}}

### Enable with Manifests
If you are installing NGINX Plus Ingress Controller with manifests follow the [Install with Manifests](#install-with-manifests) instructions and make the following changes to the NGINX Plus Ingress Controller Pod spec:

- Add the following annotation to the NGINX Plus Ingress Controller Pod spec:

    ```bash
    nsm.nginx.com/enable-egress: "true"
    ```

  This annotation prevents automatic injection of the sidecar proxy and configures the NGINX Plus Ingress Controller as the egress endpoint of the mesh.

- Add the following command-line argument to the container args in the NGINX Plus Ingress Controller Pod spec:

    ```bash
    -enable-internal-routes
    ```

  This will create a virtual server block in NGINX Plus Ingress Controller that terminates TLS connections using the SPIFFE certs fetched from the SPIRE agent.

  {{< important >}}This command-line argument must be used with the `-nginx-plus` and `spire-agent-address` command-line arguments. {{< /important >}}

### Enable with Helm 

{{<note>}} NGINX Plus Ingress Controller v2.2+ is required to deploy via Helm and integrate with NGINX Service Mesh. {{</note>}}

If you are installing NGINX Plus Ingress Controller with Helm, follow the [Install with Helm](#install-with-helm) instructions and set `nginxServiceMesh.enableEgress` to `true`.

### Allow Pods to route egress traffic through NGINX Plus Ingress Controller

If egress is enabled you can configure Pods to route **all** egress traffic - requests to non-meshed services - through NGINX Plus Ingress Controller.
This feature can be enabled by adding the following annotation to the Pod spec of an application Pod:

```bash
config.nsm.nginx.com/default-egress-allowed: "true"
```

This annotation can be removed or changed after deployment and the egress behavior of the Pod will be updated accordingly.

### Create internal routes for non-meshed services

Internal routes represent a route from NGINX Plus Ingress Controller to a non-meshed service.
This route is called "internal" because it is only accessible from a Pod in the mesh and is not accessible from the public internet.

{{< caution >}}
If you deploy NGINX Plus Ingress Controller without mTLS enabled, the internal routes could be accessible from the public internet.
We do not recommend using the egress feature with a plaintext deployment of NGINX Plus Ingress Controller.
{{< /caution >}}

To create an internal route, create an [Ingress resource](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/) using the information of your non-meshed service and add the following annotation:

 ```bash
nsm.nginx.com/internal-route: "true"
```

If your non-meshed service is external to Kubernetes, follow the [ExternalName services example](https://github.com/nginxinc/kubernetes-ingress/tree/main/examples/custom-resources/externalname-services).

{{<note>}}
The `nsm.nginx.com/internal-route: "true"` Ingress annotation is still required for routing to external endpoints.
{{</note>}}

The [NGINX Ingress Controller egress tutorial]({{< ref "/tutorials/kic/egress-walkthrough.md" >}}) provides instructions for creating internal routes for non-meshed services.


## Enable Ingress and Egress Traffic

There are a couple ways to enable both ingress and egress traffic using the NGINX Plus Ingress Controller.
You can either allow both ingress and egress traffic through the same NGINX Plus Ingress Controller,
or deploy two NGINX Plus Ingress Controllers: one for handling ingress traffic only and the other for handling egress traffic.

For the single deployment option, follow the [installation instructions](#install-nginx-plus-ingress-controller-with-mtls-enabled) and the instructions to [Enable Egress](#enable-egress).
If you would like to configure two Ingress Controllers to keep ingress and egress traffic separate you can leverage [Ingress Classes](https://docs.nginx.com/nginx-ingress-controller/installation/running-multiple-ingress-controllers/#ingress-class).

## Enable UDP Traffic

By default, NGINX Plus Ingress Controller only routes TCP traffic. You can configure it to route UDP traffic by making the following changes to the NGINX Plus Ingress Controller before deploying:

- Enable GlobalConfiguration resources for NGINX Plus Ingress Controller by following the setup defined in the [GlobalConfiguration Resource](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource) documentation.
  
  This allows you to define global configuration parameters for the NGINX Ingress Controller, and create a UDP listener to route ingress UDP traffic to your backend applications.

{{< important >}}
mTLS does not affect UDP communication, as mTLS in NGINX Service Mesh applies only to TCP traffic at this time.
{{< /important >}}

### Create a GlobalConfiguration Resource

To allow UDP traffic to be routed to your Kubernetes applications, create a UDP listener in NGINX Plus Ingress Controller. This can be done via a GlobalConfiguration Resource.

To create a GlobalConfiguration resource, see the NGINX Plus Ingress Controller [documentation](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource#globalconfiguration-specification) to create a listener with protocol UDP.

### Ingress UDP Traffic

You can pass and load balance UDP traffic by using a TransportServer resource. This will link the UDP listener defined in the [Create a GlobalConfiguration Resource](#create-a-globalconfiguration-resource) step with an upstream associated with your designated backend UDP application.

To crate a TransportServer resource, follow the steps outlined in the [TransportServer](https://docs.nginx.com/nginx-ingress-controller/configuration/transportserver-resource/) NGINX Plus Ingress Controller guide and link the UDP listener with the name and port of your backend service.

To learn how to expose a UDP application using NGINX Plus Ingress Controller, see the [Expose a UDP Application with NGINX Plus Ingress Controller]({{< ref "/tutorials/kic/ingress-udp-walkthrough.md" >}}) tutorial.

## Plaintext configuration

Deploy NGINX Service Mesh with `mtls-mode` set to `off` and follow the [instructions](https://docs.nginx.com/nginx-ingress-controller/installation) to deploy NGINX Plus Ingress Controller.

Add the enable-ingress and/or the enable-egress annotation shown below to the NGINX Plus Ingress Controller Pod spec:

```bash
nsm.nginx.com/enable-ingress: "true"
nsm.nginx.com/enable-egress: "true"
```

{{< caution >}}
All communication between NGINX Plus Ingress Controller and the services in the mesh will be over plaintext!
We do not recommend using the egress feature with a plaintext deployment of NGINX Plus Ingress Controller,
it is possible that internal routes could be accessible from the public internet.
We highly recommend [installing  NGINX Plus Ingress Controller with mTLS enabled](#install-nginx-plus-ingress-controller-with-mtls-enabled).
{{< /caution >}}

## OpenTracing Integration

To enable traces to span from NGINX Plus Ingress Controller through the backend services in the Mesh, you'll first need to [build the NGINX Plus Ingress Controller image](https://docs.nginx.com/nginx-ingress-controller/installation/building-ingress-controller-image/#) with the OpenTracing module.
Refer to the [NGINX Ingress Controller guide to using OpenTracing](https://docs.nginx.com/nginx-ingress-controller/third-party-modules/opentracing/) for more information.

NGINX Service Mesh natively supports Zipkin, Jaeger, and DataDog; refer to the [Monitoring and Tracing]( {{< ref "/guides/monitoring-and-tracing.md" >}} ) topic for more information.

If your tracing backend is being used by the Mesh, use the CLI tool to find the address of the tracing server and the sample rate.

```bash
nginx-meshctl config
{
...
  "tracing": {
    "backend": "jaeger",
    "backendAddress": "jaeger.my-namespace.svc:6831",
    "sampleRate": .01
  },
...
}
```

You will need to provide these values in the `opentracing-tracer-config` field of the NGINX Plus Ingress Controller ConfigMap.

Below is an example of the config for Jaeger:

```yaml
  opentracing-tracer-config: |
     {
       "service_name": "nginx-ingress",
       "sampler": {
          "type": "probabilistic",
          "param": .01
       },
       "reporter": {
          "localAgentHostPort": "jaeger.my-namespace.svc:6831"
       }
     }
```

Add the annotation shown below to your Ingress resources. Doing so ensures that the span context propagates to the upstream requests and the operation name displays as "nginx-ingress".

{{< note >}}
The example below uses the snippets annotation. Starting with NGINX Plus Ingress Controller version 2.1.0, snippets are disabled by default. To use snippets, set the `enable-snippets` [command-line argument](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments) on the NGINX Plus Ingress Controller Deployment or Daemonset.
{{< /note >}}

```yaml
    nginx.org/location-snippets: |
     opentracing_propagate_context;
     opentracing_operation_name "nginx-ingress";
```

## NGINX Plus Ingress Controller Metrics

To enable metrics collection for the NGINX Plus Ingress Controller, take the following steps:

1. Run the NGINX Plus Ingress Controller with both the `-enable-prometheus-metrics` and `-enable-latency-metrics` command line arguments.
    The NGINX Plus Ingress Controller exposes [NGINX Plus metrics](https://github.com/nginxinc/nginx-prometheus-exporter#exported-metrics) and latency metrics
    in Prometheus format via the `/metrics` path on port 9113. This port is customizable via the `-prometheus-metrics-listen-port` command-line argument; consult the
    [Command Line Arguments](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/) section of the NGINX Plus Ingress Controller docs for more information on available command line arguments.

1. Add the following Prometheus annotations NGINX Plus Ingress Controller Pod spec:

   ```yaml
   prometheus.io/scrape: "true"
   prometheus.io/port: "<prometheus-metrics-listen-port>"
   ```

1. Add the resource name as a label to the NGINX Plus Ingress Controller Pod spec:

    - For *Deployment*:

      ```yaml
      nsm.nginx.com/deployment: <name of NGINX Plus Ingress Controller Deployment>
      ```

    - For *DaemonSet*:

      ```yaml
      nsm.nginx.com/daemonset: <name of NGINX Plus Ingress Controller DaemonSet>
      ```

    This allows metrics scraped from NGINX Plus Ingress Controller Pods to be associated with the resource that created the Pods.

### View the metrics in Prometheus

The NGINX Service Mesh uses the Pod's container name setting to identify the NGINX Plus Ingress Controller metrics that should be consumed by the Prometheus server.
The Prometheus job targets all Pods that have the container name `nginx-plus-ingress`.

Add the `nginx-plus-ingress` scrape config to your Prometheus configuration and consult
[Monitoring and Tracing]( {{< ref "/guides/monitoring-and-tracing.md#prometheus" >}} ) for installation instructions.

- {{< fa "download" >}} {{< link "/examples/nginx-plus-ingress-scrape-config.yaml" "`nginx-plus-ingress-scrape-config.yaml`" >}}

## Available metrics
For a list of the NGINX Plus Ingress Controller metrics, consult the [Available Metrics](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/prometheus/#available-metrics) section of the NGINX Plus Ingress Controller docs.

{{< note >}}
The NGINX Plus metrics exported by the NGINX Plus Ingress Controller are renamed from `nginx_ingress_controller_<metric-name>` to `nginxplus_<metric-name>` to be consistent with the metrics exported by NGINX Service Mesh sidecars.
For example, `nginx_ingress_controller_upstream_server_response_latency_ms_count` is renamed to `nginxplus_upstream_server_response_latency_ms_count`.
The Ingress Controller specific metrics, such as `nginx_ingress_controller_nginx_reloads_total`, are not renamed.
{{< /note >}}

For more information on metrics, a list of Prometheus labels, and examples of querying and filtering, see the [Prometheus Metrics]({{< ref "/guides/prometheus-metrics.md" >}}) doc.

To view the metrics, use port-forwarding:

```bash
kubectl port-forward -n nginx-mesh svc/prometheus 9090
```

### Monitor your application in Grafana

NGINX Service Mesh provides a [custom dashboard](https://github.com/nginxinc/nginx-service-mesh/tree/main/examples/grafana) that you can import into your Grafana deployment to monitor your application. To import and view the dashboard, port-forward your Grafana service:

```bash
kubectl port-forward -n <grafana-namespace> svc/grafana 3000
```

Then you can navigate your browser to `localhost:3000` to view Grafana.

Here is a view of the provided "NGINX Mesh Top" dashboard:

{{< img src="/img/grafana.png" >}}
