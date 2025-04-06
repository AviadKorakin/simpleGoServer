# syntax=docker/dockerfile:1

########################
# Stage 1: Build the Binary
########################
FROM golang:1.24 AS builder
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code (including your .env file)
COPY . .

# Build the binary (adjust the command to point to your main package)
RUN CGO_ENABLED=0 GOOS=linux go build -o webmvc_employees cmd/webmvc_employees/main.go

########################
# Stage 2: Final Image
########################
FROM gcr.io/distroless/static-debian10:latest

WORKDIR /

# Set environment variable to signal app is running in a container
ENV DOCKERIZED=true

# Copy the built binary from the builder stage
COPY --from=builder /app/webmvc_employees .

# Copy the .env file from the builder stage.
COPY --from=builder /app/.env.docker .

# Expose the port your application listens on (adjust if needed)
EXPOSE 8080

# Run the application
ENTRYPOINT ["/webmvc_employees"]