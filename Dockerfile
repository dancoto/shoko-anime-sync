# ============================================================================
# STAGE 1: COMPILATION ENVIRONMENT
# ============================================================================
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /shoko-anime-sync

# ============================================================================
# STAGE 2: MINIMAL RUNTIME ENVIRONMENT
# ============================================================================
FROM alpine:latest

# Install secure CA Certificates and timezone database tables
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/
COPY --from=builder /shoko-anime-sync .
EXPOSE 3000
CMD ["./shoko-anime-sync"]
