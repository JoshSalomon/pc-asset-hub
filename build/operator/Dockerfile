FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /operator ./cmd/operator/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /operator /operator
ENTRYPOINT ["/operator"]
