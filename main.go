/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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
		//"compress/gzip"
		//"context"
		//"io"
		"net"
		"strconv"
		"fmt"
		"log"
		"net/http"
		"net/http/pprof"
		"os"
		"strings"
	
		"github.com/golang/glog"
		"github.com/openshift/origin/pkg/util/proc"
		"github.com/prometheus/client_golang/prometheus"
		"github.com/prometheus/client_golang/prometheus/promhttp"
		_ "k8s.io/client-go/plugin/pkg/client/auth"
		"k8s.io/client-go/tools/clientcmd"
	
		/*kcollectors "k8s.io/kube-state-metrics/pkg/collectors"*/
		/*"k8s.io/kube-state-metrics/pkg/options"
		"k8s.io/kube-state-metrics/pkg/version"
		"k8s.io/kube-state-metrics/pkg/whiteblacklist"
		*/
		
		"sort"
     	"k8s.io/client-go/rest"
		metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    	kubeclientset "k8s.io/client-go/kubernetes"
	    /*clientset "github.com/openshift/client-go/quota/clientset/versioned"*/
	    /*oapiclientset "github.com/openshift/client-go"*/

	)


const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

var (
	defaultCollectors = collectorSet{
	}
	availableCollectors = map[string]func(registry prometheus.Registerer, kubeClient kubeclientset.Interface, namespace string){
	}

	defaultCollectorsOApi = collectorSet{
		"appliedclusterresourcequotas":         struct{}{},
		"deploymentconfigs":         struct{}{},
	}
	availableCollectorsOApi = map[string]func(registry prometheus.Registerer, kubeConfig *rest.Config, namespace string){
		"appliedclusterresourcequotas":         RegisterAppliedClusterResourceQuotaCollectorOApi,
		"deploymentconfigs": RegisterDeploymentConfigCollectorOApi,
	}

	ScrapeErrorTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ksm_scrape_error_total",
			Help: "Total scrape errors encountered when scraping a resource",
		},
		[]string{"resource"},
	)

	ResourcesPerScrapeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "ksm_resources_per_scrape",
			Help: "Number of resources returned per scrape",
		},
		[]string{"resource"},
	)	
)

type collectorSet map[string]struct{}

