// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/stretchr/testify/assert"
)

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
