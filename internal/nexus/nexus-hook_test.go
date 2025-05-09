// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

func (s *NexusHookTestSuite) TestProjectUpdated() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	h.projectCreated(project)

	s.Contains(m.created, "project1", "Expected project1 to be in the created list")
}

func (s *NexusHookTestSuite) TestProjectDeleted() {
	m := &MockProjectManager{}
	h := NewNexusHook(m)

	project := NewMockNexusProject("project1", "uid1")
	project.isDeleted = true
	h.projectUpdated(project)

	s.Contains(m.deleted, "project1", "Expected project1 to be in the created list")
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
	f.Add("display name is too long at 40 chars - here", "project1")
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
		h.projectCreated(project)

		assert.Contains(t, m.created, displayName, "Expected project to be in the created list")
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
	f.Add("display name is too long at 40 chars - here", "project1")
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
	})
}
