FROM golang@sha256:ec67c62f48ddfbca1ccaef18f9b3addccd707e1885fa28702a3954340786fcf6 as dependency
WORKDIR /work
ADD ./go.* ./
RUN go mod download

FROM dependency as build
WORKDIR /work
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubernetes-kms-vault cmd/kubernetes-kms-vault/main.go

FROM gcr.io/distroless/static@sha256:957bbd91e4bfe8186bd218c08b2bbc5c852e6ebe6a7b2dcc42a86b22ea2b6bb6
LABEL org.opencontainers.image.title Trousseau for HashiCorp Vault \
    org.opencontainers.image.description "Trousseau, an open-source project leveraging the Kubernetes KMS provider framework to connect any Key Management Service the Kubernetes native way" \
    org.opencontainers.image.source https://github.com/ondat/trousseau \
    org.opencontainers.image.version v1.1.0 \
    org.opencontainers.image.base.name ghcr.io/ondat/trousseau:v1.1.0 \
    org.opencontainers.image.documentation https://github.com/ondat/trousseau/wiki 
    
COPY --from=build /work/kubernetes-kms-vault /bin/
ENTRYPOINT ["/bin/kubernetes-kms-vault"]
