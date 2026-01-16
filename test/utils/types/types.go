// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Test utility package
package types

// Constants for component testing following catalog pattern
const (
	// Port forwarding configuration
	RestAddressPortForward = "127.0.0.1"
	PortForwardLocalPort   = "8080"
	PortForwardRemotePort  = "8080"

	// Default test organization and project
	SampleOrg     = "sample-org"
	SampleProject = "sample-project"

	// Orchestrator service endpoints
	CatalogServiceEndpoint = "/catalog.orchestrator.apis/v3"
	TenantServiceEndpoint  = "/tenant.orchestrator.apis/v3"
)
