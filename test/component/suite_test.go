// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/manager"
	"github.com/stretchr/testify/suite"
)

// ComponentTestSuite is the base test suite for component-level tests
type ComponentTestSuite struct {
	suite.Suite
	Config         config.Configuration
	Context        context.Context
	Cancel         context.CancelFunc
	PortForwardCmd map[string]*exec.Cmd
	TestTimeout    time.Duration
	CleanupFuncs   []func() error
}

// SetupSuite runs once before all tests in the component test suite
func (s *ComponentTestSuite) SetupSuite() {
	s.T().Log("🚀 Starting Component Test Suite Setup")

	// Set test timeout
	s.TestTimeout = 30 * time.Second
	s.Context = context.Background()

	// Set environment variables for in-cluster configuration to work in test environment
	os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1")
	os.Setenv("KUBERNETES_SERVICE_PORT", "6443")

	// Load test configuration
	s.Config = config.Configuration{
		HarborServer:          getEnvOrDefault("HARBOR_SERVER", "http://localhost:8080"),
		KeycloakServer:        getEnvOrDefault("KEYCLOAK_SERVER", "http://localhost:8081"),
		CatalogServer:         getEnvOrDefault("CATALOG_SERVER", "http://localhost:8082"),
		AdmServer:             getEnvOrDefault("ADM_SERVER", "https://adm.kind.internal"),
		ReleaseServiceBase:    getEnvOrDefault("RELEASE_SERVICE_BASE", "registry-rs.edgeorchestration.intel.com"),
		ManifestPath:          getEnvOrDefault("MANIFEST_PATH", "development/base-system"),
		ManifestTag:           getEnvOrDefault("MANIFEST_TAG", "edge-v1.1.0"),
		HarborNamespace:       getEnvOrDefault("HARBOR_NAMESPACE", "harbor"),
		HarborAdminCredential: getEnvOrDefault("HARBOR_ADMIN_CREDENTIAL", "harbor_admin"),
		NumberWorkerThreads:   1,                // Reduce worker threads for tests
		InitialSleepInterval:  1,                // Short retry interval for tests
		MaxWaitTime:           10 * time.Second, // Short max wait for tests
	}

	s.T().Log("📝 Test Configuration Loaded:")
	s.T().Logf("  Harbor Server: %s", s.Config.HarborServer)
	s.T().Logf("  Keycloak Server: %s", s.Config.KeycloakServer)
	s.T().Logf("  Catalog Server: %s", s.Config.CatalogServer)
	s.T().Logf("  Manifest Tag: %s", s.Config.ManifestTag)

	// Skip service readiness checks for mock-based component tests
	s.T().Log("⏳ Using mock-based testing (skipping service connectivity checks)...")

	s.T().Log("✅ Component Test Suite Setup Complete")
}

// SetupTest can be used for per-test setup if needed
func (s *ComponentTestSuite) SetupTest() {
	s.T().Log("Setting up individual test")
}

// TearDownTest cleans up after each test
func (s *ComponentTestSuite) TearDownTest() {
	s.T().Log("Tearing down individual test")
}

// TearDownSuite cleans up after the entire test suite
func (s *ComponentTestSuite) TearDownSuite() {
	s.T().Log("🧹 Running Component Test Suite Cleanup")

	// Run all cleanup functions
	for _, cleanup := range s.CleanupFuncs {
		if err := cleanup(); err != nil {
			s.T().Logf("Cleanup function failed: %v", err)
		}
	}

	// Stop port forwarding
	for name, cmd := range s.PortForwardCmd {
		if cmd != nil && cmd.Process != nil {
			s.T().Logf("Stopping port forwarding for %s", name)
			_ = cmd.Process.Kill()
		}
	}

	// Cancel context
	if s.Cancel != nil {
		s.Cancel()
	}

	s.T().Log("✅ Component Test Suite Cleanup Complete")
}

// Mock-based component tests don't require actual service connectivity

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// AddCleanup adds a cleanup function to be called during teardown
func (s *ComponentTestSuite) AddCleanup(cleanup func() error) {
	s.CleanupFuncs = append(s.CleanupFuncs, cleanup)
}

// CreateTestManager creates a manager for testing with proper initialization
func (s *ComponentTestSuite) CreateTestManager() *manager.Manager {
	mgr := manager.NewManager(s.Config)

	// Note: We cannot safely initialize the eventChan here as it's unexported
	// Tests should mock or avoid calling methods that require the channel
	s.T().Log("Created test manager (eventChan will be nil - avoid CreateProject/DeleteProject)")

	return mgr
}

// TestComponentTestSuite runs the component test suite
func TestComponentTestSuite(t *testing.T) {
	suite.Run(t, &ComponentTestSuite{})
}
