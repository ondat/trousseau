FROM golang:1.17 as dependency
WORKDIR /work
ADD ./go.* ./
RUN go mod download

FROM dependency as build
WORKDIR /work
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubernetes-kms-vault cmd/kubernetes-kms-vault/main.go

FROM gcr.io/distroless/static:nonroot-amd64
LABEL org.opencontainers.image.title Trousseau for HashiCorp Vault \
    org.opencontainers.image.description "Trousseau, an open-source project leveraging the Kubernetes KMS provider framework to connect any Key Management Service the Kubernetes native way" \
    org.opencontainers.image.source https://github.com/ondat/trousseau \
    org.opencontainers.image.version v1.1.0 \
    org.opencontainers.image.base.name ghcr.io/ondat/trousseau:v1.1.0 \
    org.opencontainers.image.documentation https://github.com/ondat/trousseau/wiki 
    
COPY --from=build /work/kubernetes-kms-vault /bin/
ENTRYPOINT ["/bin/kubernetes-kms-vault"]
