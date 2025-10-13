// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ServiceCheck represents a service to check
type ServiceCheck struct {
	Name       string
	URL        string
	HealthPath string
}

// TestProject represents a test project for component tests
type TestProject struct {
	Name         string
	Namespace    string
	UUID         string
	Organization string
	Config       map[string]interface{}
}

// NewTestProject creates a new test project
func NewTestProject(name string) *TestProject {
	return &TestProject{
		Name:         name,
		Namespace:    "default",
		UUID:         "test-" + name,
		Organization: "test-org",
		Config:       make(map[string]interface{}),
	}
}

// CreateTestProject creates a test project with given configuration
func CreateTestProject(name string, config map[string]interface{}) *TestProject {
	return &TestProject{
		Name:         name,
		Namespace:    "default",
		UUID:         "test-" + name,
		Organization: "test-org",
		Config:       config,
	}
}

// WaitForService waits for a service to become available
func WaitForService(ctx context.Context, service ServiceCheck) error {
	client := &http.Client{
		Timeout: 3 * time.Second, // Shorter individual request timeout
	}

	checkURL := service.URL
	if service.HealthPath != "" {
		checkURL = service.URL + service.HealthPath
	}

	ticker := time.NewTicker(1 * time.Second) // Check more frequently
	defer ticker.Stop()

	// Try immediate check first
	resp, err := client.Get(checkURL)
	if err == nil && resp.StatusCode < 500 {
		resp.Body.Close()
		return nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Then wait with ticker
	attempts := 0
	maxAttempts := 30 // Maximum attempts before giving up

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s after %d attempts", service.Name, attempts)
		case <-ticker.C:
			attempts++
			if attempts > maxAttempts {
				return fmt.Errorf("max attempts (%d) reached for %s", maxAttempts, service.Name)
			}

			resp, err := client.Get(checkURL)
			if err == nil && resp.StatusCode < 500 {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}

			// Log every 10 attempts to show progress
			if attempts%10 == 0 {
				fmt.Printf("Still waiting for %s (attempt %d/%d)...\n", service.Name, attempts, maxAttempts)
			}
		}
	}
}
