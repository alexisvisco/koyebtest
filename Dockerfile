# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init ./cmd/init

# Runtime stage
FROM nginx:alpine

# Install fcgiwrap and bash for CGI support
RUN apk add --no-cache fcgiwrap bash spawn-fcgi busybox file

# Copy the built application
COPY --from=builder /app/init /usr/local/bin/init

# Create directories for scripts and configs
RUN mkdir -p /app/scripts /app/configs

# Copy startup script
COPY <<EOF /app/startup.sh
#!/bin/bash
set -e

# Environment variables with defaults - support both URL and DOWNLOAD_URL
URL=\${URL:-\${DOWNLOAD_URL:-""}}
IS_SCRIPT=\${IS_SCRIPT:-"false"}

# Validate required environment variables
if [ -z "\$URL" ]; then
    echo "Error: URL or DOWNLOAD_URL environment variable is required"
    exit 1
fi

echo "Downloading from: \$URL"
echo "Is script: \$IS_SCRIPT"

# Change to working directory where files will be created
cd /app

# Build downloader command arguments
INIT_ARGS="--url=\$URL"

if [ "\$IS_SCRIPT" = "true" ]; then
    INIT_ARGS="\$INIT_ARGS --script"
fi

# Run the init (creates 'output', 'nginx.conf', and 'wrapper.sh' if needed)
/usr/local/bin/init \$INIT_ARGS

echo "Files created:"
ls -la

if [ "\$IS_SCRIPT" = "true" ]; then
    echo "Content of downloaded script:"
    head -10 output

    echo "Content of generated wrapper:"
    cat wrapper.sh

    echo "Starting fcgiwrap for CGI support"
    spawn-fcgi -s /var/run/fcgiwrap.socket -M 0666 /usr/bin/fcgiwrap &

    # Wait a moment for fcgiwrap to start
    sleep 2

    echo "Testing wrapper execution:"
    /app/wrapper.sh || echo "Wrapper test failed"
else
    echo "Content of downloaded file:"
    head -10 output
fi

rm -f /etc/nginx/conf.d/default.conf
cp /app/nginx.conf /etc/nginx/conf.d/default.conf

echo "Using generated nginx config:"
cat /etc/nginx/conf.d/default.conf
echo

echo "Starting nginx..."
# Start nginx in foreground
nginx -g "daemon off;"
EOF

# Make startup script executable
RUN chmod +x /app/startup.sh

# Expose port 80
EXPOSE 80

# Environment variables (can be overridden at runtime)
ENV URL="https://pastebin.com/raw/hEFbnx33"
ENV IS_SCRIPT="false"

# Use startup script as entrypoint
ENTRYPOINT ["/app/startup.sh"]
