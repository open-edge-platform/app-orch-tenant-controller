// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/stretchr/testify/assert"
)

// mockDynamicADM is a mock ADM that allows dynamic behavior
type mockDynamicADM struct {
	listDeploymentNamesFunc func(ctx context.Context, tenant string) (map[string]string, error)
}

func (m *mockDynamicADM) ListDeploymentNames(ctx context.Context, tenant string) (map[string]string, error) {
	if m.listDeploymentNamesFunc != nil {
		return m.listDeploymentNamesFunc(ctx, tenant)
	}
	return map[string]string{}, nil
}

func (m *mockDynamicADM) CreateDeployment(_ context.Context, _ string, _ string, _ string, _ string, _ string, _ map[string]string) error {
	return nil
}

func (m *mockDynamicADM) DeleteDeployment(_ context.Context, _ string, _ string, _ string, _ string, _ string, _ bool) error {
	return nil
}

func (s *PluginsTestSuite) TestExtensionsPluginCreate() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	OrasFactory = NewTestOras
	CatalogFactory = newTestCatalog
	AppDeploymentFactory = newTestADM

	mockDeployments = map[string]*mockDeployment{}

	configuration := config.Configuration{
		HarborServer:      "https://harbor.org",
		KeycloakNamespace: "keycloak-ns",
		KeycloakSecret:    "sekret",
		AdmServer:         "http://admserver",
		ManifestPath:      "/registry/edge-node/en/manifest",
		ManifestTag:       "latest",
	}

	plugin, err := NewExtensionsProvisionerPlugin(configuration)
	s.NoError(err, "Cannot create extensions plugin")
	s.NotNil(plugin)

	RemoveAllPlugins()
	Register(plugin)
	err = Initialize(ctx)
	assert.NoError(s.T(), err, "Initialize plugin")

	err = Dispatch(ctx, Event{
		EventType: "create",
		UUID:      "foo",
	}, nil)
	s.NoError(err)

	s.Len(mockCatalog.uploadedFiles, 7)

	for fileName, file := range mockCatalog.uploadedFiles {
		s.Equal(fileName, file.path)
		s.True(file.lastUpload)
		s.Contains(file.artifact, `License-Identifier: Apache-2.0`)
	}

	s.Len(mockDeployments, 3)
	baselineKey := "base-extensions-0.2.0-baseline"
	s.Equal("base-extensions", mockDeployments[baselineKey].name)
	s.Equal("0.2.0", mockDeployments[baselineKey].version)
	s.Equal("baseline", mockDeployments[baselineKey].profileName)
	s.Equal("blue", mockDeployments[baselineKey].labels["color"])

	privKey := "base-extensions-0.2.0-privileged"
	s.Equal("base-extensions", mockDeployments[privKey].name)
	s.Equal("0.2.0", mockDeployments[privKey].version)
	s.Equal("privileged", mockDeployments[privKey].profileName)
	s.Equal("green", mockDeployments[privKey].labels["color"])
}

func (s *PluginsTestSuite) TestExtensionsPluginDeleteDeployment() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	OrasFactory = NewTestOras
	CatalogFactory = newTestCatalog
	AppDeploymentFactory = newTestADM

	// prepopulate mockDeployments with three deployments

	// nolint:gofmt
	mockDeployments = map[string]*mockDeployment{
		"base-extensions-0.2.0-baseline": &mockDeployment{
			name:        "base-extensions",
			version:     "0.2.0",
			profileName: "baseline",
			labels:      map[string]string{"color": "blue"},
		},
		"base-extensions-0.2.0-restricted": &mockDeployment{
			name:        "base-extensions",
			version:     "0.2.0",
			profileName: "restricted",
			labels:      map[string]string{"color": "red"},
		},
		"base-extensions-0.2.0-privileged": &mockDeployment{
			name:        "base-extensions",
			version:     "0.2.0",
			profileName: "privileged",
			labels:      map[string]string{"color": "green"},
		},
	}

	// create a manifest that deletes one of the deployments

	manifest := `# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
---
metadata:
  schemaVersion: 0.3.0
  release: 1.2.0
lpke:
  deploymentList:
    - dpName: base-extensions
      dpProfileName: baseline
      dpVersion: 0.2.0
      allAppTargetClusters:
        - key: color
          val: blue
      desiredState: absent`

	configuration := config.Configuration{
		HarborServer:      "https://harbor.org",
		KeycloakNamespace: "keycloak-ns",
		KeycloakSecret:    "sekret",
		AdmServer:         "http://admserver",
		ManifestPath:      "/registry/edge-node/en/manifest",
		ManifestTag:       "latest",
		UseLocalManifest:  manifest,
	}

	plugin, err := NewExtensionsProvisionerPlugin(configuration)
	s.NoError(err, "Cannot create extensions plugin")
	s.NotNil(plugin)

	RemoveAllPlugins()
	Register(plugin)
	err = Initialize(ctx)
	assert.NoError(s.T(), err, "Initialize plugin")

	err = Dispatch(ctx, Event{
		EventType: "create",
		UUID:      "foo",
	}, nil)
	s.NoError(err)

	s.Len(mockDeployments, 2)

	resKey := "base-extensions-0.2.0-restricted"
	s.Equal("base-extensions", mockDeployments[resKey].name)
	s.Equal("0.2.0", mockDeployments[resKey].version)
	s.Equal("restricted", mockDeployments[resKey].profileName)
	s.Equal("red", mockDeployments[resKey].labels["color"])

	privKey := "base-extensions-0.2.0-privileged"
	s.Equal("base-extensions", mockDeployments[privKey].name)
	s.Equal("0.2.0", mockDeployments[privKey].version)
	s.Equal("privileged", mockDeployments[privKey].profileName)
	s.Equal("green", mockDeployments[privKey].labels["color"])
}

