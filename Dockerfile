# Build stage
FROM golang:1.22 as builder

# Enable Go modules and cgo
ENV CGO_ENABLED=1
ENV GO111MODULE=on

# Install libwebp
RUN apt-get update && \
    apt-get install -y libwebp-dev && \
    rm -rf /var/lib/apt/lists/*

# Set the current working directory inside the container
WORKDIR /go/src/proteggo_api

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the working Directory inside the container
COPY . .

# Build the Go app
RUN go build -a -installsuffix cgo -o /go/bin/proteggo_api .

# Start a new stage from Ubuntu 22.04
FROM ubuntu:22.04

# Install runtime dependencies and any other necessary packages, including CA certificates
RUN apt-get update && \
    apt-get install -y \
    libwebp-dev \
    ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy the pre-built binary file from the previous stage
COPY --from=builder /go/bin/proteggo_api /go/bin/proteggo_api

# Document that the service listens on port 8080.
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/go/bin/proteggo_api"]
