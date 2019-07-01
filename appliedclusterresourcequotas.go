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

/*
oc new-project my-project
oc annotate namespace my-project clusterquota=test

oc create clusterquota crq-test \
     --project-annotation-selector clusterquota=test \
     --hard pods=10 \
	 --hard secrets=20\
	 --hard memory=1Gi\
	 --hard "limits.memory"=2Gi\
	 --hard cpu=1G

oc create clusterquota crq-storage-label \
     --project-label-selector quotalabel=test \
     --hard "requests.storage"=10G 


oc create quota rq-test \
     --hard pods=10 \
	 --hard secrets=20\
	 --hard memory=1Gi\
	 --hard "limits.memory"=2Gi\
	 --hard cpu=1G


apiVersion: v1
items:
- apiVersion: quota.openshift.io/v1
  kind: AppliedClusterResourceQuota
  metadata:
    creationTimestamp: 2019-06-08T08:52:37Z
    name: crq-test
    namespace: my-project
    resourceVersion: "18064"
    selfLink: /apis/quota.openshift.io/v1/namespaces/my-project/appliedclusterresourcequotas/crq-test
    uid: c41318ca-89ca-11e9-824d-080027df2b71
  spec:
    quota:
      hard:
        cpu: 1G
        limits.memory: 2Gi
        memory: 1Gi
        pods: "10"
        secrets: "20"
    selector:
      annotations:
        clusterquota: test
      labels: null
  status:
    namespaces:
    - namespace: my-project
      status:
        hard:
          cpu: 1G
          limits.memory: 2Gi
          memory: 1Gi
          pods: "10"
          secrets: "20"
        used:
          cpu: "0"
          limits.memory: "0"
          memory: "0"
          pods: "0"
          secrets: "9"
    total:
      hard:
        cpu: 1G
        limits.memory: 2Gi
        memory: 1Gi
        pods: "10"
        secrets: "20"
      used:
        cpu: "0"
        limits.memory: "0"
        memory: "0"
        pods: "0"
        secrets: "9"
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
[root@demo test]# oc create clusterquota crq-test      --project-annotation-selector clusterquota=test      --hard pods=10  --hard secrets=20 --hard memory=1Gi --hard "limits.memory"=2Gi --hard cpu=1G^C
[root@demo test]# oc create quota rq-test \
>      --hard pods=10 \
>  --hard secrets=20\
>  --hard memory=1Gi\
>  --hard "limits.memory"=2Gi\
>  --hard cpu=1G
resourcequota/rq-test created

[root@demo test]# oc --loglevel=7 get resourcequota -o yaml

apiVersion: v1
items:
- apiVersion: v1
  kind: ResourceQuota
  metadata:
    creationTimestamp: 2019-06-08T09:10:22Z
    name: rq-test
    namespace: my-project
    resourceVersion: "19565"
    selfLink: /api/v1/namespaces/my-project/resourcequotas/rq-test
    uid: 3edd6af7-89cd-11e9-824d-080027df2b71
  spec:
    hard:
      cpu: 1G
  status:
    hard:
      cpu: 1G
    used:
      cpu: "0"
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""



*/

package main

import (
	"time"
	"strings"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"
	kubeclientset "k8s.io/client-go/kubernetes"
	quotav1meta "github.com/openshift/api/quota/v1" 
	quotav1clientset "github.com/openshift/client-go/quota/clientset/versioned"

	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	 
	
	/* for Dump of structs */
	"github.com/davecgh/go-spew/spew"
)