func (s *PluginsTestSuite) TestExtensionsPluginDeleteDeploymentNonexistent() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	OrasFactory = NewTestOras
	CatalogFactory = newTestCatalog
	AppDeploymentFactory = newTestADM

	// prepopulate mockDeployments with three deployments

	// nolint:gofmt
	mockDeployments = map[string]*mockDeployment{
		"base-extensions-0.2.0-baseline": &mockDeployment{
			name:        "base-extensions",
			version:     "0.2.0",
			profileName: "baseline",
			labels:      map[string]string{"color": "blue"},
		},
		"base-extensions-0.2.0-restricted": &mockDeployment{
			name:        "base-extensions",
			version:     "0.2.0",
			profileName: "restricted",
			labels:      map[string]string{"color": "red"},
		},
		"base-extensions-0.2.0-privileged": &mockDeployment{
			name:        "base-extensions",
			version:     "0.2.0",
			profileName: "privileged",
			labels:      map[string]string{"color": "green"},
		},
	}

	// create a manifest that deletes a deployment that doesn't exist

	manifest := `# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
---
metadata:
  schemaVersion: 0.3.0
  release: 1.2.0
lpke:
  deploymentList:
    - dpName: base-extensions
      dpProfileName: insanelyrestricted
      dpVersion: 0.2.0
      allAppTargetClusters:
        - key: color
          val: infrared
      desiredState: absent`

	configuration := config.Configuration{
		HarborServer:      "https://harbor.org",
		KeycloakNamespace: "keycloak-ns",
		KeycloakSecret:    "sekret",
		AdmServer:         "http://admserver",
		ManifestPath:      "/registry/edge-node/en/manifest",
		ManifestTag:       "latest",
		UseLocalManifest:  manifest,
	}

	plugin, err := NewExtensionsProvisionerPlugin(configuration)
	s.NoError(err, "Cannot create extensions plugin")
	s.NotNil(plugin)

	RemoveAllPlugins()
	Register(plugin)
	err = Initialize(ctx)
	assert.NoError(s.T(), err, "Initialize plugin")

	err = Dispatch(ctx, Event{
		EventType: "create",
		UUID:      "foo",
	}, nil)
	s.NoError(err)

	s.Len(mockDeployments, 3)
	baselineKey := "base-extensions-0.2.0-baseline"
	s.Equal("base-extensions", mockDeployments[baselineKey].name)
	s.Equal("0.2.0", mockDeployments[baselineKey].version)
	s.Equal("baseline", mockDeployments[baselineKey].profileName)
	s.Equal("blue", mockDeployments[baselineKey].labels["color"])

	resKey := "base-extensions-0.2.0-restricted"
	s.Equal("base-extensions", mockDeployments[resKey].name)
	s.Equal("0.2.0", mockDeployments[resKey].version)
	s.Equal("restricted", mockDeployments[resKey].profileName)
	s.Equal("red", mockDeployments[resKey].labels["color"])

	privKey := "base-extensions-0.2.0-privileged"
	s.Equal("base-extensions", mockDeployments[privKey].name)
	s.Equal("0.2.0", mockDeployments[privKey].version)
	s.Equal("privileged", mockDeployments[privKey].profileName)
	s.Equal("green", mockDeployments[privKey].labels["color"])
}

// TestExtensionsWaitForADMSucceeds tests that waitForADM succeeds when ADM is available
func (s *PluginsTestSuite) TestExtensionsWaitForADMSucceeds() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Use working ADM factory
	AppDeploymentFactory = newTestADM

	plugin, err := NewExtensionsProvisionerPlugin(config.Configuration{
		AdmServer: "localhost:8080",
	})
	s.NoError(err, "Should create extensions provisioner plugin")

	// waitForADM should succeed immediately with mock
	err = plugin.waitForADM(ctx)
	s.NoError(err, "waitForADM should succeed when ADM is available")
}

