# Use the official Go image as a builder
FROM golang:1.20 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy Go modules and install dependencies
COPY go.mod ./

# Copy the rest of the application code
COPY . .

# Build the application
RUN go build -o gtts-service

# Use a lightweight base image for running the app
FROM debian:bullseye-slim

# Install Python and gTTS CLI
RUN apt-get update && apt-get install -y python3 python3-pip && pip3 install gtts

# Copy the built binary from the builder
COPY --from=builder /app/gtts-service /gtts-service

# Expose port 8080
EXPOSE 8080

# Run the Go binary
CMD ["/gtts-service"]