func (c *collectorSet) String() string {
	s := *c
	ss := s.asSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

func (c *collectorSet) Set(value string) error {
	s := *c
	cols := strings.Split(value, ",")
	for _, col := range cols {
		_, ok1 := availableCollectors[col]
		_, ok2 := availableCollectorsOApi[col]
		if !(ok1 || ok2) {
			glog.Fatalf("Collector \"%s\" does not exist", col)
		}
		s[col] = struct{}{}
	}
	return nil
}

func (c collectorSet) asSlice() []string {
	cols := []string{}
	for col, _ := range c {
		cols = append(cols, col)
	}
	return cols
}

func (c collectorSet) isEmpty() bool {
	return len(c.asSlice()) == 0
}

func (c *collectorSet) Type() string {
	return "string"
}


func main() {
	opts := NewOptions()
	opts.AddFlags()

	err := opts.Parse()
	if err != nil {
		glog.Fatalf("Error: %s", err)
	}

	if opts.Version {
		fmt.Printf("%#v\n", GetVersion())
		os.Exit(0)
	}

	if opts.Help {
		opts.Usage()
		os.Exit(0)
	}


	var collectors collectorSet
	if len(opts.Collectors) == 0 {
		glog.Info("Using default collectors")
		collectors = defaultCollectorsOApi
	} else {
		collectors = opts.Collectors
	}

	if opts.Namespace == metav1.NamespaceAll {
		glog.Info("Using all namespace")
	} else {
		glog.Infof("Using %s namespace", opts.Namespace)
	}

	/*if isNotExists(opts.Kubeconfig)  {
		glog.Fatalf("kubeconfig invalid and --in-cluster is false; kubeconfig must be set to a valid file(kubeconfig default file name: $HOME/.kube/config)")
	}
	if opts.Apiserver != "" {
		glog.Infof("apiserver set to: %v", opts.Apiserver)
	}
	*/

	proc.StartReaper()

	/*	kubeClientConfig, err := createOApiClient(opts.inCluster, opts.apiserver, opts.kubeconfig) */
	kubeClient, err := createKubeClient(opts.Apiserver, opts.Kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	kubeClientConfig, err := createKubeConfig(opts.Apiserver, opts.Kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to create Kube Config: %v", err)
	}

	/* TODO: update Scrape metrics! */
	ksmMetricsRegistry := prometheus.NewRegistry()
	ksmMetricsRegistry.Register(ResourcesPerScrapeMetric)
	ksmMetricsRegistry.Register(ScrapeErrorTotalMetric)
	ksmMetricsRegistry.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	ksmMetricsRegistry.Register(prometheus.NewGoCollector())
	go telemetryServer(ksmMetricsRegistry, opts.TelemetryHost, opts.TelemetryPort)


	registry := prometheus.NewRegistry()
	registerCollectorsOApi(registry, kubeClientConfig, collectors, opts.Namespace)
	registerCollectors(registry, kubeClient, collectors, opts.Namespace)


	metricsServer(registry, opts.Port)
}


func isNotExists(file string) bool {
	if file == "" {
		file = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	}
	_, err := os.Stat(file)
	return os.IsNotExist(err)
}


/* createKubeConfig: create rest.Config as base for creation clientsets
  Note: OAPI only provides very specifiy clientsets,
  the specify clients are created in the object collectors Register... method */
func createKubeConfig(apiserver string, kubeconfig string) (config *rest.Config, err error) {
	config, err = clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.UserAgent = GetVersion().String()
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := kubeclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	glog.Infof("Testing communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}
	glog.Infof("Running with Kubernetes cluster version: v%s.%s. git version: %s. git tree state: %s. commit: %s. platform: %s",
		v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)
	glog.Infof("Communication with server successful")

	return config, nil

}

/*  createKubeClient: create generic client for Kubernetes API*/
func createKubeClient(apiserver string, kubeconfig string) (kubeclientset.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.UserAgent = GetVersion().String()
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := kubeclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	glog.Infof("Testing communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}
	glog.Infof("Running with Kubernetes cluster version: v%s.%s. git version: %s. git tree state: %s. commit: %s. platform: %s",
		v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)
	glog.Infof("Communication with server successful")

	return kubeClient, nil
}

func metricsServer(registry prometheus.Gatherer, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := fmt.Sprintf(":%d", port)

	glog.Infof("Starting metrics server: %s", listenAddress)

	mux := http.NewServeMux()

	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Kube Metrics Server</title></head>
             <body>
             <h1>Kube Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
             <li><a href='` + healthzPath + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(listenAddress, mux))
}

func telemetryServer(registry prometheus.Gatherer, host string, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := net.JoinHostPort(host, strconv.Itoa(port))

	glog.Infof("Starting kube-state-metrics self metrics server: %s", listenAddress)

	mux := http.NewServeMux()

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Kube-State-Metrics Metrics Server</title></head>
             <body>
             <h1>Kube-State-Metrics Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
			 </ul>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(listenAddress, mux))
}

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	glog.Error(v...)
}


// registerCollectorsOApi creates specific OAPI Clients sets and starts informers if watch is supported
// otherwise the data is collected on demand in the Collect method of the object collector
// and initializes and registers metrics for collection.

func registerCollectorsOApi(registry prometheus.Registerer, kubeConfig *rest.Config, enabledCollectors collectorSet, namespace string) {
	activeCollectors := []string{}
	for c, _ := range enabledCollectors {
		f, ok := availableCollectorsOApi[c]
		if ok {
			f(registry, kubeConfig, namespace)
			activeCollectors = append(activeCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
}

// registerCollectors creates and starts informers and initializes and
// registers metrics for collection via Kubernetes API.

func registerCollectors(registry prometheus.Registerer, kubeClient kubeclientset.Interface, enabledCollectors collectorSet, namespace string) {
	activeCollectors := []string{}
	for c, _ := range enabledCollectors {
		f, ok := availableCollectors[c]
		if ok {
			f(registry, kubeClient, namespace)
			activeCollectors = append(activeCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
}
