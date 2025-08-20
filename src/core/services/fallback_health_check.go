package services

import (
	"fmt"
	"io"
	"net/http"
	"payment-processor/config"
	"time"
)

func GetFallbackServiceStatusData() ([]byte, error) {
	configs := config.LoadConfig()

	client := &http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: false,
		},
	}
	url := configs.Services.FallbackHealthCheckURL

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service health check failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return body, nil
}
