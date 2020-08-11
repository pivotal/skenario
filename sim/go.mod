module skenario

go 1.12

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.10.2 // indirect
	github.com/bvinc/go-sqlite-lite v0.6.1
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/go-containerregistry v0.0.0-20190222233527-d3e6a441f49f // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/josephburnett/sk-plugin v0.0.0-20190726113842-f4cc79709047
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nyarly/spies v0.0.0-20200413230442-112961b2b018 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/sclevine/agouti v3.0.0+incompatible
	github.com/sclevine/spec v1.4.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/apimachinery v0.0.0-20190117220443-572dfc7bdfcb
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/josephburnett/sk-plugin => ../plugin
