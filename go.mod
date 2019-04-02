module github.com/evry-ace/waf-util

require (
	contrib.go.opencensus.io/exporter/ocagent v0.4.10 // indirect
	github.com/Azure/azure-sdk-for-go v27.0.0+incompatible
	github.com/Azure/go-autorest v11.7.0+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/knative/pkg v0.0.0-20190328184255-c35005418bb2
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.2.1
	github.com/stretchr/testify v1.3.0 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190313024323-a1f597ede03a // indirect
	golang.org/x/net v0.0.0-20190318221613-d196dffd7c2b // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	istio.io/fortio v1.3.1
	k8s.io/api v0.0.0-20190226173710-145d52631d00
	k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc
	k8s.io/client-go v2.0.0-alpha.0.0.20190115164855-701b91367003+incompatible
	k8s.io/code-generator v0.0.0-20181128191024-b1289fc74931
	k8s.io/gengo v0.0.0-20190308184658-b90029ef6cd8 // indirect
	k8s.io/klog v0.2.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190306001800-15615b16d372 // indirect
	software.sslmate.com/src/go-pkcs12 v0.0.0-20190322163127-6e380ad96778
)

replace github.com/knative/pkg => ../knative-pkg
