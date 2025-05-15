// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type MockProjectManager struct {
	deleted []string
	created []string
}

func (m *MockProjectManager) CreateProject(orgName string, projectName string, projectUUID string, project NexusProjectInterface) {
	_ = orgName
	_ = projectUUID
	_ = project
	m.created = append(m.created, projectName)
}

func (m *MockProjectManager) DeleteProject(orgName string, projectName string, projectUUID string, project NexusProjectInterface) {
	_ = orgName
	_ = projectUUID
	_ = project
	m.deleted = append(m.deleted, projectName)
}

func (m *MockProjectManager) ManifestTag() string {
	return ""
}

type NexusHookTestSuite struct {
	suite.Suite
}

func (s *NexusHookTestSuite) SetupSuite() {
}

func (s *NexusHookTestSuite) TearDownSuite() {
}

func (s *NexusHookTestSuite) SetupTest() {
}

func (s *NexusHookTestSuite) TearDownTest() {
}

func (s *NexusHookTestSuite) TestProjectCreated() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project")

	s.Contains(m.created, "project1", "Expected project1 to be in the created list")

	s.Equal(1, len(project.activeWatchers), "Expected 1 active watcher")
	s.Contains(project.activeWatchers, "config-provisioner", "Expected 'config-provisioner' to be a key in the activeWatchers map")
	s.Equal(projectActiveWatcherv1.StatusIndicationInProgress, project.activeWatchers["config-provisioner"].Spec.StatusIndicator, "Expected status to be 'InProgress'")
}

func (s *NexusHookTestSuite) TestProjectDeleted() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	project.isDeleted = true
	h.projectUpdated(project)

	s.Contains(m.deleted, "project1", "Expected project1 to be in the created list")

	s.Equal(0, len(project.activeWatchers), "Expected 0 active watcher")
}

func (s *NexusHookTestSuite) TestSetWatcherStatusError() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project")

	err = h.SetWatcherStatusError(project, "some error")
	s.NoError(err, "Expected no error when setting watcher status to error")

	s.Contains(project.activeWatchers, "config-provisioner", "Expected 'config-provisioner' to be a key in the activeWatchers map")
	s.Equal(projectActiveWatcherv1.StatusIndicationError, project.activeWatchers["config-provisioner"].Spec.StatusIndicator, "Expected status to be 'Error'")
	s.Equal("some error", project.activeWatchers["config-provisioner"].Spec.Message, "Expected status to be 'some error'")
}

func (s *NexusHookTestSuite) TestSetWatcherStatusInProgress() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project")

	err = h.SetWatcherStatusInProgress(project, "making progress")
	s.NoError(err, "Expected no error when setting watcher status to in progress")

	s.Contains(project.activeWatchers, "config-provisioner", "Expected 'config-provisioner' to be a key in the activeWatchers map")
	s.Equal(projectActiveWatcherv1.StatusIndicationInProgress, project.activeWatchers["config-provisioner"].Spec.StatusIndicator, "Expected status to be 'In Progress'")
	s.Equal("making progress", project.activeWatchers["config-provisioner"].Spec.Message, "Expected status to be 'making progress'")
}

func (s *NexusHookTestSuite) TestSetWatcherStatusIdle() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	err := h.projectCreated(project)
	s.NoError(err, "Expected no error when creating project")

	err = h.SetWatcherStatusIdle(project)
	s.NoError(err, "Expected no error when setting watcher status to idle")

	s.Contains(project.activeWatchers, "config-provisioner", "Expected 'config-provisioner' to be a key in the activeWatchers map")
	s.Equal(projectActiveWatcherv1.StatusIndicationIdle, project.activeWatchers["config-provisioner"].Spec.StatusIndicator, "Expected status to be 'Idle'")
}

func TestNexusHook(t *testing.T) {
	suite.Run(t, &NexusHookTestSuite{})
}

func FuzzCreateProject(f *testing.F) {
	f.Add("Test Project", "project1")
	f.Add(" space at start", "project1")
	f.Add("space at end ", "project1")
	f.Add("starts with hyphen", "-")
	f.Add("Single letter OK", "a")
	f.Add("contains .", "a.")
	f.Add("ID is long > 40", "aaaaa-bbbb-cccc-dddd-eeee-ffff-gggg-hhhhh")
	f.Add("display name is kinda long ----------------------- here", "project1")
	f.Add(strings.Repeat("display name is very long", 10), "project1")
	f.Add(`display name contains
new line`, "project1")

	s := &NexusHookTestSuite{}
	s.SetupSuite()
	defer s.TearDownSuite()

	f.Fuzz(func(t *testing.T, uid string, displayName string) {
		s.SetT(t)
		s.SetupTest() // SetupTest cannot be called until here because it depends on T's Assertions
		defer s.TearDownTest()

		m := &MockProjectManager{}
		h := NewNexusHook(m)

		project := NewMockNexusProject(displayName, uid)
		err := h.projectCreated(project)

		if err != nil {
			allowedErr := []string{"Organization name is empty", "Organization name is too long", "Project name is empty", "Project name is too long", "Project UUID is empty", "Project UUID is too long", "Organization name contains illegal characters", "Project name contains illegal characters"}
			assert.Contains(t, allowedErr, err.Error(), "Expected error to be one of the allowed errors")
		} else {
			assert.Contains(t, m.created, displayName, "Expected project to be in the created list")

			s.Equal(1, len(project.activeWatchers), "Expected 1 active watcher")
			s.Contains(project.activeWatchers, "config-provisioner", "Expected 'config-provisioner' to be a key in the activeWatchers map")
			s.Equal(projectActiveWatcherv1.StatusIndicationInProgress, project.activeWatchers["config-provisioner"].Spec.StatusIndicator, "Expected status to be 'InProgress'")
		}
	})
}

func FuzzDeleteProject(f *testing.F) {
	f.Add("Test Project", "project1")
	f.Add(" space at start", "project1")
	f.Add("space at end ", "project1")
	f.Add("starts with hyphen", "-")
	f.Add("Single letter OK", "a")
	f.Add("contains .", "a.")
	f.Add("ID is long > 40", "aaaaa-bbbb-cccc-dddd-eeee-ffff-gggg-hhhhh")
	f.Add("display name is kinda long ----------------------- here", "project1")
	f.Add(strings.Repeat("display name is very long", 10), "project1")
	f.Add(`display name contains
new line`, "project1")

	s := &NexusHookTestSuite{}
	s.SetupSuite()
	defer s.TearDownSuite()

	f.Fuzz(func(t *testing.T, uid string, displayName string) {
		s.SetT(t)
		s.SetupTest() // SetupTest cannot be called until here because it depends on T's Assertions
		defer s.TearDownTest()

		m := &MockProjectManager{}
		h := NewNexusHook(m)

		project := NewMockNexusProject(displayName, uid)
		project.isDeleted = true
		h.projectUpdated(project)

		assert.Contains(t, m.deleted, displayName, "Expected project to be in the created list")

		s.Equal(0, len(project.activeWatchers), "Expected 0 active watcher")
	})
}
