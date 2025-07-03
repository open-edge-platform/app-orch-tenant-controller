// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/southbound"
)

type Harbor interface {
	Configurations(ctx context.Context) error
	CreateProject(ctx context.Context, org string, displayName string) error
	SetMemberPermissions(ctx context.Context, roleID int, org string, displayName string, groupName string) error
	CreateRobot(ctx context.Context, robotName string, org string, displayName string) (string, string, int, error)
	GetRobot(ctx context.Context, org string, displayName string, robotName string) (*southbound.HarborRobot, error)
	DeleteRobot(ctx context.Context, org string, displayName string, robotID int) error
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

func (p *HarborProvisionerPlugin) waitForHarbor(ctx context.Context) {
	log.Info("Waiting for Harbor")
	harbor, _ := HarborFactory(ctx, p.harborHost, p.oidcURL, p.harborNamespace, p.harborAdminCredential)

	for {
		lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := harbor.Ping(lctx)
		cancel()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	log.Info("Harbor ready")
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
	p.waitForHarbor(ctx)
	return p.harbor.Configurations(ctx)
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

	/* Leaving this check in because it was known to work for older harbor versions.
	 * On the current Harbot version it is returning a 404 because the /projects/{project_name}/robots endpoint does not exist.
	 * TODO: Delete this code when appropriate.
	 */
	robot, _ := p.harbor.GetRobot(ctx, org, name, "catalog-apps-read-write")
	if robot != nil {
		err = p.harbor.DeleteRobot(ctx, org, name, robot.ID)
		if err != nil {
			return err
		}
	}

	var secret string
	var statusCode int
	name, secret, statusCode, err = p.harbor.CreateRobot(ctx, `catalog-apps-read-write`, org, name)
	if err != nil && statusCode == http.StatusConflict {
		log.Info("Robot already exists, trying to delete and recreate")
		err = p.harbor.DeleteRobot(ctx, org, name, robot.ID)
		if err != nil {
			return err
		}
		name, secret, statusCode, err = p.harbor.CreateRobot(ctx, `catalog-apps-read-write`, org, name)
		if err != nil {
			return err
		}
	} else if err != nil {
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
