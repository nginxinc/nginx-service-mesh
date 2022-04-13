# Prometheus Configuration

NGINX Service Mesh deploys a Prometheus server that is configured to scrape metrics
from NGINX Service Mesh sidecars and NGINX Plus Ingress Controller Pods. If you
prefer to use an existing Prometheus deployment, you can add the following
scrape configs to your Prometheus configuration file. 

## NGINX Mesh Sidecar Scrape Config

To add the scrape config for NGINX Mesh Sidecars, copy the text below or
download the [file](./nginx-mesh-sidecars-scrape-config.yaml).
```
- job_name: 'nginx-mesh-sidecars'
  kubernetes_sd_configs:
    - role: pod
  relabel_configs:
    - source_labels: [__meta_kubernetes_pod_container_name]
      action: keep
      regex: nginx-mesh-sidecar
    - action: labelmap
      regex: __meta_kubernetes_pod_label_nsm_nginx_com_(.+)
    - action: labeldrop
      regex: __meta_kubernetes_pod_label_nsm_nginx_com_(.+)
    - action: labelmap
      regex: __meta_kubernetes_pod_label_(.+)
    - source_labels: [__meta_kubernetes_namespace]
      action: replace
      target_label: namespace
    - source_labels: [__meta_kubernetes_pod_name]
      action: replace
      target_label: pod
```

The `nginx-mesh-sidecars` job scrapes NGINX Service Mesh sidecars on
port `8887` and the path `/metrics`. All metrics collected from this job are published under
the
`nginxplus` namespace. For a list of available metrics, labels, and example
Prometheus queries, check out the [Traffic
Metrics](https://docs.nginx.com/nginx-service-mesh/guides/prometheus-metrics/)
guide.


## NGINX Plus Ingress Controller Scrape Config
If you are deploying NGINX Plus Ingress Controller with NGINX Service Mesh,
copy the following scrape config to your Prometheus configuration file, or
download the [file](./nginx-plus-ingress-scrape-config.yaml). 
```
- job_name: 'nginx-plus-ingress'
  kubernetes_sd_configs:
    - role: pod
  relabel_configs:
    - source_labels: [__meta_kubernetes_pod_container_name]
      action: keep
      regex: nginx-plus-ingress
    - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
      action: replace
      target_label: __address__
      regex: (.+)(?::\d+);(\d+)
      replacement: $1:$2
    - source_labels: [__meta_kubernetes_namespace]
      action: replace
      target_label: namespace
    - source_labels: [__meta_kubernetes_pod_name]
      action: replace
      target_label: pod
    - action: labelmap
      regex: __meta_kubernetes_pod_label_nsm_nginx_com_(.+)
    - action: labeldrop
      regex: __meta_kubernetes_pod_label_nsm_nginx_com_(.+)
    - action: labelmap
      regex: __meta_kubernetes_pod_label_(.+)
    - action: labelmap
      regex: __meta_kubernetes_pod_annotation_nsm_nginx_com_enable_(.+)
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: 'nginx_ingress_controller_upstream_server_response_latency_ms(.+)'
      target_label: __name__
      replacement: 'nginxplus_upstream_server_response_latency_ms$1'
    - source_labels: [__name__]
      regex: 'nginx_ingress_nginxplus(.+)'
      target_label: __name__
      replacement: 'nginxplus$1'
    - source_labels: [service]
      target_label: dst_service
    - source_labels: [resource_namespace]
      target_label: dst_namespace
    - source_labels: [pod_owner]
      regex: '(.+)\/(.+)'
      target_label: dst_$1
      replacement: $2
    - action: labeldrop
      regex: pod_owner
    - source_labels: [pod_name]
      target_label: dst_pod
```
The `nginx-plus-ingress` job scrapes metrics from all Pods with a container
name of `nginx-plus-ingress`. Metrics collected by this job are published under
the `nginxplus` and `nginx_ingress_controlller` namespaces. For more
information about NGINX Plus Ingress Controller metrics, see the [NGINX Plus
Ingress
Controller Metrics](https://docs.nginx.com/nginx-service-mesh/tutorials/kic/deploy-with-kic/#nginx-plus-ingress-controller-metrics)
section of the docs.
