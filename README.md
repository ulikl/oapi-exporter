# Overview

# Deprecation notice

This project is no longer maintained.
Its functionality and more is provided by
https://github.com/openshift/openshift-state-metrics

For migration note, that the metrics name and labels are slightly different.

# Description

oapi-exporter is a simple service that listens to the OpenShift API
server and generates metrics about the state of the OpenShift objects. (See examples in
the Metrics section below.) 

It provides metrics for the state/health and configuration data of the various OpenShift specific objects.
such as deploymentConfigs and resource quota.
It was inspired by [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) exporter, which provides analog metrics using the Kubernetes API server.

Hence to get a full overview over the objects in the OpenShift Cluster, you need both exporter.

oapi-exporter is about generating metrics from Kubernetes API objects
without modification. This ensures that features provided by oapi-exporter
have the same grade of stability as the OpenShift API objects themselves. oapi-exporter exposes raw data unmodified from the
OpenShift API, this way users have all the data they require and perform heuristics as they see fit.

The metrics are exported on the HTTP endpoint `/metrics` on the listening port
(default 80). They are served as plaintext. They are designed to be consumed
either by Prometheus itself or by a scraper that is compatible with scraping a
Prometheus client endpoint. You can also open `/metrics` in a browser to see
the raw metrics.

## Table of Contents

- [Versioning](#versioning)
  - [Kubernetes Version](#kubernetes-version)
  - [Compatibility matrix](#compatibility-matrix)
  - [Resource group version compatibility](#resource-group-version-compatibility)
  - [Container Image](#container-image)
- [Metrics Documentation](#metrics-documentation)
- [Self metrics](#oapi-exporter-self-metrics)
- [Resource recommendation](#resource-recommendation)
- [A note on costing](#a-note-on-costing)
- [oapi-exporter vs. metrics-server](#oapi-exporter-vs-metrics-server)
- [Setup](#setup)
  - [Building the Docker container](#building-the-docker-container)
- [Usage](#usage)
  - [Kubernetes Deployment](#kubernetes-deployment)
  - [Limited privileges environment](#limited-privileges-environment)
  - [Development](#development)

### Versioning

#### OpenShift Version

oapi-exporter uses  [`openshift/client-go`](https: github.com/openshift/client-go) to get information about the OpenShift Objects 
and [`kubernetes/client-go`](https://github.com/kubernetes/client-go) to talk with
Kubernetes clusters. 

The supported Kubernetes cluster version is determined by the `client-go` module versions.
All additional compatibility is only best effort, or happens to still/already be supported.

#### Compatibility 

This is very new project.
Currently only metrics for AppliedClusterResourceQuotas and DeploymentConfigs are implemented.

This master branch is tested against Openshift 3.6, 3.9, 3.10 and 3.11.

#### Resource group version compatibility
Resources in Kubernetes can evolve, i.e., the group version for a resource may change from alpha to beta and finally GA
in different Kubernetes versions. For now, oapi-exporter will only use the oldest API available in the latest
release.

#### Container Image

The latest container image can be found via:
* `docker.io/ulikl/oapi-exporter:latest`


### Metrics Documentation

There are many more metrics we could report, but this first pass is focused on what was needed the most.
Please contribute PR's for additional metrics!


### oapi-exporter self metrics

oapi-exporter exposes its own general process metrics under `--telemetry-host` and `--telemetry-port` (default 81)

### Resource recommendation

Resource usage for oapi-exporter changes with the number of the OpenShift objects (DeploymentConfig/ClusterResourceQuotas/Secrets etc.) in the cluster.
To some extent, the OpenShift objects in a cluster are in direct proportion to the node number of the cluster.

As a general rule, you should allocate

* 200MiB memory
* 0.1 cores

Note that if CPU limits are set too low, oapi-exporter' internal queues will not be able to be worked off quickly enough, resulting in increased memory consumption as the queue length grows. If you experience problems resulting from high memory allocation, try increasing the CPU limits.

### A note on costing

By default, oapi-exporter exposes several metrics for events across your cluster. If you have a large number of frequently-updating resources on your cluster, you may find that a lot of data is ingested into these metrics. This can incur high costs on some cloud providers. Please take a moment to [configure what metrics you'd like to expose](docs/cli-arguments.md), as well as consult the documentation for your OpenShift environment in order to avoid unexpectedly high costs.  

Also take the following section into accoutn.

### ClusterResourceQuotas vs AppliedClusterResourceQuotas

Both provide nearly the same metrics on the ClusterResourceQuotas and the usage of the namespaces:
- AppliedClusterResourceQuotas can be accessed with roles per namespace (e.g. namespace admin), but they provide no watch interface. Hence using them for all namespaces requires looping over all namespaces, which leads to many OAPI calls and can easily take more than 10 seconds in total. So the collection is only recommened for use with few namespaces. Here only the ClusterResourceQuotas which apply to the selected namespaces are shown.
- ClusterResourceQuotas require the Role Cluster-Reader or a special Cluster-wide RBAC.
If its selector currently doesn't apply to any namespace, metrics with the hard quotas are provived. If it is used per namespace all ClusterResourceQuotas are still gathered via watch, but only the  which apply to the selected namespaces are shown.
So this collector uses more memory than expected.


### Compilation

Install this project by checking out the git repo and using an new `$GOPATH` via:

```bash
export GO111MODULE=on
export GOPATH=<empty dir>

go build
```


##### patch needed for ../goPath/pkg/mod/k8s.io/client-go\@v11.0.0+incompatible/rest/request.go

        return watch.NewStreamWatcher(restclientwatch.NewDecoder(wrapperDecoder, embeddedDecoder)), nil

fix:

        return watch.NewStreamWatcher(restclientwatch.NewDecoder(wrapperDecoder, embeddedDecoder),nil), nil

#### Building the Docker container

Simply run the following command in this root folder, which will create a
self-contained, statically-linked binary and build a Docker image:
```
build_docker.sh
```

### Usage

Simply build and run oapi-exporter inside a pod which has a
service account token that has read-only access to the OpenShift cluster.

#### OpenShift Deployment

Sample OpenShift template for Deployment:
```
tpl_oapi_exporter.yaml
``` 

#### Limited privileges environment


The services account you are using needs access to the objects for the active collectors.

You can specify a namespace (using the `--namespace` option) and a set of Kubernetes objects (using the `--collectors`) that your service account  in the `oapi-exporter` deployment configuration has access to these.

```yaml
spec:
  template:
    spec:
      containers:
        - args:
          - '--namespace=project1'
```

For the full list of arguments available, see the documentation in [docs/cli-arguments.md](./docs/cli-arguments.md)

#### Development

When developing, test a metric dump against your local Kubernetes cluster by
running:


	oapi-exporter --port=8080 --telemetry-port=8081 --kubeconfig=<KUBE-CONFIG> 

> Users can override the apiserver address in KUBE-CONFIG file with `--apiserver` command line.

Then curl the metrics endpoint

	curl localhost:8080/metrics

To run the e2e tests locally see the documentation in [tests/README.md](./tests/README.md).
