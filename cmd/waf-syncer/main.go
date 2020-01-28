package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/spf13/pflag"

	"github.com/evry-bergen/waf-syncer/pkg/config"
	"github.com/evry-bergen/waf-syncer/pkg/director"

	istio "github.com/evry-bergen/waf-syncer/pkg/clients/istio/clientset/versioned"
	istioInformers "github.com/evry-bergen/waf-syncer/pkg/clients/istio/informers/externalversions"

	azureNetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

	"go.uber.org/zap"
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
		zap.S().Info("using in-cluster configuration")
		return rest.InClusterConfig()
	} else {
		zap.S().Infof("using configuration from '%s'", kubeconfig)
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

func newAzureClient() *azureNetwork.ApplicationGatewaysClient {
	agClient := azureNetwork.NewApplicationGatewaysClient(viper.GetString("azure_subscription_id"))

	// create an authorizer from env vars or Azure Managed Service Idenity
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		agClient.Authorizer = authorizer
	}
	return &agClient
}

func main() {
	config.Pflag()

	viper.BindPFlags(pflag.CommandLine)
	viper.AutomaticEnv()

	// logger, _ := zap.NewProduction()
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// creates the connection
	kubeConfig := viper.GetString(config.KubeConfig)
	master := viper.GetString(config.Ks8MasterUrl)

	config, err := clientcmd.BuildConfigFromFlags(master, kubeConfig)
	if err != nil {
		zap.S().Error(err)
	}

	// creates the clientset
	clientset := newGenericClientset(config)
	istioSet := newIstioClientSet(config)

	var azureAgClient *azureNetwork.ApplicationGatewaysClient
	azureAgClient = newAzureClient()

	stopCh := StopCh()

	gatewayInformerFactory := newIstioInformerFactory(config)
	gatewayInformer := gatewayInformerFactory.Networking().V1alpha3().Gateways()

	director := director.NewDirector(clientset, istioSet, azureAgClient, gatewayInformer)

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
