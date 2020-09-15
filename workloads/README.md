# Workloads

A set of helpers, demo tools, and documentation companions.

## Table of Contents
- [Prerequisites](#prereqs)
- [Interval Sender](#sender) - basic workload to send configurable request on an interval.
- [Interval Responder](#responder) - basic workload to serve requests with simple configurability.

## Prerequisites <a name="prereqs"></a>

* For Linux
  * [Docker](https://docs.docker.com/engine/install/#server) - Install the latest stable version for the appropriate distribution.

* For macOS:
  * [Docker](https://docs.docker.com/docker-for-mac/install/) - Install the latest stable version.

## Interval Sender <a name="sender"></a>

The interval sender is a simple application we've created to help illustrate examples in our documentation. During walk-throughs and tutorials readers will be notified when the Interval Sender is to be used and a prerequisite constraint to beginning the tutorial.

Its behavior is very limited and for example purposes only, but it is capable of some limited configuration. It supports environment variables that can be set and passed in via ConfigMaps.

### Configuration
- `HOST`: Destination host to send to, use form `[scheme]://[hostname][:port]`. For example, `http://target:8080`. Default: `http://localhost:8080`,
- `REQUEST_PATH`: URI path segment for request. Default: `/echo`.
- `METHOD`: HTTP method to use. Default: `GET`.
- `HEADERS`: Optional additional headers to apply to request. Use the form `[header]:[value]` separated by commas, for example, `header1:one, header2: two, header3: three`.

### Usage
```
git clone github.com/nginxinc/nginx-service-mesh
cd nginx-service-mesh/workloads/interval-sender
docker build -t [container-registry/]interval-sender:nsm .
docker push [container-registry/]interval-sender:nsm
```

## Interval Responder <a name="responder"></a>

The interval responder is a simple application we've created in conjunction with Interval Sender to help illustrate examples in our documentation. During walk-throughs and tutorials readers will be notified when the Interval Responder is to be used and a prerequisite constraint to beginning the tutorial.

Its behavior is very limited and for example purposes only, but it is capable of some limited configuration. It supports environment variables that can be set and passed in via ConfigMaps.

### Configuration
- `RECEIVE_PATHS`: Basic routing for the workload, the server will echo the request for each path set. The paths `/echo` and `/error` are set by default, the `/echo` path also echos the request while the `/error` path will respond with a 503. Each path must begin with `/` and the list must be comma delimited, for example, `/path1,/path2,/path3`.
- `PORT`: The applications listen port. Default: 8080.

### Usage
```
git clone github.com/nginxinc/nginx-service-mesh
cd nginx-service-mesh/workloads/interval-responder
docker build -t [container-registry/]interval-responder:nsm .
docker push [container-registry/]interval-responder:nsm
```
