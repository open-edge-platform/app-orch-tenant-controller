// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/southbound"
	"github.com/stretchr/testify/assert"
)

func (s *PluginsTestSuite) TestHarborPluginCreate() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	HarborFactory = NewTestHarbor
	CatalogFactory = newTestCatalog

	plugin, err := NewHarborProvisionerPlugin(ctx, "", "", "harbor", "credential")
	s.NoError(err, "Cannot create harbor provisioner plugin")
	s.NotNil(plugin)

	Register(plugin)
	err = Initialize(ctx)
	assert.NoError(s.T(), err, "Initialize harbor plugin")

	err = Dispatch(ctx, Event{
		EventType:    "create",
		Name:         "fOo",
		Organization: "xYzzY",
	}, nil)
	s.NoError(err)

	s.Len(testHarborInstance.createdProjects, 1)
	createdProject := testHarborInstance.createdProjects[`xyzzy-foo`]
	s.Equal(`xyzzy-foo`, createdProject)

	expectedRobotName := `robot$catalog-apps-xyzzy-foo+catalog-apps-read-write`
	s.Len(testHarborInstance.robots, 1)
	r := testHarborInstance.robots[expectedRobotName]
	s.Equal(expectedRobotName, r.robotName)
	s.Equal(1, r.robotID)

	err = Dispatch(ctx, Event{
		EventType:    "create",
		Name:         "fOo",
		Organization: "xYzzY",
	}, nil)
	s.NoError(err)
	s.Len(testHarborInstance.robots, 1)
	r2 := testHarborInstance.robots[expectedRobotName]
	s.Equal(expectedRobotName, r2.robotName)
	s.Equal(2, r2.robotID)

	// Now delete the project
	err = Dispatch(ctx, Event{
		EventType:    "delete",
		Name:         "fOo",
		Organization: "xYzzY",
	}, nil)
	s.NoError(err)
	s.Len(testHarborInstance.createdProjects, 0)
}

// Mock Harbor that fails Ping operations for testing failure scenarios
type failingHarborPing struct {
	pingCallCount           int
	configurationsCallCount int
	failPingUntilAttempt    int // Succeed after this many attempts (0 = always fail)
}

func (t *failingHarborPing) Ping(_ context.Context) error {
	t.pingCallCount++
	if t.failPingUntilAttempt == 0 || t.pingCallCount <= t.failPingUntilAttempt {
		return errors.New("harbor ping failed: connection refused")
	}
	return nil
}

func (t *failingHarborPing) Configurations(_ context.Context) error {
	t.configurationsCallCount++
	return nil
}

func (t *failingHarborPing) CreateProject(_ context.Context, _ string, _ string) error {
	return nil
}

func (t *failingHarborPing) SetMemberPermissions(_ context.Context, _ int, _ string, _ string, _ string) error {
	return nil
}

func (t *failingHarborPing) GetProjectID(_ context.Context, _ string, _ string) (int, error) {
	return HarborProjectID, nil
}

func (t *failingHarborPing) CreateRobot(_ context.Context, _ string, _ string, _ string) (string, string, error) {
	return "name", "secret", nil
}

func (t *failingHarborPing) GetRobot(_ context.Context, _ string, _ string, _ string, _ int) (*southbound.HarborRobot, error) {
	return nil, errors.New("robot not found")
}

func (t *failingHarborPing) DeleteRobot(_ context.Context, _ int) error {
	return nil
}

func (t *failingHarborPing) DeleteProject(_ context.Context, _ string, _ string) error {
	return nil
}

// Mock Harbor that fails Configuration operations for testing failure scenarios
type failingHarborConfig struct {
	pingCallCount                  int
	configurationsCallCount        int
	failConfigurationsUntilAttempt int // Succeed after this many attempts (0 = always fail)
}

func (t *failingHarborConfig) Ping(_ context.Context) error {
	t.pingCallCount++
	return nil
}

