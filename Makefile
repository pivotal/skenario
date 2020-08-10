goals = serve test build-plugin-k8s
.DEFAULT_GOAL : serve
.PHONY : $(goals)
.ONESHELL : $(goals)

run : build
	SKENARIO_PLUGIN=./build/plugin-k8s ./build/sim

build : build-sim build-plugin-k8s

build-sim :
	mkdir -p build
	cd sim
	go build -o ../build/sim ./cmd/skenario/main.go

build-plugin-k8s :
	mkdir -p build
	cd plugin-k8s
	go build -o ../build/plugin-k8s ./cmd/main.go

build-plugin-k8s-vpa :
	mkdir -p build
	cd plugin-k8s-vpa
	go build -o ../build/plugin-k8s-vpa ./cmd/main.go
