FROM golang@sha256:5b75b529da0f2196ee8561a90e5b99aceee56e125c6ef09a3da4e32cf3cc6c20 AS base
ADD . /work
WORKDIR /work
RUN (cd proxy ; go mod download)
RUN (cd providers/debug ; go mod download)
RUN (cd providers/vault ; go mod download)
RUN (cd providers/awskms ; go mod download)
RUN (cd trousseau ; go mod download)

FROM base AS worker
ARG PROJECT=required
ADD . /work
WORKDIR /work/${PROJECT}
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o endpoint main.go

FROM gcr.io/distroless/static@sha256:957bbd91e4bfe8186bd218c08b2bbc5c852e6ebe6a7b2dcc42a86b22ea2b6bb6
ARG PROJECT=required
LABEL org.opencontainers.image.title "Trousseau - ${PROJECT}" 
LABEL org.opencontainers.image.vendor "Trousseau.io" 
LABEL org.opencontainers.image.licenses "Apache-2.0 License" 
LABEL org.opencontainers.image.source "https://github.com/ondat/trousseau" 
LABEL org.opencontainers.image.description "Trousseau, an open-source project leveraging the Kubernetes KMS provider framework to connect any Key Management Service the Kubernetes native way" 
LABEL org.opencontainers.image.documentation "https://github.com/ondat/trousseau/wiki" 
    
COPY --from=worker /work/${PROJECT}/endpoint /bin/
ENTRYPOINT ["/bin/endpoint"]
