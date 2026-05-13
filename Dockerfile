FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/web ./cmd/web

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
	&& addgroup -S app && adduser -S app -G app
USER app
WORKDIR /home/app

COPY --from=builder /bin/web ./web

EXPOSE 8080
CMD ["./web"]
