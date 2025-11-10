// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
)

type InitPlugin struct{}

func (p *InitPlugin) CreateEvent(_ context.Context, _ Event, pluginData PluginData) error {
	(*pluginData)[HarborTokenName] = "token"
	(*pluginData)[HarborUsernameName] = "user"
	return nil
}

func (p *InitPlugin) Name() string {
	return "init"
}

func (p *InitPlugin) Initialize(_ context.Context, _ PluginData) error {
	return nil
}

func (p *InitPlugin) DeleteEvent(_ context.Context, _ Event, _ PluginData) error { return nil }

func (s *PluginsTestSuite) TestCatalogProvisionerPluginCreate() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	CatalogFactory = newTestCatalog

	initPlugin := &InitPlugin{}
	Register(initPlugin)

	plugin, err := NewCatalogProvisionerPlugin(config.Configuration{ReleaseServiceRootURL: `oci://release-service-root.root.io`})
	s.NoError(err, "Cannot create catalog provisioner plugin")
	s.NotNil(plugin)

	Register(plugin)

	err = Initialize(ctx)
	s.NoError(err, "Cannot initialize plugins")

	err = Dispatch(ctx, Event{
		EventType:    "create",
		UUID:         "default",
		Organization: "test-org",
	}, nil)
	s.NoError(err, "Cannot dispatch create event")

	s.Len(mockCatalog.registries, 4)
	s.Equal("intel-rs-helm", mockCatalog.registries["intel-rs-helm"].Name)
	s.Equal(`Repo on registry release-service-root.root.io`, mockCatalog.registries["intel-rs-helm"].Description)
	s.Equal("intel-rs-images", mockCatalog.registries["intel-rs-images"].Name)
	s.Equal(`Repo on registry release-service-root.root.io`, mockCatalog.registries["intel-rs-images"].Description)
	s.Equal("harbor-helm-oci", mockCatalog.registries["harbor-helm-oci"].Name)
	s.Equal("harbor-docker-oci", mockCatalog.registries["harbor-docker-oci"].Name)
	s.Equal("token", mockCatalog.registries["harbor-docker-oci"].AuthToken)
	s.Equal("token", mockCatalog.registries["harbor-helm-oci"].AuthToken)
	s.Equal("user", mockCatalog.registries["harbor-docker-oci"].Username)
	s.Equal("user", mockCatalog.registries["harbor-helm-oci"].Username)
	s.Equal("/catalog-apps-test-org-", mockCatalog.registries["harbor-docker-oci"].RootURL)
	s.Equal("/catalog-apps-test-org-", mockCatalog.registries["harbor-helm-oci"].RootURL)
	s.Equal("use-dynamic-cacert", mockCatalog.registries["harbor-docker-oci"].Cacerts)
	s.Equal("use-dynamic-cacert", mockCatalog.registries["harbor-helm-oci"].Cacerts)

	for _, reg := range mockCatalog.registries {
		s.Equal("default", reg.ProjectUUID)
	}
}

// TestCatalogWaitForCatalogSucceeds tests that waitForCatalog succeeds when catalog is available
func (s *PluginsTestSuite) TestCatalogWaitForCatalogSucceeds() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Use working catalog factory
	CatalogFactory = newTestCatalog

	plugin, err := NewCatalogProvisionerPlugin(config.Configuration{
		CatalogServer: "localhost:8080",
	})
	s.NoError(err, "Should create catalog provisioner plugin")

	// waitForCatalog should succeed immediately with mock
	err = plugin.waitForCatalog(ctx)
	s.NoError(err, "waitForCatalog should succeed when catalog is available")
}

// TestCatalogWaitForCatalogFailsAfterRetries tests that waitForCatalog fails after max retries
func (s *PluginsTestSuite) TestCatalogWaitForCatalogFailsAfterRetries() {
	// Skip this test unless explicitly requested as it takes time
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestCatalogWaitForCatalogFailsAfterRetries -timeout 2m")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	failingAttempts := 0
	CatalogFactory = func(_ config.Configuration) (Catalog, error) {
		failingAttempts++
		return nil, fmt.Errorf("catalog factory failed (attempt %d)", failingAttempts)
	}

	plugin := &CatalogProvisionerPlugin{
		config: config.Configuration{
			CatalogServer: "invalid-catalog:9999",
		},
	}

	startTime := time.Now()
	err := plugin.waitForCatalog(ctx)
	duration := time.Since(startTime)

	s.Error(err, "waitForCatalog should fail when catalog is not available")
	s.Contains(err.Error(), "catalog not available after", "Error should indicate max retries exceeded")
	s.Greater(failingAttempts, 1, "Should attempt multiple times")
	s.Less(duration, time.Minute*2, "Should not exceed context timeout")
}

// TestCatalogWaitForCatalogRecoversAfterRetries tests that waitForCatalog succeeds after some failures
func (s *PluginsTestSuite) TestCatalogWaitForCatalogRecoversAfterRetries() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	attempts := 0
	// Create a mock catalog that fails initially then succeeds
	mockCatalog := &mockDynamicCatalog{
		listRegistriesFunc: func(_ context.Context) error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("catalog not ready yet (attempt %d)", attempts)
			}
			// After 3 attempts, succeed
			return nil
		},
	}

	CatalogFactory = func(_ config.Configuration) (Catalog, error) {
		return mockCatalog, nil
	}

	plugin := &CatalogProvisionerPlugin{
		config: config.Configuration{
			CatalogServer: "localhost:8080",
		},
	}

	startTime := time.Now()
	err := plugin.waitForCatalog(ctx)
	duration := time.Since(startTime)

	s.NoError(err, "waitForCatalog should succeed after recovering")
	s.GreaterOrEqual(attempts, 3, "Should attempt at least 3 times before succeeding")
	s.Greater(duration, time.Second*10, "Should take time due to retries with backoff")
	s.T().Logf("Recovered after %d attempts in %v", attempts, duration)
}