var (
	descAppliedClusterResourceQuotaCreated = prometheus.NewDesc(
		"oapi_appliedclusterresourcequota_created",
		"Unix creation timestamp of clusterresourcequota",
		[]string{"clusterresourcequota", "namespace"}, nil,
	)
	descAppliedClusterResourceQuotaSelector = prometheus.NewDesc(
		"oapi_appliedclusterresourcequota_selector",
		"Selector of clusterresourcequota to determine the effected namespaces",
		[]string{"clusterresourcequota","type","key","value"}, nil,
	)
	descAppliedClusterResourceQuota = prometheus.NewDesc(
		"oapi_appliedclusterresourcequota",
		"Information about resource requests and limits of appliedclusterresourcequota.",
		[]string{
			"clusterresourcequota",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
)



/*
type AppliedClusterResourceQuotaLister func() (quotav1meta.AppliedClusterResourceQuotaList, error)

func (l AppliedClusterResourceQuotaLister) List() (quotav1meta.AppliedClusterResourceQuotaList, error) {
	return l()
}

func RegisterAppliedClusterResourceQuotaCollector(registry prometheus.Registerer, kubeClient quotav1clientset.Interface, namespace string) {
	glog.Infof("collect appliedclusterresourcequota on demand")

	_, err := kubeClient.QuotaV1().AppliedClusterResourceQuotas("dummyns").List(v1meta.ListOptions{})
	if err != nil {
		glog.Fatalf("Failed to access quotas api: %v", err)
	}

	resourceQuotaLister := AppliedClusterResourceQuotaLister(func() (quotas quotav1meta.AppliedClusterResourceQuotaList, err error) {


		return (quotav1meta.AppliedClusterResourceQuotaList{}), nil
	})

	registry.MustRegister(&resourceQuotaCollector{store: resourceQuotaLister})
}
*/
func RegisterAppliedClusterResourceQuotaCollectorOApi(registry prometheus.Registerer, kubeConfig *rest.Config, namespace string) {
	 /* NOTE: appliedclusterresourcequata does not support watch and select by all namespaces*/

  /* for retrieving the current namespace list */
	kubeClient, err := kubeclientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Failed to access kube api: %v", err)
	}
  /* Note: OAPI only provides very specifiy clientsets */
	quotaClient, err := quotav1clientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Failed to access quotas oapi: %v", err)
	}

	glog.Infof("collect appliedclusterresourcequotas on demand")
	
	if (namespace == v1meta.NamespaceAll) {
		glog.Infof("using appliedclusterresourcequotas for all namespace may be an performance issue. It is recommended to use clusterresourcequotas instead.")
	}
	
  m := make(map[string]int)
	registry.MustRegister(&resourceQuotaCollector{quotaclientset: quotaClient, kubeclientset: kubeClient, namespace: namespace, m: m})
}


type resourceQuotaCollector struct {
	namespace string
	quotaclientset quotav1clientset.Interface
	kubeclientset kubeclientset.Interface
	m map[string]int
}


// Describe implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descAppliedClusterResourceQuotaCreated
	ch <- descAppliedClusterResourceQuotaSelector
	ch <- descAppliedClusterResourceQuota
}

// Collect implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Collect(ch chan<- prometheus.Metric) {
	 /* NOTE: appliedclusterresourcequata does not support watch! */
	quotaClient := rqc.quotaclientset
	kubeClient := rqc.kubeclientset
	
  //m := make(map[string]int)
	rqc.m = make(map[string]int)


	if rqc.namespace == v1meta.NamespaceAll {

	 	/* collect metrics for execution times */
	 	start := time.Now()

	 	namespaceList, err := kubeClient.CoreV1().Namespaces().List(v1meta.ListOptions{})
	 	if err != nil {
			glog.Fatalf("Failed to list namespaces: %v", err)
	 	}

	 	duration := time.Since(start)
	 	ScrapeDurationHistogram.WithLabelValues("appliedclusterresourcequotas ns list").Observe(duration.Seconds())
	 	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "appliedclusterresourcequotas ns list"}).Observe(float64(len(namespaceList.Items)))

	 	start = time.Now()
	 	var ns_count int 
	 	ns_count = 0
	 	for _, ns := range namespaceList.Items {
			resourceQuota, err := quotaClient.QuotaV1().AppliedClusterResourceQuotas(ns.Name).List(v1meta.ListOptions{})
			if err != nil {
				glog.Fatalf("Failed to read quotas: %v", err)
			}
		
			for _, rq := range resourceQuota.Items {
				rqc.collectAppliedClusterResourceQuota(ch, rq)
			}
			ns_count = ns_count + len(resourceQuota.Items)
	 	}

	 	duration = time.Since(start)
	 	ScrapeDurationHistogram.WithLabelValues("appliedclusterresourcequotas").Observe(duration.Seconds())
	 	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "appliedclusterresourcequotas"}).Observe(float64(ns_count))
	 
	 
	 	glog.Infof("collected %d appliedclusterresourcequotas for %s", ns_count , "all namespaces")
	} else {

	  /* collect metrics for execution times */
	  start := time.Now()

	  resourceQuota, err := quotaClient.QuotaV1().AppliedClusterResourceQuotas(rqc.namespace).List(v1meta.ListOptions{})
		if err != nil {
			glog.Fatalf("Failed to read quotas: %v", err)
		}

		for _, rq := range resourceQuota.Items {
			rqc.collectAppliedClusterResourceQuota(ch, rq)
		}

		duration := time.Since(start)
		ScrapeDurationHistogram.WithLabelValues("appliedclusterresourcequotas").Observe(duration.Seconds())
		ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "appliedclusterresourcequotas"}).Observe(float64(len(resourceQuota.Items)))
		glog.Infof("collected %d appliedclusterresourcequotas for %s", len(resourceQuota.Items), rqc.namespace)

  	}
}

