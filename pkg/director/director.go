package director

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/evry-bergen/waf-util/pkg/crypto"
	"k8s.io/api/core/v1"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/spf13/viper"

	istio "github.com/evry-bergen/waf-util/pkg/clients/istio/clientset/versioned"
	istioApiv1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"

	"github.com/evry-bergen/waf-util/pkg/clients/istio/informers/externalversions/istio/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	azureNetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/client-go/kubernetes"

	"go.uber.org/zap"
	sslMate "software.sslmate.com/src/go-pkcs12"
)

type Director struct {
	AzureAGClient         *azureNetwork.ApplicationGatewaysClient
	ClientSet             *kubernetes.Clientset
	IstioClient           *istio.Clientset
	GatewayInformer       v1alpha3.GatewayInformer
	GatewayInformerSynced cache.InformerSynced

	CurrentTargets map[string]TerminationTarget
}

type TerminationTarget struct {
	Host      string
	Port      int
	Secret    string
	Namespace string
	Target    string
}

func (t *TerminationTarget) GenerateName() string {
	return fmt.Sprintf("%s-tls", t.Host)
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
					Target:    "test-beap",
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
	newListeners := []azureNetwork.ApplicationGatewayHTTPListener{}
	sslCertificates := []azureNetwork.ApplicationGatewaySslCertificate{}
	routingRules := []azureNetwork.ApplicationGatewayRequestRoutingRule{}
	secretCertMap := map[string]*v1.Secret{}

	listeners := *waf.HTTPListeners
	for host, target := range d.CurrentTargets {
		zap.S().Debugf("Syncing %s > %s", host, target.Target)

		var listener *azureNetwork.ApplicationGatewayHTTPListener
		for _, i := range listeners {
			if i.HostName != nil && *i.HostName == target.Host {
				zap.S().Debugf("Found listener")

				listener = &i
				break
			}
		}

		if listener == nil {
			zap.S().Infof("Creating listener for %s", target.Host)

			listener = &azureNetwork.ApplicationGatewayHTTPListener{}

			listenerName := target.GenerateName()
			listener.Name = &listenerName
		}

		frontendIPRef := resourceRef(*(*waf.FrontendIPConfigurations)[0].ID)

		listener.ApplicationGatewayHTTPListenerPropertiesFormat = &azureNetwork.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration: frontendIPRef,
			FrontendPort:            resourceRef(*waf.ID + "/frontEndPorts/https"),
			HostName:                to.StringPtr(host),
			Protocol:                azureNetwork.HTTPS,
			SslCertificate:          resourceRef(*waf.ID + fmt.Sprintf("/sslCertificates/%s", target.Namespace+"-"+target.Secret)),
		}

		secretName := target.Secret
		secret, err := d.ClientSet.CoreV1().Secrets(target.Namespace).Get(secretName, metav1.GetOptions{})

		if err != nil {
			zap.S().Infof("Error getting secret for listener %s, not added to listener list", host)
			zap.S().Error(err)
			continue
		}

		httpListenerSubResource := azureNetwork.SubResource{ID: to.StringPtr(fmt.Sprintf("%s/httpListeners/%s", *waf.ID, *listener.Name))}
		routingRule := azureNetwork.ApplicationGatewayRequestRoutingRule{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(target.GenerateName()),
			ApplicationGatewayRequestRoutingRulePropertiesFormat: &azureNetwork.ApplicationGatewayRequestRoutingRulePropertiesFormat{
				RuleType:            azureNetwork.Basic,
				HTTPListener:        &httpListenerSubResource,
				BackendAddressPool:  resourceRef(fmt.Sprintf("%s/backendAddressPools/%s", *waf.ID, target.Target)),
				BackendHTTPSettings: resourceRef(fmt.Sprintf("%s/backendHttpSettingsCollection/%s", *waf.ID, "test-be-htst")),
			},
		}

		routingRules = append(routingRules, routingRule)
		newListeners = append(newListeners, *listener)
		secretCertMap[secret.Namespace+"-"+secretName] = secret
	}

	for secretName, secret := range secretCertMap {
		wrapper, err := crypto.ParseSecretToCertContainer(secret)

		if err != nil {
			zap.S().Error(err)
			continue
		}

		fmt.Println(wrapper)
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

	waf.HTTPListeners = &newListeners
	waf.SslCertificates = &sslCertificates
	waf.RequestRoutingRules = &routingRules

	zap.S().Debugf("Have %d certificates", len(*waf.SslCertificates))
}

func (d *Director) syncWAFLoop(stop <-chan struct{}) {
	agName := viper.GetString("azure_waf_name")
	agRgName := viper.GetString("azure_waf_rg")

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

	director := &Director{
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
