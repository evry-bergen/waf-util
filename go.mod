module github.com/evry-bergen/waf-util

require (
	github.com/Azure/azure-sdk-for-go v27.0.0+incompatible
	github.com/Azure/go-autorest v11.7.0+incompatible
	github.com/evry-ace/waf-util v0.0.0-20190403075839-588c8898dc52
	github.com/knative/pkg v0.0.0-20190328184255-c35005418bb2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.2.1
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
	gopkg.in/inf.v0 v0.9.1 // indirect
	istio.io/fortio v1.3.1
	k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc
	k8s.io/client-go v2.0.0-alpha.0.0.20190115164855-701b91367003+incompatible
	k8s.io/code-generator v0.0.0-20181128191024-b1289fc74931
	software.sslmate.com/src/go-pkcs12 v0.0.0-20190322163127-6e380ad96778
)

replace github.com/knative/pkg => ./knative-pkg