func (rqc *resourceQuotaCollector) collectAppliedClusterResourceQuota(ch chan<- prometheus.Metric, rql quotav1meta.AppliedClusterResourceQuota) {

	//glog.Infof("m before %s", rpc.m)

	for _, rq := range rql.Status.Namespaces { 

	if (rqc.namespace == rq.Namespace || rqc.namespace == v1meta.NamespaceAll) {
	 // only include metrics from selected namespaces:


	 _, ok := rqc.m[strings.Join([]string{rql.Name, rq.Namespace},"/")]
	 if !(ok)  {	
		
		rqc.m[strings.Join([]string{rql.Name, rq.Namespace},"/")] = 1
		addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
			lv = append([]string{rql.Name, rq.Namespace}, lv...)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
		}
		if !rql.CreationTimestamp.IsZero() {
			addGauge(descAppliedClusterResourceQuotaCreated, float64(rql.CreationTimestamp.Unix()))
		}
		for res, qty := range rq.Status.Hard {
			addGauge(descAppliedClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
		}
		for res, qty := range rq.Status.Used {
			addGauge(descAppliedClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
		}
	 }
	}
  }
	_, ok := rqc.m[strings.Join([]string{rql.Name, ""},"/")]
	if !(ok)  {	
		rqc.m[strings.Join([]string{rql.Name, ""},"/")] = 1

		rqTotal := rql.Status.Total
		addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
			lv = append([]string{rql.Name, ""}, lv...)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
		}
		 if !rql.CreationTimestamp.IsZero() {
			addGauge(descAppliedClusterResourceQuotaCreated, float64(rql.CreationTimestamp.Unix()))
		}
		for res, qty := range rqTotal.Hard {
			addGauge(descAppliedClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
		}
		for res, qty := range rqTotal.Used {
			addGauge(descAppliedClusterResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
		}

		sel := rql.Spec.Selector
		if (false) {spew.Dump(sel)}
		for key, value := range sel.AnnotationSelector {
			 lv := append([]string{rql.Name, "annotation", key, value})
			ch <- prometheus.MustNewConstMetric(descAppliedClusterResourceQuotaSelector, prometheus.GaugeValue, 1, lv...)
		}
 
		if (sel.LabelSelector != nil) {
			labelMap := (make(map[string]string))
			v1meta.Convert_v1_LabelSelector_To_Map_string_To_string(sel.LabelSelector,&labelMap,nil)
			for key, value := range labelMap {
			 lv := append([]string{rql.Name, "label", key, value})
			 ch <- prometheus.MustNewConstMetric(descAppliedClusterResourceQuotaSelector, prometheus.GaugeValue, 1, lv...)
			}
		}
 
	}


}
