# Multi-arch friendly container for podman / docker
FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/lab ./cmd/lab

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/lab /usr/local/bin/lab
COPY webui /app/webui
COPY scripts /app/scripts
COPY extensions/README.md /app/extensions/README.md
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/lab", "serve", "--port", "8787"]
