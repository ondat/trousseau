TAG = v1.0.0-alpha
IMAGE ?= ghcr.io/trousseau-io/trousseau:$(TAG)
GO_BUILD_CMD = go build -v 
GO_ENV = CGO_ENABLED=0

all: build

.PHONY: build

build: 
        @echo "Building trousseau binaries"
        $(GO_ENV) $(GO_BUILD_CMD) -o ./build/kubernetes-kms-vault cmd/kubernetes-kms-vault/main.go 

docker-build:
        docker build --no-cache . -f Dockerfile -t $(IMAGE)

docker-push:
        docker push $(IMAGE)

clean:
        rm -rf ./build 