// TestCatalogWaitForVaultSucceeds tests that waitForVault succeeds when vault is available
func (s *PluginsTestSuite) TestCatalogWaitForVaultSucceeds() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Use working catalog factory
	CatalogFactory = newTestCatalog

	plugin, err := NewCatalogProvisionerPlugin(config.Configuration{
		VaultServer: "localhost:8200",
	})
	s.NoError(err, "Should create catalog provisioner plugin")

	// waitForVault should succeed immediately with mock
	err = plugin.waitForVault(ctx)
	s.NoError(err, "waitForVault should succeed when vault is available")
}

// TestCatalogWaitForVaultFailsAfterRetries tests that waitForVault fails after max retries
func (s *PluginsTestSuite) TestCatalogWaitForVaultFailsAfterRetries() {
	// Skip this test unless explicitly requested as it takes time
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestCatalogWaitForVaultFailsAfterRetries -timeout 2m")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	// Use factory that returns error
	failingAttempts := 0
	CatalogFactory = func(_ config.Configuration) (Catalog, error) {
		failingAttempts++
		return nil, fmt.Errorf("vault connection failed (attempt %d)", failingAttempts)
	}

	plugin := &CatalogProvisionerPlugin{
		config: config.Configuration{
			VaultServer: "invalid-vault:9999",
		},
	}

	startTime := time.Now()
	err := plugin.waitForVault(ctx)
	duration := time.Since(startTime)

	s.Error(err, "waitForVault should fail when vault is not available")
	s.Contains(err.Error(), "vault not available after", "Error should indicate max retries exceeded")
	s.Greater(failingAttempts, 1, "Should attempt multiple times")
	s.Less(duration, time.Minute*2, "Should not exceed context timeout")
}

// TestCatalogWaitForVaultRecoversAfterRetries tests that waitForVault succeeds after some failures
func (s *PluginsTestSuite) TestCatalogWaitForVaultRecoversAfterRetries() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	attempts := 0
	// Create a mock catalog that fails initially then succeeds
	mockCatalog := &mockDynamicCatalog{
		initializeClientSecretFunc: func(_ context.Context) (string, error) {
			attempts++
			if attempts < 4 {
				return "", fmt.Errorf("vault not ready yet (attempt %d)", attempts)
			}
			// After 4 attempts, succeed
			return "", nil
		},
	}

	CatalogFactory = func(_ config.Configuration) (Catalog, error) {
		return mockCatalog, nil
	}

	plugin := &CatalogProvisionerPlugin{
		config: config.Configuration{
			VaultServer: "localhost:8200",
		},
	}

	startTime := time.Now()
	err := plugin.waitForVault(ctx)
	duration := time.Since(startTime)

	s.NoError(err, "waitForVault should succeed after recovering")
	s.GreaterOrEqual(attempts, 4, "Should attempt at least 4 times before succeeding")
	s.Greater(duration, time.Second*15, "Should take time due to retries with backoff")
	s.T().Logf("Vault recovered after %d attempts in %v", attempts, duration)
}

// TestCatalogInitializeFailsWhenVaultFails tests that Initialize propagates vault errors
func (s *PluginsTestSuite) TestCatalogInitializeFailsWhenVaultFails() {
	// Skip this test unless explicitly requested as it takes time
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestCatalogInitializeFailsWhenVaultFails -timeout 2m")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	// Use factory that always fails
	CatalogFactory = func(_ config.Configuration) (Catalog, error) {
		return nil, fmt.Errorf("vault connection permanently failed")
	}

	plugin := &CatalogProvisionerPlugin{
		config: config.Configuration{
			VaultServer:   "invalid-vault:9999",
			CatalogServer: "localhost:8080",
		},
	}

	err := plugin.Initialize(ctx, nil)
	s.Error(err, "Initialize should fail when vault fails")
	s.Contains(err.Error(), "catalog initialization failed during vault check", "Error should indicate vault check failure")
}

// TestCatalogInitializeFailsWhenCatalogFails tests that Initialize propagates catalog errors
func (s *PluginsTestSuite) TestCatalogInitializeFailsWhenCatalogFails() {
	// Skip this test unless explicitly requested as it takes time
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestCatalogInitializeFailsWhenCatalogFails -timeout 2m")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	vaultCallCount := 0
	catalogCallCount := 0
	CatalogFactory = func(_ config.Configuration) (Catalog, error) {
		// First call is for vault - succeed
		if vaultCallCount == 0 {
			vaultCallCount++
			return newTestCatalog(config.Configuration{})
		}
		// Subsequent calls are for catalog - fail
		catalogCallCount++
		return nil, fmt.Errorf("catalog connection failed (attempt %d)", catalogCallCount)
	}

	plugin := &CatalogProvisionerPlugin{
		config: config.Configuration{
			VaultServer:   "localhost:8200",
			CatalogServer: "invalid-catalog:9999",
		},
	}

	err := plugin.Initialize(ctx, nil)
	s.Error(err, "Initialize should fail when catalog fails")
	s.Contains(err.Error(), "catalog initialization failed during catalog check", "Error should indicate catalog check failure")
}
