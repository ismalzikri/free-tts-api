# Use a lightweight base image with Go
FROM golang:1.22.4-alpine AS builder

# Install gTTS CLI dependencies
RUN apk add --no-cache python3 py3-pip ffmpeg

# Create a virtual environment and install gTTS there
RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install gTTS

# Set the working directory
WORKDIR /app

# Copy Go modules and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application
RUN go build -o gtts-service

# Final stage: Create a smaller image for running the app
FROM alpine:3.18

# Install FFmpeg for audio processing
RUN apk add --no-cache ffmpeg python3 py3-pip

# Copy the built application binary and virtual environment with gTTS
COPY --from=builder /app/gtts-service /app/gtts-service
COPY --from=builder /opt/venv /opt/venv

# Set the working directory
WORKDIR /app

# Set the Python path to use the virtual environment
ENV PATH="/opt/venv/bin:$PATH"

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./gtts-service"]