# --- Build Stage ---
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed.
RUN go mod download

# Copy the rest of the application's source code
COPY . .

# Build the application.
# -o /app/main specifies the output file name and location.
# -ldflags="-w -s" strips debugging information, reducing the binary size.
# CGO_ENABLED=0 creates a statically-linked binary, which doesn't depend on system libraries.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main -ldflags="-w -s" .

# --- Final Stage ---
# Use a minimal base image for the final container to reduce the attack surface and size.
# alpine is a good choice for its small size.
FROM alpine:latest

RUN apk update && apk add --no-cache docker-cli

# Set the working directory
WORKDIR /app

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/main .

# IMPORTANT: This application needs to communicate with the Docker daemon on the host.
# You will mount the Docker socket from the host to the container when you run it.
# This Dockerfile does not expose any ports, as it's a TUI application that runs in the terminal,
# not a web server.

# The command to run the application when the container starts.
CMD ["./main"]
