// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"testing"
)

// SetUpAccessToken retrieves an access token from deployed Keycloak
// This follows the catalog pattern for authentication in component tests
func SetUpAccessToken(t *testing.T, keycloakServer string) string {
	// For component tests, this would normally make a real OAuth request
	// to the deployed Keycloak server to get an auth token

	// For now, return a placeholder token
	// In a real implementation, this would:
	// 1. Make OAuth client credentials request to keycloakServer
	// 2. Parse the response to extract the access token
	// 3. Return the token for use in subsequent API calls

	t.Logf("Getting access token from Keycloak server: %s", keycloakServer)

	// Placeholder implementation
	return "component-test-token"
}

// GetProjectID retrieves a project ID for the given project and organization
// This follows the catalog pattern for getting project context
func GetProjectID(_ context.Context, project, org string) (string, error) {
	// In real implementation, this would query the deployed orchestrator
	// to get the actual project UUID for the given project/org combination

	return fmt.Sprintf("test-project-%s-%s", org, project), nil
}
