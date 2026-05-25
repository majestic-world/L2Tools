.PHONY: build manifest

build: manifest
	go build -ldflags="-s -w" -o ./build/GameServer.exe ./cmd/GameServer
	go build -ldflags="-s -w" -o ./build/Builder.exe ./cmd/InterfaceBuilder

manifest:
	cd cmd/InterfaceBuilder && goversioninfo -o rsrc.syso
	cd cmd/GameServer && goversioninfo -o rsrc.syso
