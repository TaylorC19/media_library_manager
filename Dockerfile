FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN export CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=$TARGETARCH; go build -trimpath -o /bin/web ./cmd/web \
    && go build -trimpath -o /bin/users ./cmd/users \
    && go build -trimpath -ldflags "-X main.version=${VERSION}" -o /bin/mlm-setup ./cmd/setup

FROM alpine:3.20 AS runtime
ARG VERSION=dev
ARG REVISION=unknown
LABEL org.opencontainers.image.source="https://github.com/TaylorC19/media_library_manager" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"
RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S app -G app
WORKDIR /home/app
USER app

FROM runtime AS setup
COPY --from=builder /bin/mlm-setup /usr/local/bin/mlm-setup
ENTRYPOINT ["/usr/local/bin/mlm-setup"]

FROM runtime AS app
COPY --from=builder /bin/web ./web
COPY --from=builder /bin/users ./users
COPY --from=builder /bin/mlm-setup /usr/local/bin/mlm-setup
EXPOSE 8080
CMD ["./web"]
