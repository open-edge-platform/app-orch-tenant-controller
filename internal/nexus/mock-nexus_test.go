// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"context"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

// This module contains mocks for the Nexus client. It maintains an in-memory list of watchers.

type MockNexusOrganization struct {
}

func (o *MockNexusOrganization) DisplayName() string {
	return "MockNexusOrganization"
}

type MockNexusFolder struct {
	parent *MockNexusOrganization
}

func (f *MockNexusFolder) GetParent(ctx context.Context) (NexusOrganizationInterface, error) {
	_ = ctx
	return f.parent, nil
}

type MockNexusProject struct {
	isDeleted      bool
	displayName    string
	uid            string
	parent         *MockNexusFolder
	activeWatchers map[string]*nexus.ProjectactivewatcherProjectActiveWatcher
}

func (p *MockNexusProject) GetActiveWatchers(ctx context.Context, name string) (*nexus.ProjectactivewatcherProjectActiveWatcher, error) {
	_ = ctx
	return p.activeWatchers[name], nil
}

func (p *MockNexusProject) AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (*nexus.ProjectactivewatcherProjectActiveWatcher, error) {
	_ = ctx
	p.activeWatchers[watcher.Name] = &nexus.ProjectactivewatcherProjectActiveWatcher{ProjectActiveWatcher: watcher}

	return p.activeWatchers[watcher.Name], nil
}

func (p *MockNexusProject) DeleteActiveWatchers(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return nil
}

func (p *MockNexusProject) GetParent(ctx context.Context) (NexusFolderInterface, error) {
	_ = ctx
	return p.parent, nil
}

func (p *MockNexusProject) DisplayName() string {
	return p.displayName
}

func (p *MockNexusProject) GetUID() string {
	return p.uid
}

func (p *MockNexusProject) IsDeleted() bool {
	return p.isDeleted
}

func NewMockNexusProject(name string, uid string) *MockNexusProject {
	return &MockNexusProject{
		isDeleted:      false,
		displayName:    name,
		uid:            uid,
		parent:         &MockNexusFolder{},
		activeWatchers: make(map[string]*nexus.ProjectactivewatcherProjectActiveWatcher),
	}
}
