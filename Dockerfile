# Use a lightweight base image with Go
FROM golang:1.22.4-alpine AS builder

# Install gTTS CLI dependencies
RUN apk add --no-cache python3 py3-pip ffmpeg

# Install gTTS
RUN pip install gTTS

# Set the working directory
WORKDIR /app

# Copy Go modules and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application
RUN go build -o gtts-service

# Add specific permissions to avoid file access issues
RUN chmod +x /app/gtts-service

# Final stage: Create a smaller image for running the app
FROM alpine:3.18

# Install FFmpeg for audio processing
RUN apk add --no-cache ffmpeg

# Copy the built application binary and gTTS installation
COPY --from=builder /app/gtts-service /app/gtts-service
COPY --from=builder /usr/bin/ffmpeg /usr/bin/ffmpeg
COPY --from=builder /usr/local/lib/python3.11 /usr/local/lib/python3.11
COPY --from=builder /usr/bin/python3 /usr/bin/python3
COPY --from=builder /usr/local/bin/gtts-cli /usr/local/bin/gtts-cli

# Set the working directory
WORKDIR /app

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./gtts-service"]