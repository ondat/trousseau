FROM golang:1.17 as dependency
WORKDIR /work
ADD ./go.* ./
RUN go mod download

FROM dependency as build
WORKDIR /work
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hello-world cmd/hello-world/main.go


FROM gcr.io/distroless/static:nonroot-amd64
LABEL org.opencontainers.image.source https://github.com/Trousseau-io/trousseau

COPY --from=build /work/hello-world /bin/
ENTRYPOINT ["/bin/hello-world"] 
