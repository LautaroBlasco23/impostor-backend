# --- build stage ---
FROM golang:1.25-alpine AS builder
WORKDIR /app

# deps
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -o app ./cmd/server

# --- runtime stage ---
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/app /app/app

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/app"]
