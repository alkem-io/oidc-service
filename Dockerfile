# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.24

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
WORKDIR /workspace
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . ./
ARG GIT_SHA="dev"
ARG TARGETARCH
RUN apk add --no-cache git && \
    echo "$GIT_SHA" > /tmp/.gitsha && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w" -o bin/oidc-service ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot AS runtime
WORKDIR /app
COPY --from=build /workspace/bin/oidc-service /app/oidc-service
ENV OIDC_LOG_LEVEL=info
EXPOSE 8080
ENTRYPOINT ["/app/oidc-service"]
