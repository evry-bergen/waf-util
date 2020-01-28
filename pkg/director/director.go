package director

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/evry-bergen/waf-syncer/pkg/config"
	"github.com/evry-bergen/waf-syncer/pkg/crypto"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/spf13/viper"

	istioApiv1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"

	istio "github.com/evry-bergen/waf-syncer/pkg/clients/istio/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/evry-bergen/waf-syncer/pkg/clients/istio/informers/externalversions/istio/v1alpha3"

	azureNetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/client-go/kubernetes"

	"go.uber.org/zap"
	sslMate "software.sslmate.com/src/go-pkcs12"
)

// Director - struct for convenience
type Director struct {
	AzureWafConfig        *config.AzureWafConfig
	AzureAGClient         *azureNetwork.ApplicationGatewaysClient
	ClientSet             *kubernetes.Clientset
	IstioClient           *istio.Clientset
	GatewayInformer       v1alpha3.GatewayInformer
	GatewayInformerSynced cache.InformerSynced

	CurrentTargets map[string]TerminationTarget
}

// Run - run it
func (d *Director) Run(stop <-chan struct{}) {
	zap.S().Info("Starting application synchronization")

	if !cache.WaitForCacheSync(stop, d.GatewayInformerSynced) {
		zap.S().Error("timed out waiting for cache sync")
		return
	}

	go d.syncWAFLoop(stop)
}

func (d *Director) add(gw interface{}) {
	// zap.S().Infof("Add: %s", gw)
	d.update(nil, gw)
}

func (d *Director) update(old interface{}, new interface{}) {
	var gw, previous *istioApiv1alpha3.Gateway

	if old != nil {
		previous = old.(*istioApiv1alpha3.Gateway)
	}

	if previous != nil {
		// fmt.Printf("%s", previous)
	}

	gw = new.(*istioApiv1alpha3.Gateway)

	for _, srv := range gw.Spec.Servers {
		if srv.TLS != nil {
			zap.S().Info("Found TLS enabled port")

			for _, host := range srv.Hosts {
				secretName := srv.TLS.CredentialName

				target := TerminationTarget{
					Host:      host,
					Secret:    secretName,
					Target:    d.AzureWafConfig.BackendPool,
					Namespace: gw.Namespace,
				}

				zap.S().Debugf("Adding for %s for configuration with secret %s", host, secretName)
				d.CurrentTargets[host] = target
			}
		}
	}
}

func resourceRef(id string) *azureNetwork.SubResource {
	return &azureNetwork.SubResource{ID: to.StringPtr(id)}
}

