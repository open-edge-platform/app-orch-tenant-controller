// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"github.com/stretchr/testify/assert"
	"time"
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
