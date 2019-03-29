package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evry-ace/waf-util/pkg/director"

	istio "github.com/evry-ace/waf-util/pkg/clients/istio/clientset/versioned"
	istioInformers "github.com/evry-ace/waf-util/pkg/clients/istio/informers/externalversions"

	"go.uber.org/zap"
	"istio.io/fortio/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig string

type Syncer struct {
	ClientSet kubernetes.Clientset
}

func getK8sConfig() (*rest.Config, error) {
	if kubeconfig == "" {
		log.Infof("using in-cluster configuration")
		return rest.InClusterConfig()
	} else {
		log.Infof("using configuration from '%s'", kubeconfig)
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
}

func newIstioInformerFactory(kubeconfig *rest.Config) istioInformers.SharedInformerFactory {
	config, err := istio.NewForConfig(kubeconfig)
	if err != nil {
		zap.S().Panic("unable to create naiserator clientset")
	}
	return istioInformers.NewSharedInformerFactory(config, time.Second*30)
}

func newIstioClientSet(kubeconfig *rest.Config) *istio.Clientset {
	clientSet, err := istio.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create new clientset")
	}

	return clientSet
}

func newGenericClientset(kubeconfig *rest.Config) *kubernetes.Clientset {
	cs, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return cs
}

func main() {
	// logger, _ := zap.NewProduction()
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		zap.S().Error(err)
	}

	// creates the clientset
	clientset := newGenericClientset(config)

	istioSet := newIstioClientSet(config)

	stopCh := StopCh()

	gatewayInformerFactory := newIstioInformerFactory(config)
	gatewayInformer := gatewayInformerFactory.Networking().V1alpha3().Gateways()

	director := director.NewDirector(clientset, istioSet, gatewayInformer)

	gatewayInformerFactory.Start(stopCh)
	director.Run(stopCh)
	<-stopCh
}

func StopCh() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}
