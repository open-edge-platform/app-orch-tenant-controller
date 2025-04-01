// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/stretchr/testify/assert"
	"time"
)

func (s *PluginsTestSuite) TestExtensionsPluginCreate() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	OrasFactory = NewTestOras
	CatalogFactory = newTestCatalog
	AppDeploymentFactory = newTestADM

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
