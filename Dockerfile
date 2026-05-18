FROM golang:1.24 AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-crypto-arb-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-crypto-arb-tui ./cmd/tui

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/go-crypto-arb-api /usr/local/bin/go-crypto-arb-api
COPY --from=build /out/go-crypto-arb-tui /usr/local/bin/go-crypto-arb-tui
COPY configs/config.example.yaml /app/configs/config.example.yaml

ENV CONFIG_PATH=/app/configs/config.yaml
ENV HTTP_ADDR=:8080
EXPOSE 8080
CMD ["go-crypto-arb-api"]
