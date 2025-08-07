// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"testing"

	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	"github.com/stretchr/testify/suite"
)

type UpgradeScenarioTestSuite struct {
	suite.Suite
}

func (s *UpgradeScenarioTestSuite) TestUpgradeScenarioWithoutManifestTag() {
	// Test simulates upgrade scenario where existing watcher lacks manifest-tag
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("test-project-upgrade", "test-uid-123")
	
	// First, create the project normally (simulating pre-upgrade state)
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project initially")
	
	// Now simulate upgrade by removing manifest-tag from the watcher 
	// (this simulates pre-upgrade watcher without manifest-tag)
	watcher := project.activeWatchers["config-provisioner"]
	s.NotNil(watcher, "Expected active watcher to exist")
	
	// Remove the manifest-tag to simulate pre-upgrade state
	annotations := watcher.GetAnnotations()
	delete(annotations, ManifestTagAnnotationKey)
	
	// Now call projectCreated again which should handle the upgrade scenario
	err = h.projectCreated(project) 
	s.NoError(err, "Expected no error when handling upgrade scenario")
}

func (s *UpgradeScenarioTestSuite) TestUpgradeScenarioWithIncorrectManifestTag() {
	// Test simulates upgrade scenario where existing watcher has wrong manifest-tag
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("test-project-upgrade", "test-uid-456")
	
	// First, create the project normally
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project initially")
	
	// Now simulate upgrade by changing manifest-tag to an old value
	watcher := project.activeWatchers["config-provisioner"]
	s.NotNil(watcher, "Expected active watcher to exist")
	
	// Set an old manifest-tag to simulate upgrade from older version
	annotations := watcher.GetAnnotations()
	annotations[ManifestTagAnnotationKey] = "v1.2.0" // Old tag, different from current v1.3.5
	
	// Now call projectCreated again which should handle the upgrade scenario
	err = h.projectCreated(project) 
	s.NoError(err, "Expected no error when handling upgrade scenario with incorrect manifest tag")
}

func (s *UpgradeScenarioTestSuite) TestCorrectManifestTagNoUpdate() {
	// Test simulates scenario where manifest tag is already correct (no upgrade needed)
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("correct-tag-project", "correct-uid-789")
	
	// First, create the project normally
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project initially")
	
	// Simulate the project creation completing successfully by updating watcher status
	watcher := project.activeWatchers["config-provisioner"]
	s.NotNil(watcher, "Expected active watcher to exist")
	
	// Simulate successful completion - this would normally be done by the ProjectManager
	annotations := watcher.GetAnnotations()
	annotations[ManifestTagAnnotationKey] = "v1.3.5" // Set correct manifest tag
	watcher.GetSpec().StatusIndicator = projectActiveWatcherv1.StatusIndicationIdle
	watcher.GetSpec().Message = "Created"
	
	initialCreatedCount := len(m.created)
	
	// Now call projectCreated again - since manifest tag is already correct, no update should occur
	err = h.projectCreated(project)
	s.NoError(err, "Expected no error when manifest tag is already correct")
	
	// Verify that the manifest tag remains correct
	annotations = watcher.GetAnnotations()
	s.Contains(annotations, ManifestTagAnnotationKey, "Expected manifest tag to be present")
	s.Equal("v1.3.5", annotations[ManifestTagAnnotationKey], "Expected manifest tag to remain unchanged")
	
	// Verify that CreateProject was NOT called again since no update was needed
	s.Equal(initialCreatedCount, len(m.created), "Expected no additional project creation when manifest tag is correct")
}

func (s *UpgradeScenarioTestSuite) TestSameManifestTagNoProcessing() {
	// Test simulates scenario where manifest tag hasn't changed during upgrade
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("same-tag-project", "same-tag-uid-999")
	
	// First, create the project normally
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project initially")
	
	// Simulate successful completion with correct manifest tag
	watcher := project.activeWatchers["config-provisioner"]
	s.NotNil(watcher, "Expected active watcher to exist")
	
	annotations := watcher.GetAnnotations()
	annotations[ManifestTagAnnotationKey] = "v1.3.5" // Same as MockProjectManager returns
	watcher.GetSpec().StatusIndicator = projectActiveWatcherv1.StatusIndicationIdle
	watcher.GetSpec().Message = "Created"
	
	initialCreatedCount := len(m.created)
	
	// Simulate "upgrade" where manifest tag hasn't actually changed
	// This should result in early exit with no processing
	err = h.projectCreated(project)
	s.NoError(err, "Expected no error when manifest tag is unchanged")
	
	// Verify that NO additional processing occurred
	s.Equal(initialCreatedCount, len(m.created), "Expected no project creation when manifest tag unchanged")
}

func TestUpgradeScenario(t *testing.T) {
	suite.Run(t, &UpgradeScenarioTestSuite{})
}
