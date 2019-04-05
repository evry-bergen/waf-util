package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/spf13/pflag"

	"github.com/evry-bergen/waf-util/pkg/director"

	istio "github.com/evry-bergen/waf-util/pkg/clients/istio/clientset/versioned"
	istioInformers "github.com/evry-bergen/waf-util/pkg/clients/istio/informers/externalversions"

	azureNetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

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
	pflag.String("kubeconfig", "", "ABS path to kubeconfig")
	pflag.String("master", "", "k8s master url")
	pflag.String("azure_subscription_id", "", "Subscription to use where the WAF / AG is")

	pflag.String("azure_waf_rg", "", "The AG / WAF RG to use")
	pflag.String("azure_waf_name", "", "The AG / WAF instance to use")
	pflag.String("azure_waf_backend_pool", "", "The AG / WAF backend pool")
	pflag.String("azure_waf_frontend_port", "https", "The AG / WAF frontend port name")
	pflag.String("azure_waf_backend_http_settings", "", "The AG / WAF backend http settings name")

	viper.BindPFlags(pflag.CommandLine)
	viper.AutomaticEnv()

	// logger, _ := zap.NewProduction()
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// creates the connection
	kubeConfig := viper.GetString("kubeconfig")
	master := viper.GetString("master")

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
