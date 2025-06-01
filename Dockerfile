# Use an official Go runtime as a parent image
FROM golang:1.24-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# -o app will output the executable named 'app'
RUN CGO_ENABLED=0 GOOS=linux go build -v -o app .

# Start a new stage from scratch for a smaller image
FROM alpine:latest  

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/app .

# Copy static assets or templates if any (not currently in this project)
# COPY --from=builder /app/templates ./templates
# COPY --from=builder /app/static ./static

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./app"]
