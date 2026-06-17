FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 go build -o webhook-mcp-gateway .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/webhook-mcp-gateway /
EXPOSE 8080
ENTRYPOINT ["/webhook-mcp-gateway"]
