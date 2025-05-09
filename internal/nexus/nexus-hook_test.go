// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type MockProjectManager struct {
	deleted []string
	created []string
}

func (m *MockProjectManager) CreateProject(orgName string, projectName string, projectUUID string, project NexusProjectInterface) {
	m.created = append(m.created, projectName)
}

func (m *MockProjectManager) DeleteProject(orgName string, projectName string, projectUUID string, project NexusProjectInterface) {
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

	project := NewMockNexusProject("project1", "uid1", nil)
	h.projectCreated(project)

	s.Contains(m.created, "project1", "Expected project1 to be in the created list")
}

func TestNexusHook(t *testing.T) {
	suite.Run(t, &NexusHookTestSuite{})
}