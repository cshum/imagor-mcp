# A one-line install path: the server is a static Go binary, so the image carries the
# binary and its certificates and nothing else.
FROM golang:1.27-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/imagor-mcp ./cmd/imagor-mcp

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/imagor-mcp /usr/local/bin/imagor-mcp
ENTRYPOINT ["/usr/local/bin/imagor-mcp"]
