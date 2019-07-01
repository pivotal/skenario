module skenario

go 1.12

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.10.2 // indirect
	github.com/bvinc/go-sqlite-lite v0.6.1
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/go-containerregistry v0.0.0-20190623150931-ca8b66cb1b79 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/knative/pkg v0.0.0-20190701174718-7b4cf0bfe593 // indirect
	github.com/knative/serving v0.7.0 // indirect
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20190617123014-9caf012348a4 // indirect
	github.com/logrusorgru/aurora v0.0.0-20181002194514-a7b3b318ed4e
	github.com/sclevine/agouti v3.0.0+incompatible
	github.com/sclevine/spec v1.2.0
	github.com/stretchr/testify v1.3.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/text v0.3.2
	k8s.io/api v0.0.0-20190627205229-acea843d18eb
	k8s.io/apimachinery v0.0.0-20190629125103-05b5762916b3
	k8s.io/apiserver v0.0.0-20190701164347-9434caf4a4cb // indirect
	k8s.io/client-go v0.0.0-20190629125432-98902b2ea1c2
	k8s.io/metrics v0.0.0-20190627210813-f9a3814d33e8 // indirect
	knative.dev/pkg v0.0.0-20190627143708-1864f499dcaa
	knative.dev/serving v0.7.0
)

replace knative.dev/serving => /Users/jchesterpivotal/go/src/knative.dev/serving
