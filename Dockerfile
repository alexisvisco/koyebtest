# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init ./cmd/init

# Runtime stage
FROM nginx:alpine

# Install fcgiwrap and bash for CGI support
RUN apk add --no-cache fcgiwrap bash spawn-fcgi busybox file

COPY --from=builder /app/init /usr/local/bin/init

RUN mkdir -p /app/scripts /app/configs

# Create the startup script
COPY <<EOF /app/startup.sh
#!/bin/bash
set -e

URL=\${URL:-\${DOWNLOAD_URL:-""}}
IS_SCRIPT=\${IS_SCRIPT:-"false"}

# Validate required environment variables
if [ -z "\$URL" ]; then
    exit 1
fi

cd /app

# Build downloader command arguments
INIT_ARGS="--url=\$URL"

if [ "\$IS_SCRIPT" = "true" ]; then
    INIT_ARGS="\$INIT_ARGS --script"
fi

/usr/local/bin/init \$INIT_ARGS

if [ "\$IS_SCRIPT" = "true" ]; then
    spawn-fcgi -s /var/run/fcgiwrap.socket -M 0666 /usr/bin/fcgiwrap &
    sleep 2
fi

rm -f /etc/nginx/conf.d/default.conf
cp /app/nginx.conf /etc/nginx/conf.d/default.conf

# Start nginx in foreground
nginx -g "daemon off;"
EOF

# Make startup script executable
RUN chmod +x /app/startup.sh

EXPOSE 80

# Environment variables (can be overridden at runtime)
ENV URL="https://pastebin.com/raw/hEFbnx33"
ENV IS_SCRIPT="false"

ENTRYPOINT ["/app/startup.sh"]
