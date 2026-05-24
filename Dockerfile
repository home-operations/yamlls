FROM golang:1.26-alpine AS build
WORKDIR /src
# ca-certificates so the scratch stage can verify TLS to ghcr.io,
# helm-chart repos, OCI registries, etc.
RUN apk add --no-cache ca-certificates upx
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/yamlls ./cmd/yamlls
RUN upx --best --lzma /out/yamlls

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/yamlls /yamlls
ENTRYPOINT ["/yamlls"]
