package main

import (
	"context"
	"net/http"
	"os"
	"time"
)

const requestTimeout = 3 * time.Second

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8080/health", nil)
	if err != nil {
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		os.Exit(1)
	}
}
