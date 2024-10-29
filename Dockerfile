# Use a lightweight base image with Go
FROM golang:1.22.4-alpine AS builder

# Install Go dependencies only
WORKDIR /app
COPY go.mod ./
RUN go mod download

# Copy the application code and build
COPY . .
RUN go build -o gtts-service

# Final stage: Create a smaller image for running the app
FROM alpine:3.18

# Install FFmpeg, Python3, pip, and any other dependencies
RUN apk add --no-cache python3 py3-pip ffmpeg

# Install gTTS directly
RUN pip install gTTS

# Copy the built application binary
COPY --from=builder /app/gtts-service /app/gtts-service

# Set the working directory
WORKDIR /app

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./gtts-service"]