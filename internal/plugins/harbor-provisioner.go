// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/southbound"
)

type Harbor interface {
	Configurations(ctx context.Context) error
	CreateProject(ctx context.Context, org string, displayName string) error
	SetMemberPermissions(ctx context.Context, roleID int, org string, displayName string, groupName string) error
	CreateRobot(ctx context.Context, robotName string, org string, displayName string) (string, string, error)
	GetProjectID(ctx context.Context, org string, displayName string) (int, error)
	GetRobot(ctx context.Context, org string, displayName string, robotName string, projectID int) (*southbound.HarborRobot, error)
	DeleteRobot(ctx context.Context, robotID int) error
	DeleteProject(ctx context.Context, org string, displayName string) error
	Ping(ctx context.Context) error
}

type HarborProvisionerPlugin struct {
	harbor                Harbor
	harborHost            string
	harborNamespace       string
	harborAdminCredential string
	oidcURL               string
}

func NewHarbor(ctx context.Context, harborHost string, oidcURL string, harborNamespace string, harborAdminCredential string) (Harbor, error) {
	return southbound.NewHarborOCI(ctx, harborHost, oidcURL, harborNamespace, harborAdminCredential)
}

var HarborFactory = NewHarbor

func (p *HarborProvisionerPlugin) waitForHarbor(ctx context.Context) error {
	log.Info("Waiting for Harbor")
	harbor, err := HarborFactory(ctx, p.harborHost, p.oidcURL, p.harborNamespace, p.harborAdminCredential)
	if err != nil {
		return fmt.Errorf("failed to create Harbor client: %w", err)
	}

	maxRetries := 12 // Total wait time: ~5 minutes with exponential backoff
	retryDelay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := harbor.Ping(lctx)
		cancel()

		if err == nil {
			log.Info("Harbor ready")
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("harbor not available after %d attempts: %w", maxRetries, err)
		}

		log.Infof("Harbor ping failed (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)

		// Exponential backoff with cap at 60 seconds
		retryDelay = retryDelay * 2
		if retryDelay > 60*time.Second {
			retryDelay = 60 * time.Second
		}
	}

	return fmt.Errorf("harbor not available after maximum retries")
}

func harborGroupName(event Event, kind string) string {
	return fmt.Sprintf("%s_Edge-%s-Group", event.UUID, kind)
}

func NewHarborProvisionerPlugin(ctx context.Context, harborHost string, oidcURL string, harborNamespace string, harborAdminCredential string) (*HarborProvisionerPlugin, error) {
	harbor, err := HarborFactory(ctx, harborHost, oidcURL, harborNamespace, harborAdminCredential)
	if err != nil {
		return nil, err
	}
	plugin := &HarborProvisionerPlugin{
		harbor:                harbor,
		harborHost:            harborHost,
		oidcURL:               oidcURL,
		harborNamespace:       harborNamespace,
		harborAdminCredential: harborAdminCredential,
	}
	return plugin, nil
}

func (p *HarborProvisionerPlugin) Initialize(ctx context.Context, _ PluginData) error {
	if err := p.waitForHarbor(ctx); err != nil {
		return fmt.Errorf("harbor initialization failed during ping check: %w", err)
	}

	// Retry configuration
	maxRetries := 3
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := p.harbor.Configurations(ctx)
		if err == nil {
			log.Info("Harbor configuration applied successfully")
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("failed to apply harbor configuration after %d attempts: %w", maxRetries, err)
		}

		log.Infof("Harbor configuration failed (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)
		retryDelay = retryDelay * 2
	}

	return fmt.Errorf("harbor configuration failed after maximum retries")
}

func (p *HarborProvisionerPlugin) CreateEvent(ctx context.Context, event Event, pluginData PluginData) error {
	org := strings.ToLower(event.Organization)
	name := strings.ToLower(event.Name)
	err := p.harbor.CreateProject(ctx, org, name)
	if err != nil {
		return err
	}

	operatorGroupName := harborGroupName(event, "Operator")

	err = p.harbor.SetMemberPermissions(ctx, 3, org, name, operatorGroupName)
	if err != nil {
		return err
	}

	managerGroupName := harborGroupName(event, "Manager")

	err = p.harbor.SetMemberPermissions(ctx, 4, org, name, managerGroupName)
	if err != nil {
		return err
	}

	projectID, err := p.harbor.GetProjectID(ctx, org, name)
	if err != nil {
		return err
	}

	robot, _ := p.harbor.GetRobot(ctx, org, name, "catalog-apps-read-write", projectID)
	if robot != nil {
		err = p.harbor.DeleteRobot(ctx, robot.ID)
		if err != nil {
			return err
		}
	}

	name, secret, err := p.harbor.CreateRobot(ctx, `catalog-apps-read-write`, org, name)
	if err != nil {
		return err
	}

	(*pluginData)[HarborUsernameName] = name
	(*pluginData)[HarborTokenName] = secret

	return nil
}

func (p *HarborProvisionerPlugin) DeleteEvent(ctx context.Context, event Event, _ PluginData) error {
	org := strings.ToLower(event.Organization)
	name := strings.ToLower(event.Name)
	return p.harbor.DeleteProject(ctx, org, name)
}

func (p *HarborProvisionerPlugin) Name() string {
	return "Harbor Provisioner"
}
