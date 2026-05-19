# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o myaku-api ./myaku-api

# Stage 2: Run
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/myaku-api .
COPY --from=builder /app/myaku-api/static ./static
EXPOSE 8081
CMD ["./myaku-api"]