func (d *Director) syncTargetsToWAF(waf *azureNetwork.ApplicationGateway) {
	wdPrefix := d.AzureWafConfig.ListenerPrefix
	secretCertMap := map[string]*v1.Secret{}
	listenersByName := map[string]azureNetwork.ApplicationGatewayHTTPListener{}
	/*
	 Loop through the settings we manage and keep the already existing
	 resources not prefixed by us.
	*/
	sslCertificates := d.certificatesToSync(waf)
	listeners := d.listenersToSync(waf, listenersByName)
	routingRules := d.rulesToSync(waf)

	/*
		We are looking at the current Targets aka VirtualGateways and their secrets, from this
		we get the information needed to create AG Listeners and their Certifiates based on the k8s
		secrets.
	*/
	for host, target := range d.CurrentTargets {
		zap.S().Debugf("Syncing %s > %s", host, target.Target)
		listener, listenerName := d.targetListener(target, wdPrefix, waf, host)
		/*
			Tie the listener together to a routing rule which ties together a hostname based listener
			to a backend
		*/
		routingRule := d.targetRoutingRules(waf, listener, listenerName, target)

		routingRules = append(routingRules, routingRule)
		listeners = append(listeners, listener)

		// Prefix all secrets with wdPrefix so we can track which ones we own
		secret, err := d.secret(target)
		if err != nil {
			zap.S().Infof("Error getting secret for listener %s, not added to listener list", host)
			zap.S().Error(err)
			continue
		}
		secretCertMap[target.generateSecretName(wdPrefix)] = secret
	}

	for secretName, secret := range secretCertMap {
		zap.S().Debugf("Converting certificate %s", secretName)

		wrapper, err := crypto.ParseSecretToCertContainer(secret)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		certPfx, err := sslMate.Encode(rand.Reader, wrapper.PrivateKey, wrapper.Certificates[0], wrapper.CACertificates, "azure")
		if err != nil {
			zap.S().Errorf("Error constructing PFX for %s", secretName)
			zap.S().Error(err)
			continue
		}

		certB64 := base64.StdEncoding.EncodeToString(certPfx)
		agCert := azureNetwork.ApplicationGatewaySslCertificate{
			Etag: to.StringPtr(""),
			Name: to.StringPtr(secretName),
			ApplicationGatewaySslCertificatePropertiesFormat: &azureNetwork.ApplicationGatewaySslCertificatePropertiesFormat{
				Data:     to.StringPtr(certB64),
				Password: to.StringPtr("azure"),
			},
		}

		sslCertificates = append(sslCertificates, agCert)
	}

	waf.HTTPListeners = &listeners
	waf.SslCertificates = &sslCertificates
	waf.RequestRoutingRules = &routingRules

	zap.S().Debugf("Have %d certificatesToSync", len(*waf.SslCertificates))
}

func (d *Director) secret(target TerminationTarget) (*v1.Secret, error) {
	secret, err := d.ClientSet.CoreV1().Secrets(target.Namespace).Get(target.Secret, metav1.GetOptions{})
	return secret, err
}

func (d *Director) targetRoutingRules(waf *azureNetwork.ApplicationGateway, listener azureNetwork.ApplicationGatewayHTTPListener, listenerName string, target TerminationTarget) azureNetwork.ApplicationGatewayRequestRoutingRule {
	httpListenerSubResource := azureNetwork.SubResource{ID: to.StringPtr(fmt.Sprintf("%s/httpListeners/%s", *waf.ID, *listener.Name))}
	routingRule := azureNetwork.ApplicationGatewayRequestRoutingRule{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(listenerName),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &azureNetwork.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			RuleType:            azureNetwork.Basic,
			HTTPListener:        &httpListenerSubResource,
			BackendAddressPool:  resourceRef(fmt.Sprintf("%s/backendAddressPools/%s", *waf.ID, target.Target)),
			BackendHTTPSettings: resourceRef(fmt.Sprintf("%s/backendHttpSettingsCollection/%s", *waf.ID, viper.GetString("azure_waf_backend_http_settings"))),
		},
	}
	return routingRule
}

func (d *Director) targetListener(target TerminationTarget, wdPrefix string, waf *azureNetwork.ApplicationGateway, host string) (azureNetwork.ApplicationGatewayHTTPListener, string) {
	listener := azureNetwork.ApplicationGatewayHTTPListener{}
	listenerName := target.generateNameWithPrefix(wdPrefix)
	listener.Name = &listenerName

	frontendIPRef := resourceRef(*(*waf.FrontendIPConfigurations)[0].ID)
	listener.ApplicationGatewayHTTPListenerPropertiesFormat = &azureNetwork.ApplicationGatewayHTTPListenerPropertiesFormat{
		FrontendIPConfiguration: frontendIPRef,
		FrontendPort:            resourceRef(fmt.Sprintf("%s/frontEndPorts/%s", *waf.ID, viper.GetString("azure_waf_frontend_port"))),
		HostName:                to.StringPtr(host),
		Protocol:                azureNetwork.HTTPS,
		SslCertificate:          resourceRef(fmt.Sprintf("%s/sslCertificates/%s", *waf.ID, target.generateSecretName(wdPrefix))),
	}
	return listener, listenerName
}

