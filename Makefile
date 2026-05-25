.PHONY: build manifest

build: manifest
	go build -ldflags="-s -w" -o ./build/GameServer.exe ./cmd/GameServer
	go build -ldflags="-s -w" -o ./build/Builder.exe ./cmd/InterfaceBuilder
	go build -ldflags="-s -w" -o ./build/ProGuard.exe ./cmd/ProGuard

manifest:
	cd cmd/InterfaceBuilder && goversioninfo -o rsrc.syso
	cd cmd/GameServer && goversioninfo -o rsrc.syso
	cd cmd/ProGuard && goversioninfo -o rsrc.syso