// TestExtensionsWaitForADMSkipsWhenNotConfigured tests that waitForADM skips when AdmServer is empty
func (s *PluginsTestSuite) TestExtensionsWaitForADMSkipsWhenNotConfigured() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	plugin, err := NewExtensionsProvisionerPlugin(config.Configuration{
		AdmServer: "", // Empty AdmServer
	})
	s.NoError(err, "Should create extensions provisioner plugin")

	// waitForADM should skip and return no error
	err = plugin.waitForADM(ctx)
	s.NoError(err, "waitForADM should skip when AdmServer is not configured")
}

// TestExtensionsWaitForADMFailsAfterRetries tests that waitForADM fails after max retries
func (s *PluginsTestSuite) TestExtensionsWaitForADMFailsAfterRetries() {
	// Skip this test unless explicitly requested as it takes time
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestExtensionsWaitForADMFailsAfterRetries -timeout 2m")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	// Use factory that returns error
	failingAttempts := 0
	AppDeploymentFactory = func(_ config.Configuration) (AppDeployment, error) {
		failingAttempts++
		return nil, fmt.Errorf("ADM factory failed (attempt %d)", failingAttempts)
	}

	plugin := &ExtensionsProvisionerPlugin{
		configuration: config.Configuration{
			AdmServer: "invalid-adm:9999",
		},
	}

	startTime := time.Now()
	err := plugin.waitForADM(ctx)
	duration := time.Since(startTime)

	s.Error(err, "waitForADM should fail when ADM is not available")
	s.Contains(err.Error(), "ADM not available after", "Error should indicate max retries exceeded")
	s.Greater(failingAttempts, 1, "Should attempt multiple times")
	s.Less(duration, time.Minute*2, "Should not exceed context timeout")
}

// TestExtensionsWaitForADMRecoversAfterRetries tests that waitForADM succeeds after some failures
func (s *PluginsTestSuite) TestExtensionsWaitForADMRecoversAfterRetries() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	attempts := 0
	// Create a mock ADM that fails initially then succeeds
	mockADM := &mockDynamicADM{
		listDeploymentNamesFunc: func(_ context.Context, _ string) (map[string]string, error) {
			attempts++
			if attempts < 3 {
				return nil, fmt.Errorf("ADM not ready yet (attempt %d)", attempts)
			}
			// After 3 attempts, succeed
			return map[string]string{}, nil
		},
	}

	AppDeploymentFactory = func(_ config.Configuration) (AppDeployment, error) {
		return mockADM, nil
	}

	plugin := &ExtensionsProvisionerPlugin{
		configuration: config.Configuration{
			AdmServer: "localhost:8080",
		},
	}

	startTime := time.Now()
	err := plugin.waitForADM(ctx)
	duration := time.Since(startTime)

	s.NoError(err, "waitForADM should succeed after recovering")
	s.GreaterOrEqual(attempts, 3, "Should attempt at least 3 times before succeeding")
	s.Greater(duration, time.Second*10, "Should take time due to retries with backoff")
	s.T().Logf("ADM recovered after %d attempts in %v", attempts, duration)
} // TestExtensionsInitializeFailsWhenADMFails tests that Initialize propagates ADM errors
func (s *PluginsTestSuite) TestExtensionsInitializeFailsWhenADMFails() {
	// Skip this test unless explicitly requested as it takes time
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestExtensionsInitializeFailsWhenADMFails -timeout 2m")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	// Use factory that always fails
	AppDeploymentFactory = func(_ config.Configuration) (AppDeployment, error) {
		return nil, fmt.Errorf("ADM connection permanently failed")
	}

	plugin := &ExtensionsProvisionerPlugin{
		configuration: config.Configuration{
			AdmServer: "invalid-adm:9999",
		},
	}

	err := plugin.Initialize(ctx, nil)
	s.Error(err, "Initialize should fail when ADM fails")
	s.Contains(err.Error(), "extensions initialization failed during ADM check", "Error should indicate ADM check failure")
}

// TestExtensionsInitializeSucceedsWhenADMNotConfigured tests that Initialize succeeds when ADM is not configured
func (s *PluginsTestSuite) TestExtensionsInitializeSucceedsWhenADMNotConfigured() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	plugin := &ExtensionsProvisionerPlugin{
		configuration: config.Configuration{
			AdmServer: "", // No ADM configured
		},
	}

	err := plugin.Initialize(ctx, nil)
	s.NoError(err, "Initialize should succeed when ADM is not configured")
}