func (d *Director) rulesToSync(waf *azureNetwork.ApplicationGateway) []azureNetwork.ApplicationGatewayRequestRoutingRule {
	routingRules := []azureNetwork.ApplicationGatewayRequestRoutingRule{}
	for _, rr := range *waf.RequestRoutingRules {
		if d.HasNotPrefix(*rr.Name) {
			routingRules = append(routingRules, rr)
		} else {
			zap.S().Debugf("Skipping %s", *rr.Name)
		}
	}
	return routingRules
}

func (d *Director) listenersToSync(waf *azureNetwork.ApplicationGateway, listenersByName map[string]azureNetwork.ApplicationGatewayHTTPListener) []azureNetwork.ApplicationGatewayHTTPListener {
	listeners := []azureNetwork.ApplicationGatewayHTTPListener{}
	for _, l := range *waf.HTTPListeners {
		if d.HasNotPrefix(*l.Name) {
			listeners = append(listeners, l)
			listenersByName[*l.Name] = l
		} else {
			zap.S().Debugf("Skipping %s", *l.Name)
		}
	}
	return listeners
}

func (d *Director) certificatesToSync(waf *azureNetwork.ApplicationGateway) []azureNetwork.ApplicationGatewaySslCertificate {
	sslCertificates := []azureNetwork.ApplicationGatewaySslCertificate{}
	for _, sslCert := range *waf.SslCertificates {
		if d.HasNotPrefix(*sslCert.Name) {
			sslCertificates = append(sslCertificates, sslCert)
		} else {
			zap.S().Debugf("Skipping %s", *sslCert.Name)

		}
	}
	return sslCertificates
}

func (d *Director) HasNotPrefix(name string) bool {
	wdPrefix := d.AzureWafConfig.ListenerPrefix
	return !strings.HasPrefix(name, wdPrefix)
}

func (d *Director) syncWAFLoop(stop <-chan struct{}) {
	agName := viper.GetString(config.AzureWafName)
	agRgName := viper.GetString(config.AzureWafRg)

	var (
		updateFuture azureNetwork.ApplicationGatewaysCreateOrUpdateFuture
	)

	for {
		waf, err := d.AzureAGClient.Get(context.Background(), agRgName, agName)
		if err != nil {
			zap.S().Error(err)
			zap.S().Infof("Error getting WAF %s %s", agRgName, agName)
			goto sleep
		}

		// var future azureNetwork.ApplicationGatewaysCreateOrUpdateFuture
		if *waf.ProvisioningState == "Updating" {
			zap.S().Debugf("WAF is updating, sleeping.")
			goto sleep
		}

		d.syncTargetsToWAF(&waf)
		zap.S().Info("Updating WAF")

		updateFuture, err = d.AzureAGClient.CreateOrUpdate(context.Background(), agRgName, agName, waf)
		if err != nil {
			zap.S().Error(err)
		}

		err = updateFuture.WaitForCompletionRef(context.Background(), d.AzureAGClient.Client)
		if err != nil {
			zap.S().Error(err)
		}
		zap.S().Info("Successfully updated WAF")

	sleep:
		time.Sleep(time.Second * 5)
	}
}

// NewDirector - Creates a new instance of the director
func NewDirector(k8sClient *kubernetes.Clientset, istioClient *istio.Clientset, agClient *azureNetwork.ApplicationGatewaysClient, gwInformer v1alpha3.GatewayInformer) *Director {

	azureConfig := config.NewAzureConfig()
	director := &Director{
		AzureWafConfig:        azureConfig,
		AzureAGClient:         agClient,
		ClientSet:             k8sClient,
		IstioClient:           istioClient,
		GatewayInformer:       gwInformer,
		GatewayInformerSynced: gwInformer.Informer().HasSynced,
		CurrentTargets:        make(map[string]TerminationTarget),
	}

	gwInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(newPod interface{}) {
				director.add(newPod)
			},
			UpdateFunc: func(oldGw, newGw interface{}) {
				director.update(oldGw, newGw)
			},
		})

	return director
}
