package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	AzureWafListenerPrefix      = "azure_waf_listener_prefix"
	azureWafBackendHttpSettings = "azure_waf_backend_http_settings"
	azureWafFrontendPort        = "azure_waf_frontend_port"
	AzureWafBackendPool         = "azure_waf_backend_pool"
	AzureWafName                = "azure_waf_name"
	AzureWafRg                  = "azure_waf_rg"
	azureSubscriptionId         = "azure_subscription_id"
	Ks8MasterUrl                = "ks8MasterUrl"
	KubeConfig                  = "KubeConfig"
	UseIstioTls                 = "use_istio_tls"
	IstioTlsNamespace           = "istio_tls_namespace"
)

type AzureWafConfig struct {
	ListenerPrefix      string
	BackendHttpSettings string
	FrontendPort        string
	BackendPool         string
	Name                string
	ResourceGroup       string
	SubscriptionID      string
	UseIstioTls         bool
	IstioTlsNamespace   string
}

type Ks8Config struct {
	MasterUrl  string
	KubeConfig string
}

func NewAzureConfig() *AzureWafConfig {
	a := AzureWafConfig{
		ListenerPrefix:      viper.GetString(AzureWafListenerPrefix),
		BackendHttpSettings: viper.GetString(azureWafBackendHttpSettings),
		FrontendPort:        viper.GetString(azureWafFrontendPort),
		BackendPool:         viper.GetString(AzureWafBackendPool),
		Name:                viper.GetString(AzureWafName),
		ResourceGroup:       viper.GetString(AzureWafRg),
		UseIstioTls:         viper.GetBool(UseIstioTls),
		IstioTlsNamespace:   viper.GetString(IstioTlsNamespace),
		SubscriptionID:      "",
	}
	return &a
}

func Pflag() {
	pflag.String(KubeConfig, "", "ABS path to KubeConfig")
	pflag.String(Ks8MasterUrl, "", "k8s master url")
	pflag.String(azureSubscriptionId, "", "Subscription to use where the WAF / AG is")
	pflag.String(AzureWafRg, "", "The AG / WAF RG to use")
	pflag.String(AzureWafName, "", "The AG / WAF instance to use")
	pflag.String(AzureWafBackendPool, "", "The AG / WAF backend pool")
	pflag.String(azureWafFrontendPort, "https", "The AG / WAF frontend port name")
	pflag.String(azureWafBackendHttpSettings, "", "The AG / WAF backend http settings name")
	pflag.String(AzureWafListenerPrefix, "wd", "Prefix all WAF Director listeners with this")
	pflag.Bool(UseIstioTls, false, "get tls secret from istio namespace and not target namespace")
	pflag.String(IstioTlsNamespace, "istio-system", "Namespace where istio and tls certificates are deployed")
}
