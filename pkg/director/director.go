package director

import (
	"fmt"

	istio "github.com/evry-ace/waf-util/pkg/clients/istio/clientset/versioned"
	istioApiv1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"

	"github.com/evry-ace/waf-util/pkg/clients/istio/informers/externalversions/istio/v1alpha3"
	"k8s.io/client-go/tools/cache"

	"k8s.io/client-go/kubernetes"

	"go.uber.org/zap"
)

type Director struct {
	ClientSet             *kubernetes.Clientset
	IstioClient           *istio.Clientset
	GatewayInformer       v1alpha3.GatewayInformer
	GatewayInformerSynced cache.InformerSynced
}

func (d *Director) Run(stop <-chan struct{}) {
	zap.S().Info("Starting application synchronization")

	if !cache.WaitForCacheSync(stop, d.GatewayInformerSynced) {
		zap.S().Error("timed out waiting for cache sync")
		return
	}
}

func (d *Director) add(gw interface{}) {
	// zap.S().Infof("Add: %s", gw)
	d.update(nil, gw)
}

func (d *Director) update(old interface{}, new interface{}) {
	// zap.S().Infof("Update:  gw %s", new)
	var gw, previous *istioApiv1alpha3.Gateway

	if old != nil {
		previous = old.(*istioApiv1alpha3.Gateway)
	}

	if previous != nil {
		// fmt.Printf("%s", previous)
	}
	gw = new.(*istioApiv1alpha3.Gateway)

	fmt.Printf("%s - %s\n", gw.Name, gw.Spec.Servers[0].Hosts)
}

func NewDirector(k8sClient *kubernetes.Clientset, istioClient *istio.Clientset, gwInformer v1alpha3.GatewayInformer) *Director {
	director := &Director{
		ClientSet:             k8sClient,
		IstioClient:           istioClient,
		GatewayInformer:       gwInformer,
		GatewayInformerSynced: gwInformer.Informer().HasSynced,
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