func (t *failingHarborConfig) Configurations(_ context.Context) error {
	t.configurationsCallCount++
	if t.failConfigurationsUntilAttempt == 0 || t.configurationsCallCount <= t.failConfigurationsUntilAttempt {
		return fmt.Errorf(`{"errors":[{"code":"UNKNOWN","message":"internal server error"}]}`)
	}
	return nil
}

func (t *failingHarborConfig) CreateProject(_ context.Context, _ string, _ string) error {
	return nil
}

func (t *failingHarborConfig) SetMemberPermissions(_ context.Context, _ int, _ string, _ string, _ string) error {
	return nil
}

func (t *failingHarborConfig) GetProjectID(_ context.Context, _ string, _ string) (int, error) {
	return HarborProjectID, nil
}

func (t *failingHarborConfig) CreateRobot(_ context.Context, _ string, _ string, _ string) (string, string, error) {
	return "name", "secret", nil
}

func (t *failingHarborConfig) GetRobot(_ context.Context, _ string, _ string, _ string, _ int) (*southbound.HarborRobot, error) {
	return nil, errors.New("robot not found")
}

func (t *failingHarborConfig) DeleteRobot(_ context.Context, _ int) error {
	return nil
}

func (t *failingHarborConfig) DeleteProject(_ context.Context, _ string, _ string) error {
	return nil
}

// Test: Harbor Ping fails permanently - should return error after max retries
// Note: This test is SKIPPED by default as it takes ~5 minutes due to realistic exponential backoff
// To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestHarborPingFailsPermanently -timeout 10m
func (s *PluginsTestSuite) TestHarborPingFailsPermanently() {
	// Skip this test unless RUN_LONG_TESTS environment variable is set
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestHarborPingFailsPermanently -timeout 10m")
	}

	// Use a longer timeout since retries take time with exponential backoff
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*6)
	defer cancel()

	mockHarbor := &failingHarborPing{
		failPingUntilAttempt: 0, // Always fail
	}

	HarborFactory = func(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
		return mockHarbor, nil
	}

	plugin, err := NewHarborProvisionerPlugin(ctx, "http://harbor", "http://keycloak", "harbor", "credential")
	s.NoError(err, "Plugin creation should succeed")
	s.NotNil(plugin)

	// Initialize should fail after max retries
	start := time.Now()
	err = plugin.Initialize(ctx, nil)
	elapsed := time.Since(start)

	s.Error(err, "Initialize should fail when Harbor ping fails permanently")
	s.Contains(err.Error(), "harbor not available after", "Error should mention retry exhaustion")

	// Verify it attempted multiple retries (12 attempts)
	s.Equal(12, mockHarbor.pingCallCount, "Should attempt exactly 12 retries")

	// Verify Configurations was never called
	s.Equal(0, mockHarbor.configurationsCallCount, "Configurations should not be called if ping fails")

	s.T().Logf("Total elapsed time: %v", elapsed)
	s.GreaterOrEqual(elapsed, 4*time.Minute, "Should respect exponential backoff timing")
}

// Test: Harbor Ping recovers after a few retries
func (s *PluginsTestSuite) TestHarborPingRecoversAfterRetries() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	mockHarbor := &failingHarborPing{
		failPingUntilAttempt: 3,
	}

	HarborFactory = func(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
		return mockHarbor, nil
	}

	plugin, err := NewHarborProvisionerPlugin(ctx, "http://harbor", "http://keycloak", "harbor", "credential")
	s.NoError(err, "Plugin creation should succeed")
	s.NotNil(plugin)

	// Initialize should succeed after retries
	err = plugin.Initialize(ctx, nil)
	s.NoError(err, "Initialize should succeed when Harbor recovers")

	s.Equal(4, mockHarbor.pingCallCount, "Should have made 4 ping attempts (3 failures + 1 success)")

	// Verify Configurations was called once
	s.Equal(1, mockHarbor.configurationsCallCount, "Configurations should be called once after ping succeeds")
}

