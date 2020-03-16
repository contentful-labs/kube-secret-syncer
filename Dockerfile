# Build the manager binary
FROM golang:1.14 as base

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Test image
FROM base as test

RUN curl -sL https://go.kubebuilder.io/dl/2.2.0/linux/amd64 | tar -xz -C /tmp/ && \
  mv /tmp/kubebuilder_2.2.0_linux_amd64 /usr/local/kubebuilder

COPY Makefile Makefile
COPY hack/ hack/
COPY config/ config/

ENV PATH=$PATH:/usr/local/kubebuilder/bin

FROM base as builder

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/manager .
ENTRYPOINT ["/manager"]
