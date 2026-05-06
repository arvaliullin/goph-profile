FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /avatard ./cmd/avatard

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /avatard /avatard
COPY migrations ./migrations
CMD ["/avatard"]
