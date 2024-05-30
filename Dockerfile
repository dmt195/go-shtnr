# Start with the official Golang image as the base
FROM golang:1.22-alpine as builder

# Install necessary dependencies for go-sqlite3
RUN apk add --no-cache build-base

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the working directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Start a new stage from scratch
FROM alpine:latest

# Install necessary libraries for go-sqlite3
RUN apk add --no-cache ca-certificates libc6-compat

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Copy the views directory
COPY --from=builder /app/views ./views

# Copy the static directory
COPY --from=builder /app/static ./static

# Copy the database file
COPY --from=builder /app/data ./data/

# Expose port 3003 to the outside world
EXPOSE 3003

# Command to run the executable
CMD ["./main"]