// Test: Harbor Configuration fails permanently
func (s *PluginsTestSuite) TestHarborConfigurationFailsPermanently() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	mockHarbor := &failingHarborConfig{
		failConfigurationsUntilAttempt: 0, // Always fail
	}

	HarborFactory = func(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
		return mockHarbor, nil
	}

	plugin, err := NewHarborProvisionerPlugin(ctx, "http://harbor", "http://keycloak", "harbor", "credential")
	s.NoError(err, "Plugin creation should succeed")
	s.NotNil(plugin)

	// Initialize should fail after max retries
	err = plugin.Initialize(ctx, nil)
	s.Error(err, "Initialize should fail when Harbor configuration fails permanently")
	s.Contains(err.Error(), "failed to apply harbor configuration", "Error should mention configuration failure")

	// Verify ping succeeded
	s.GreaterOrEqual(mockHarbor.pingCallCount, 1, "Ping should have been attempted")

	// Verify it attempted multiple configuration retries (3 attempts)
	s.Equal(3, mockHarbor.configurationsCallCount, "Should attempt 3 configuration retries")
}

// Test: Harbor Configuration recovers after retries - should succeed
func (s *PluginsTestSuite) TestHarborConfigurationRecoversAfterRetries() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	mockHarbor := &failingHarborConfig{
		failConfigurationsUntilAttempt: 2, // Fail first 2 attempts, succeed on 3rd
	}

	HarborFactory = func(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
		return mockHarbor, nil
	}

	plugin, err := NewHarborProvisionerPlugin(ctx, "http://harbor", "http://keycloak", "harbor", "credential")
	s.NoError(err, "Plugin creation should succeed")
	s.NotNil(plugin)

	// Initialize should succeed after retries
	err = plugin.Initialize(ctx, nil)
	s.NoError(err, "Initialize should succeed when Harbor configuration recovers")

	// Verify ping succeeded
	s.GreaterOrEqual(mockHarbor.pingCallCount, 1, "Ping should have been attempted")

	s.Equal(3, mockHarbor.configurationsCallCount, "Should have made 3 configuration attempts (2 failures + 1 success)")
}

// Test: Harbor factory fails - should return error immediately
func (s *PluginsTestSuite) TestHarborFactoryFails() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	HarborFactory = func(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
		return nil, errors.New("failed to read harbor credentials")
	}

	plugin, err := NewHarborProvisionerPlugin(ctx, "http://harbor", "http://keycloak", "harbor", "credential")
	s.Error(err, "Plugin creation should fail when factory fails")
	s.Nil(plugin)
	s.Contains(err.Error(), "failed to read harbor credentials", "Error should propagate from factory")
}

// Test: Verify exponential backoff timing
// Note: This test is SKIPPED by default as it takes ~2.5 minutes
// To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestHarborPingExponentialBackoff -timeout 5m
func (s *PluginsTestSuite) TestHarborPingExponentialBackoff() {
	// Skip this test unless RUN_LONG_TESTS environment variable is set
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		s.T().Skip("Skipping long-running test. To run: RUN_LONG_TESTS=1 go test -v -run TestPlugins/TestHarborPingExponentialBackoff -timeout 5m")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
	defer cancel()

	mockHarbor := &failingHarborPing{
		failPingUntilAttempt: 5, // Fail first 5 attempts
	}

	HarborFactory = func(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
		return mockHarbor, nil
	}

	plugin, err := NewHarborProvisionerPlugin(ctx, "http://harbor", "http://keycloak", "harbor", "credential")
	s.NoError(err, "Plugin creation should succeed")

	start := time.Now()
	err = plugin.Initialize(ctx, nil)
	elapsed := time.Since(start)

	s.NoError(err, "Initialize should succeed after retries")

	// With exponential backoff: 5s, 10s, 20s, 40s, 60s(capped) = ~135s for 5 attempts
	// Allow some tolerance for test execution time
	s.T().Logf("Total elapsed time: %v", elapsed)
	s.GreaterOrEqual(elapsed, 2*time.Minute, "Should have waited for exponential backoff delays")
	s.LessOrEqual(elapsed, 170*time.Second, "Should not have waited excessively long")

	s.Equal(6, mockHarbor.pingCallCount, "Should have made 6 ping attempts (5 failures + 1 success)")
}
