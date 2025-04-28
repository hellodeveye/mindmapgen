# Stage 1: Build the Go application
FROM golang:1.23.4-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Set the Go proxy
RUN go env -w GOPROXY=https://goproxy.cn,direct

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# -ldflags="-w -s" reduces the size of the binary by removing debug information.
# CGO_ENABLED=0 prevents the usage of C libraries (useful for Alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o /app/mindmapgen ./main.go

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/mindmapgen /app/mindmapgen

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/app/mindmapgen"] 