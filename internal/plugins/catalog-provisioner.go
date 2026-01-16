// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Internal package
package plugins

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/southbound"
	"strings"
	"time"
)

const (
	HarborTokenName    = `harborToken`
	HarborUsernameName = `harborUsername`
)

type Catalog interface {
	CreateOrUpdateRegistry(ctx context.Context, attrs southbound.RegistryAttributes) error
	ListRegistries(ctx context.Context) error
	UploadYAMLFile(ctx context.Context, projectUUID string, fileName string, artifact []byte, lastFile bool) error
	InitializeClientSecret(ctx context.Context) (string, error)
	WipeProject(ctx context.Context, projectUUID string, catalogServer string) error
}

type CatalogProvisionerPlugin struct {
	config config.Configuration
}

func NewCatalog(config config.Configuration) (Catalog, error) {
	return southbound.NewAppCatalog(config)
}

var CatalogFactory = NewCatalog

func NewCatalogProvisionerPlugin(config config.Configuration) (*CatalogProvisionerPlugin, error) {
	return &CatalogProvisionerPlugin{
		config: config,
	}, nil

}

func (p *CatalogProvisionerPlugin) waitForCatalog(ctx context.Context) error {
	log.Info("Waiting for catalog")
	catalog, err := CatalogFactory(p.config)
	if err != nil {
		return fmt.Errorf("failed to create catalog client: %w", err)
	}

	maxRetries := 12 // Total wait time: ~5 minutes
	retryDelay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := catalog.ListRegistries(lctx)
		cancel()

		if err == nil || strings.Contains(err.Error(), "Unauthenticated") {
			log.Info("Catalog ready")
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("catalog not available after %d attempts: %w", maxRetries, err)
		}

		log.Infof("Catalog ping failed (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)

		retryDelay = retryDelay * 2
		if retryDelay > 60*time.Second {
			retryDelay = 60 * time.Second
		}
	}

	return fmt.Errorf("catalog not available after maximum retries")
}

func (p *CatalogProvisionerPlugin) waitForVault(ctx context.Context) error {
	log.Info("Waiting for vault")
	catalog, err := CatalogFactory(p.config)
	if err != nil {
		return fmt.Errorf("failed to create catalog client for vault: %w", err)
	}

	maxRetries := 12 // Total wait time: ~5 minutes
	retryDelay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := catalog.InitializeClientSecret(lctx)
		cancel()

		if err == nil {
			log.Info("Vault ready")
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("vault not available after %d attempts: %w", maxRetries, err)
		}

		log.Infof("Vault login failed (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)

		retryDelay = retryDelay * 2
		if retryDelay > 60*time.Second {
			retryDelay = 60 * time.Second
		}
	}

	return fmt.Errorf("vault not available after maximum retries")
}

func (p *CatalogProvisionerPlugin) Initialize(ctx context.Context, _ PluginData) error {
	var err error
	var catalog Catalog
	catalog, err = CatalogFactory(p.config)
	if err != nil {
		return fmt.Errorf("catalog initialization failed: %w", err)
	}

	if err := p.waitForVault(ctx); err != nil {
		return fmt.Errorf("catalog initialization failed during vault check: %w", err)
	}

	_, err = catalog.InitializeClientSecret(ctx)
	if err != nil {
		return fmt.Errorf("catalog initialization failed to initialize client secret: %w", err)
	}

	if err := p.waitForCatalog(ctx); err != nil {
		return fmt.Errorf("catalog initialization failed during catalog check: %w", err)
	}

	log.Info("Completed Initializing Catalog plugin")
	return nil
}

func (p *CatalogProvisionerPlugin) CreateEvent(ctx context.Context, event Event, pluginData PluginData) error {
	var err error
	var catalog Catalog
	catalog, err = CatalogFactory(p.config)
	if err != nil {
		return err
	}

	rsReposDescription := fmt.Sprintf("Repo on registry %s", strings.ReplaceAll(p.config.ReleaseServiceRootURL, "oci://", ""))
	rsHelmRegistryAttrs := southbound.RegistryAttributes{
		Name:        `intel-rs-helm`,
		DisplayName: `intel-rs-helm`,
		Description: rsReposDescription,
		Type:        `HELM`,
		ProjectUUID: event.UUID,
		RootURL:     p.config.ReleaseServiceProxyRootURL,
	}
	err = catalog.CreateOrUpdateRegistry(ctx, rsHelmRegistryAttrs)
	if err != nil {
		log.Errorf("Error creating registry intel-rs-helm: %v", err)
		return err
	}

	rsDockerRegistryAttrs := southbound.RegistryAttributes{
		Name:        `intel-rs-images`,
		DisplayName: `intel-rs-image`,
		Description: rsReposDescription,
		Type:        `IMAGE`,
		ProjectUUID: event.UUID,
		RootURL:     p.config.ReleaseServiceRootURL,
	}
	err = catalog.CreateOrUpdateRegistry(ctx, rsDockerRegistryAttrs)
	if err != nil {
		log.Errorf("Error creating registry intel-rs-images: %v", err)
		return err
	}

	var (
		token    string
		username string
		cacerts  string
	)

	if v, ok := (*pluginData)[HarborTokenName]; ok {
		token = v
	}
	if v, ok := (*pluginData)[HarborUsernameName]; ok {
		username = v
	}

	cacerts = "use-dynamic-cacert"

	ociRegistry := strings.ReplaceAll(p.config.HarborServerExternal, "https://", "oci://")

	harborProjectName := southbound.HarborProjectName(event.Organization, event.Name)
	OCIHelmRegistryAttrs := southbound.RegistryAttributes{
		Name:         `harbor-helm-oci`,
		DisplayName:  `harbor oci helm`,
		Description:  `Harbor OCI helm charts registry`,
		Type:         `HELM`,
		ProjectUUID:  event.UUID,
		RootURL:      ociRegistry + "/" + harborProjectName,
		InventoryURL: p.config.HarborServerExternal + `/api/v2.0/projects/` + harborProjectName,
		Username:     username,
		Cacerts:      cacerts,
		AuthToken:    token,
	}
	err = catalog.CreateOrUpdateRegistry(ctx, OCIHelmRegistryAttrs)
	if err != nil {
		log.Errorf("Error creating registry harbor-helm-oci: %v", err)
		return err
	}

	OCIimageRegistryAttrs := southbound.RegistryAttributes{
		Name:        `harbor-docker-oci`,
		DisplayName: `harbor oci docker`,
		Description: `Harbor OCI docker images registry`,
		Type:        "IMAGE",
		ProjectUUID: event.UUID,
		RootURL:     ociRegistry + "/" + strings.ToLower(harborProjectName),
		Username:    username,
		Cacerts:     cacerts,
		AuthToken:   token,
	}
	err = catalog.CreateOrUpdateRegistry(ctx, OCIimageRegistryAttrs)
	if err != nil {
		log.Errorf("Error creating registry harbor-docker-oci: %v", err)
		return err
	}

	return nil
}

func (p *CatalogProvisionerPlugin) DeleteEvent(ctx context.Context, event Event, _ PluginData) error {
	catalog, err := CatalogFactory(p.config)
	if err != nil {
		return err
	}
	return catalog.WipeProject(ctx, event.UUID, p.config.CatalogServer)
}

func (p *CatalogProvisionerPlugin) Name() string {
	return "Catalog Provisioner"
}
