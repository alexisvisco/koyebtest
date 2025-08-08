package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	fileOutput  = "output"
	nginxConfig = "nginx.conf"
)

func main() {
	logger := slog.With("component", "init")

	flagIsScript := flag.Bool("script", false, "If set to true it will execute the script in the url")
	flagUrl := flag.String("url", "", "Script to downloadFromURL")

	flag.Parse()

	if *flagUrl == "" {
		logger.Error("url parameter is required")
		os.Exit(1)
	}

	parsedURL, err := url.Parse(*flagUrl)
	if err != nil {
		logger.Error("failed to parse url", "error", err, "url", *flagUrl)
		os.Exit(1)
	}

	var writer io.Writer
	var file *os.File

	file, err = os.Create(fileOutput)
	if err != nil {
		logger.Error("failed to create output file", "error", err, "filename", fileOutput)
		os.Exit(1)
	}
	defer file.Close()
	writer = file

	err = downloadFromURL(parsedURL, writer)
	if err != nil {
		logger.Error("failed to download content from url", "error", err, "url", parsedURL.String())
		os.Exit(1)
	}

	_ = file.Close()

	if *flagIsScript {
		err = os.Chmod(fileOutput, 0755)
		if err != nil {
			logger.Error("failed to make file executable", "error", err, "filename", fileOutput)
			os.Exit(1)
		}
	}

	var configWriter io.Writer
	var configFile *os.File

	configFile, err = os.Create(nginxConfig)
	if err != nil {
		logger.Error("failed to create nginx config file", "error", err, "filename", nginxConfig)
		os.Exit(1)
	}
	defer configFile.Close()
	configWriter = configFile

	err = generateNginxConfig(*flagIsScript, configWriter)
	if err != nil {
		logger.Error("failed to generate nginx configuration", "error", err)
		os.Exit(1)
	}

	if *flagIsScript {
		err = createCGIWrapper()
		if err != nil {
			logger.Error("failed to create cgi wrapper", "error", err)
			os.Exit(1)
		}
	}
}

// generateNginxConfig generates an nginx configuration based on whether the file is a script or not.
// if isScript it will execute the script at the flagOutput using cgi at each request
// if not it serve the static file at the flagOutput
func generateNginxConfig(isScript bool, outConfig io.Writer) error {
	var config string

	if isScript {
		// Configure nginx to execute the script using CGI at root endpoint with wrapper
		config = `server {
    listen 80;
    server_name localhost;
    
    # Execute script at root endpoint
    location = / {
        root /app;
        fastcgi_pass unix:/var/run/fcgiwrap.socket;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME /app/wrapper.sh;
        fastcgi_param SCRIPT_NAME /;
        fastcgi_param REQUEST_METHOD $request_method;
        fastcgi_param CONTENT_TYPE $content_type;
        fastcgi_param CONTENT_LENGTH $content_length;
    }
    
    error_log /var/log/nginx/error.log;
    access_log /var/log/nginx/access.log;
}`
	} else {
		// Configure nginx to serve the static file
		config = `server {
    listen 80;
    server_name localhost;
    
    # Serve the specific downloaded file at root
    location = / {
       root /app;
       try_files /{{output}} =404;
       add_header Content-Type 'text/plain';
    }
    
    error_log /var/log/nginx/error.log;
    access_log /var/log/nginx/access.log;
}`
	}

	// Write the config to the provided writer
	_, err := outConfig.Write([]byte(strings.ReplaceAll(config, "{{output}}", fileOutput)))
	if err != nil {
		return fmt.Errorf("failed to write nginx config: %w", err)
	}

	return nil
}

// createCGIWrapper creates a wrapper script that adds CGI headers and executes the downloaded script
// It is needed because otherwise nginx will not display the output of the script
func createCGIWrapper() error {
	wrapperContent := `#!/bin/sh
echo "Content-Type: text/plain"
echo ""
/bin/sh /app/` + fileOutput + ` 2>&1
`

	err := os.WriteFile("wrapper.sh", []byte(wrapperContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}

	return nil
}

func downloadFromURL(parsedURL *url.URL, writer io.Writer) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Koyebtest")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response to output: %w", err)
	}

	return nil
}
