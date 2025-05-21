// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"time"
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
