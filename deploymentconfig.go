/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/


package main

import (
	"time"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	/*"k8s.io/api/core/v1"*/
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/fields"
	/*"k8s.io/client-go/kubernetes"*/
	"k8s.io/client-go/tools/cache"

	deploymentconfigv1meta "github.com/openshift/api/apps/v1" 
	deploymentconfigv1clientset "github.com/openshift/client-go/apps/clientset/versioned"
	/*deploymentconfigv1client "github.com/openshift/client-go/deploymentconfig/clientset/versioned/typed/deploymentconfig/v1"*/

	/*v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"*/
    /* "k8s.io/apimachinery/pkg/runtime"*/



)

var (
	descDeploymentConfigLabelsName          = "oapi_deploymentconfig_labels"
	descDeploymentConfigLabelsHelp          = "DeploymentConfig labels converted to Prometheus labels."
	descDeploymentConfigLabelsDefaultLabels = []string{"namespace", "deployment"}

	descDeploymentConfigCreated = prometheus.NewDesc(
		"oapi_deploymentconfig_created",
		"Unix creation timestamp of DeploymentConfig",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigStatusReplicas = prometheus.NewDesc(
		"oapi_deploymentconfig_status_replicas",
		"The number of replicas per DeploymentConfig.",
		[]string{"namespace", "deployment"}, nil,
	)
	descDeploymentConfigStatusReplicasAvailable = prometheus.NewDesc(
		"oapi_deploymentconfig_status_replicas_available",
		"The number of available replicas per DeploymentConfig.",
		[]string{"namespace", "deployment"}, nil,
	)
	descDeploymentConfigStatusReplicasUnavailable = prometheus.NewDesc(
		"oapi_deploymentconfig_status_replicas_unavailable",
		"The number of unavailable replicas per DeploymentConfig.",
		[]string{"namespace", "deployment"}, nil,
	)
	descDeploymentConfigStatusReplicasUpdated = prometheus.NewDesc(
		"oapi_deploymentconfig_status_replicas_updated",
		"The number of updated replicas per DeploymentConfig.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigStatusObservedGeneration = prometheus.NewDesc(
		"oapi_deploymentconfig_status_observed_generation",
		"The generation observed by the deployment controller???.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigSpecReplicas = prometheus.NewDesc(
		"oapi_deploymentconfig_spec_replicas",
		"Number of desired pods for a DeploymentConfig.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigSpecPaused = prometheus.NewDesc(
		"oapi_deploymentconfig_spec_paused",
		"Whether the deployment is paused and will not be processed by the deployment controller.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigStrategyRollingUpdateMaxUnavailable = prometheus.NewDesc(
		"oapi_deploymentconfig_spec_strategy_rollingupdate_max_unavailable",
		"Maximum number of unavailable replicas during a rolling update of a deployment.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigMetadataGeneration = prometheus.NewDesc(
		"oapi_deploymentconfig_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentConfigLabels = prometheus.NewDesc(
		descDeploymentConfigLabelsName,
		descDeploymentConfigLabelsHelp,
		descDeploymentConfigLabelsDefaultLabels, nil,
	)
)


type DeploymentConfigLister func() ([]deploymentconfigv1meta.DeploymentConfig, error)

func (l DeploymentConfigLister) List() ([]deploymentconfigv1meta.DeploymentConfig, error) {
	return l()
}

func RegisterDeploymentConfigCollectorOApi(registry prometheus.Registerer, kubeConfig *rest.Config, namespace string) {
	/* NOTE: appliedclusterresourcequata does not support watch and select by all namespaces*/

 /* Note: OAPI only provides very specifiy clientsets */
   deploymentconfigClient, err := deploymentconfigv1clientset.NewForConfig(kubeConfig)
   if err != nil {
	   glog.Fatalf("Failed to access deploymentconfigs api: %v", err)
   }

   resyncPeriod, _ := time.ParseDuration("0h0m30s")

   client := deploymentconfigClient.AppsV1().RESTClient()

	rqlw := cache.NewListWatchFromClient(client, "deploymentconfigs", namespace, fields.Everything())
	rqinf := cache.NewSharedInformer(rqlw, &deploymentconfigv1meta.DeploymentConfig{}, resyncPeriod)

	deploymentConfigLister := DeploymentConfigLister(func() (deploymentconfigs []deploymentconfigv1meta.DeploymentConfig, err error) {
		for _, dc := range rqinf.GetStore().List() {
			deploymentconfigs = append(deploymentconfigs, *(dc.(*deploymentconfigv1meta.DeploymentConfig)))
		}
		return deploymentconfigs, nil
	})

	registry.MustRegister(&deploymentConfigCollector{store: deploymentConfigLister})
	go rqinf.Run(context.Background().Done())
}

type deploymentConfigStore interface {
	List() ([]deploymentconfigv1meta.DeploymentConfig, error)
}

// deploymentConfigCollector collects metrics about all resource deploymentconfigs in the cluster.
type deploymentConfigCollector struct {
	store deploymentConfigStore
}


// Describe implements the prometheus.Collector interface.
func (dc *deploymentConfigCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDeploymentConfigCreated
	ch <- descDeploymentConfigStatusReplicas
	ch <- descDeploymentConfigStatusReplicasAvailable
	ch <- descDeploymentConfigStatusReplicasUnavailable
	ch <- descDeploymentConfigStatusReplicasUpdated
	ch <- descDeploymentConfigStatusObservedGeneration
	ch <- descDeploymentConfigSpecPaused
	ch <- descDeploymentConfigStrategyRollingUpdateMaxUnavailable
	ch <- descDeploymentConfigSpecReplicas
	ch <- descDeploymentConfigMetadataGeneration
	ch <- descDeploymentConfigLabels
}

// Collect implements the prometheus.Collector interface.
func (dc *deploymentConfigCollector) Collect(ch chan<- prometheus.Metric) {

    /* collect metrics for execution times */
	start := time.Now()

	ds, err := dc.store.List()
	if err != nil {
		glog.Errorf("listing deployments failed: %s", err)
		return
	}

	for _, d := range ds {
		dc.collectDeploymentConfig(ch, d)
	}

	duration := time.Since(start)
	ScrapeDurationHistogram.WithLabelValues("deploymentconfig").Observe(duration.Seconds())

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "deploymentconfig"}).Observe(float64(len(ds)))

	glog.Infof("collected %d deployments", len(ds))
}

func deploymentLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descDeploymentConfigLabelsName,
		descDeploymentConfigLabelsHelp,
		append(descDeploymentConfigLabelsDefaultLabels, labelKeys...),
		nil,
	)
}



func (dc *deploymentConfigCollector) collectDeploymentConfig(ch chan<- prometheus.Metric, d deploymentconfigv1meta.DeploymentConfig) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.Labels)
	addGauge(deploymentLabelsDesc(labelKeys), 1, labelValues...)
	if !d.CreationTimestamp.IsZero() {
		addGauge(descDeploymentConfigCreated, float64(d.CreationTimestamp.Unix()))
	}
	/* TODO: addGauge(descDeploymentConfigStatusReplicas, float64(d.Status.Replicas))*/
	addGauge(descDeploymentConfigStatusReplicasAvailable, float64(d.Status.AvailableReplicas))
	addGauge(descDeploymentConfigStatusReplicasUnavailable, float64(d.Status.UnavailableReplicas))
	addGauge(descDeploymentConfigStatusReplicasUpdated, float64(d.Status.UpdatedReplicas))
	addGauge(descDeploymentConfigStatusObservedGeneration, float64(d.Status.ObservedGeneration))
	addGauge(descDeploymentConfigSpecPaused, boolFloat64(d.Spec.Paused))
	/*addGauge(descDeploymentConfigSpecReplicas, float64(*d.Spec.Replicas))*/
	addGauge(descDeploymentConfigMetadataGeneration, float64(d.ObjectMeta.Generation))

	/* TODO:
	if d.Spec.Strategy.RollingUpdate == nil {
		return
	}

	maxUnavailable, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxUnavailable, int(*d.Spec.Replicas), true)
	if err != nil {
		glog.Errorf("Error converting RollingUpdate MaxUnavailable to int: %s", err)
	} else {
		addGauge(descDeploymentConfigStrategyRollingUpdateMaxUnavailable, float64(maxUnavailable))
	}
	*/
}


