# Build the manager binary
FROM golang:1.21 as builder


WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
#RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Copy the go source
COPY . .


ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=$TARGETPLATFORM alpine
WORKDIR /controller
COPY --from=builder /workspace/manager .
COPY  deploypkg/  /pkg/nodedeploy/

USER root:root

ENTRYPOINT ["/controller/manager"]
