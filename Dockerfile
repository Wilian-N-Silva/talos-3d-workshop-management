FROM golang:1.27-alpine AS build

ARG SERVER_VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/server ./cmd/server
COPY internal ./internal
COPY migrations ./migrations

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/buildinfo.ServerVersion=${SERVER_VERSION}" \
    -o /out/talos-server \
    ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup --system --gid 10001 talos \
    && adduser --system --disabled-password --no-create-home --uid 10001 --ingroup talos talos \
    && mkdir -p /var/lib/talos \
    && chown talos:talos /var/lib/talos

COPY --from=build /out/talos-server /usr/local/bin/talos-server

ENV TALOS_DATA_DIR=/var/lib/talos

USER talos
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/talos-server"]
