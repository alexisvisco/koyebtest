# Koyeb Test - Dynamic Service Deployment API

A REST API that creates dynamic Nginx services from URLs using HashiCorp Nomad for container orchestration. This project allows you to deploy both static content and executable scripts as containerized web services.

## Overview

This system implements the requirements for a service deployment API that:
- Provides a REST endpoint to create services
- Downloads content from URLs and serves it via Nginx
- Supports both static content and executable scripts
- Uses HashiCorp Nomad for container orchestration
- Provides automatic subdomain-based routing (bonus)

## Architecture

The system consists of several components:

1. **Main Go Application**: A reverse proxy and API server that manages job creation and routing
2. **Docker Image**: A custom nginx-based container that dynamically configures itself based on environment variables
3. **Nomad Integration**: Uses HashiCorp Nomad to deploy and manage containers
4. **Dynamic Nginx Configuration**: Automatically generates nginx configs for static files or CGI script execution

Note: We can use kata to have a secure environment for running scripts, but for simplicity, this implementation uses a standard Docker container.

## API Endpoints

### Create Job

The API endpoint requires a service name as a path parameter: `/services/{name}`

#### With Script execution

```bash
curl -X PUT http://api.koyebtest.alexisvis.co/services/my-service \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://pastebin.com/raw/UCVAQpD4",
    "is_script": true
  }'
```
**You can try it right now ! :)**


Response:
```json
{
  "url": "http://XXXXXXXXXXXXXX.koyebtest.alexisvis.co"
}
```


#### With Static content

```bash
curl -X PUT http://api.koyebtest.alexisvis.co/services/my-static-site \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://pastebin.com/raw/UCVAQpD4",
    "is_script": false
  }'
```
Response:
```json
{
  "url": "http://XXXXXXXXXXXXXX.koyebtest.alexisvis.co"
}
```


## Local Setup Instructions

### Prerequisites

1. **Go** (version 1.24 or later)
2. **Docker** (for building the nginx image)
3. **HashiCorp Nomad** (for container orchestration)

### Run nomad

Make sure you have Nomad running locally or in a cluster. You can start a local Nomad agent with:

```bash
nomad agent -dev -bind 0.0.0.0 -
network-interface='{{ GetDefaultInterfaces | attr "name" }}'
```

### Build and Run the Application

```bash
export HOST=127.0.0.1.nip.io
export API_HOST=api.127.0.0.1.nip.io

go run main.go
```

nip.io is used for dynamic DNS resolution, allowing you to access the service via subdomains without needing a real DNS setup.

### Call the API
```bash
curl -X PUT http://api.127.0.0.1.nip.io/services/my-service \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://pastebin.com/raw/UCVAQpD4",
    "is_script": true
  }'
```

Response :
```json
{"url":"http://3ffafdc6-d32d-42dd-9cf3-b7faa0884bcb.127.0.0.1.nip.io"}
```

## How It Works

1. **Job Creation**: When you call the API, it creates a unique Nomad job with a UUID as job id
2. **Container Deployment**: Nomad deploys the `alexisvisco/koyeb-nginx` Docker image with environment variables
3. **Content Download**: The container's `init` binary downloads content from the specified URL
4. **Dynamic Configuration**: Based on the `is_script` flag, it generates appropriate nginx configuration:
    - **Static content**: Serves the downloaded file directly
    - **Scripts**: Configures CGI with fcgiwrap to execute the script on each request
5. **Routing**: The main application routes subdomain requests to the appropriate container port

## Project Structure

```
koyebtest/
├── cmd/init/           # Go binary that downloads content and configures nginx
├── internal/
│   ├── handler/        # HTTP handlers for API endpoints
│   ├── service/        # Nomad job management service
│   └── types/          # Type definitions and interfaces
├── .github/workflows/  # GitHub Actions for Docker image building
├── Dockerfile          # Multi-stage Docker build for the nginx container
├── main.go            # Main application entry point
└── README.md          # This file
```

## Error Handling

The system includes comprehensive error handling:
- Invalid URLs are rejected with appropriate HTTP status codes
- Nomad job failures are logged and reported
- Container startup issues are detected and handled (via timeout)

## Security Considerations

- The system validates URLs before processing
- CGI execution is sandboxed within the container environment
- Each service gets its own container with limited CPU and memory

## Limitations

- Services are not persisted - they're lost when containers are stopped
- No HTTPS/TLS termination
- Limited resource monitoring and cleanup

## Future Enhancements

Potential improvements for production use:
- Add HTTPS/TLS support
- Add monitoring and observability
- Service lifecycle management (start/stop/restart)
- Support multi node deployments with Nomad
- Test E2E
