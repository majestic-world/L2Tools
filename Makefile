.PHONY: build manifest

build: manifest
	go build -ldflags="-s -w" -o ./build ./cmd/GameServer
	go build -ldflags="-s -w" -o ./build ./cmd/InterfaceBuilder

manifest:
	cd cmd/InterfaceBuilder && goversioninfo -o rsrc.syso
	cd cmd/GameServer && goversioninfo -o rsrc.syso
