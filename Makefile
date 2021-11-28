IMAGE ?= ghcr.io/Trousseau-io/trousseau:dev
GO_BUILD_CMD = go build -v 
GO_ENV = CGO_ENABLED=0

all: build

.PHONY: build

build: 
	@echo "Building trousseau binaries"
	$(GO_ENV) $(GO_BUILD_CMD) -o ./build/trousseau ./cmd/kubernetes-kms-main.go 

docker-build:
	docker build --no-cache . -f Dockerfile -t $(IMAGE)

docker-push:
	docker push $(IMAGE)

clean:
	rm -rf ./build 

