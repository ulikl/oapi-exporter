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
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	/*"k8s.io/api/core/v1"*/
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/fields"
	/*"k8s.io/client-go/kubernetes"*/
	"k8s.io/client-go/tools/cache"

	quotav1meta "github.com/openshift/api/quota/v1" 
	clusterresourcequotav1meta "github.com/openshift/client-go/quota/clientset/versioned"
    /* from NamespaceAll: */
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
		descClusterResourceQuotaCreated = prometheus.NewDesc(
			"oapi_clusterresourcequota_created",
			"Unix creation timestamp of clusterresourcequota",
			[]string{"clusterresourcequota", "namespace"}, nil,
		)
		descClusterResourceQuota = prometheus.NewDesc(
			"oapi_clusterresourcequota",
			"Information about resource requests and limits of appliedclusterresourcequota.",
			[]string{
				"clusterresourcequota",
				"namespace",
				"resource",
				"type",
			}, nil,
		)
	)


type ClusterResourceQuotaLister func() ([]quotav1meta.ClusterResourceQuota, error)

func (l ClusterResourceQuotaLister) List() ([]quotav1meta.ClusterResourceQuota, error) {
	return l()
}

func RegisterClusterResourceQuotaCollectorOApi(registry prometheus.Registerer, kubeConfig *rest.Config, namespace string) {
	/* NOTE: appliedclusterresourcequata does not support watch and select by all namespaces*/

 /* Note: OAPI only provides very specifiy clientsets */
   clusterresourcequotaClient, err := clusterresourcequotav1meta.NewForConfig(kubeConfig)
   if err != nil {
	   glog.Fatalf("Failed to access clusterresourcequotas api: %v", err)
   }

   resyncPeriod, _ := time.ParseDuration("0h0m30s")

   client := clusterresourcequotaClient.QuotaV1().RESTClient()

	// note: namespace not supported here, filter at collection
	rqlw := cache.NewListWatchFromClient(client, "clusterresourcequotas", "", fields.Everything())
	rqinf := cache.NewSharedInformer(rqlw, &quotav1meta.ClusterResourceQuota{}, resyncPeriod)

	clusterResourceQuotaLister := ClusterResourceQuotaLister(func() (clusterresourcequotas []quotav1meta.ClusterResourceQuota, err error) {
		for _, rq := range rqinf.GetStore().List() {
			clusterresourcequotas = append(clusterresourcequotas, *(rq.(*quotav1meta.ClusterResourceQuota)))
		}
		return clusterresourcequotas, nil
	})


	m := make(map[string]int)
	registry.MustRegister(&clusterResourceQuotaCollector{store: clusterResourceQuotaLister, m: m, namespace: namespace})
	go rqinf.Run(context.Background().Done())
}

type clusterResourceQuotaStore interface {
	List() ([]quotav1meta.ClusterResourceQuota, error)
}

// clusterResourceQuotaCollector collects metrics about all resource clusterresourcequotas in the cluster.
type clusterResourceQuotaCollector struct {
	store clusterResourceQuotaStore
	namespace string
	m map[string]int
}


// Describe implements the prometheus.Collector interface.
func (rqc *clusterResourceQuotaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descClusterResourceQuotaCreated
	ch <- descClusterResourceQuota
}

// Collect implements the prometheus.Collector interface.
func (rqc *clusterResourceQuotaCollector) Collect(ch chan<- prometheus.Metric) {

	rqc.m = make(map[string]int)

	rq, err := rqc.store.List()
	if err != nil {
		glog.Errorf("listing deployments failed: %s", err)
		return
	}
	for _, d := range rq {
		rqc.collectClusterResourceQuota(ch, d)
	}

	glog.Infof("collected %d deployments", len(rq))
}



func (rqc *clusterResourceQuotaCollector) collectClusterResourceQuota(ch chan<- prometheus.Metric, rql quotav1meta.ClusterResourceQuota) {

		//glog.Infof("m before %s", rqc.m)
		nsfound := (rqc.namespace == v1meta.NamespaceAll)
		
		for _, rq := range rql.Status.Namespaces { 
		if (rqc.namespace == rq.Namespace || rqc.namespace == v1meta.NamespaceAll)  {
		 nsfound = true
		 _, ok := rqc.m[strings.Join([]string{rql.Name, rq.Namespace},"/")]
		 if !(ok)  {	
			
			rqc.m[strings.Join([]string{rql.Name, rq.Namespace},"/")] = 1
			addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
				lv = append([]string{rql.Name, rq.Namespace}, lv...)
				ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
			}
			 if !rql.CreationTimestamp.IsZero() {
				addGauge(descClusterResourceQuotaCreated, float64(rql.CreationTimestamp.Unix()))
			}
			for res, qty := range rq.Status.Hard {
				addGauge(descClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
			}
			for res, qty := range rq.Status.Used {
				addGauge(descClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
			}
		 }
		}
	    } 
		_, ok := rqc.m[strings.Join([]string{rql.Name, ""},"/")]
		if (!ok && nsfound)  {	
			rqc.m[strings.Join([]string{rql.Name, ""},"/")] = 1
	
			rqTotal := rql.Status.Total
			addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
				lv = append([]string{rql.Name, ""}, lv...)
				ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
			}
			 if !rql.CreationTimestamp.IsZero() {
				addGauge(descClusterResourceQuotaCreated, float64(rql.CreationTimestamp.Unix()))
			}
			for res, qty := range rqTotal.Hard {
				addGauge(descClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
			}
			for res, qty := range rqTotal.Used {
				addGauge(descClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
			}
		}
		
	}
	

