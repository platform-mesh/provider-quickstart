FROM --platform=$BUILDPLATFORM golang:1.26@sha256:f96cc555eb8db430159a3aa6797cd5bae561945b7b0fe7d0e284c63a3b291609 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /workspace

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binaries
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o wild-west-controller ./cmd/wild-west/... && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o wild-west-init ./cmd/init/...

FROM gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240

WORKDIR /

COPY --from=builder /workspace/wild-west-controller /wild-west-controller
COPY --from=builder /workspace/wild-west-init /wild-west-init

USER 65532:65532

ENTRYPOINT ["/wild-west-controller"]
