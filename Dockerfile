# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:1.19 as builder

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY vendor/ vendor/
COPY cmd/ cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o multi-arch-resolver cmd/resolver/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.2
COPY --from=builder /opt/app-root/src/multi-arch-resolver /
USER 65532:65532

ENTRYPOINT ["/multi-arch-resolver"]
