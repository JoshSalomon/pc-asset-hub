FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -tags postgres -o /api-server ./cmd/api-server/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /api-server /api-server
EXPOSE 8080
ENTRYPOINT ["/api-server"]
