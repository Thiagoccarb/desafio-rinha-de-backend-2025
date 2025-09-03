package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"payment-processor/config"
)

func GetDefaultServiceStatusData(ctx context.Context) ([]byte, error) {
	configs := config.LoadConfig()

	client := &http.Client{}
	url := configs.Services.DefaultHealthCheckURL

	req, err := http.NewRequest("GET", url, nil)
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
