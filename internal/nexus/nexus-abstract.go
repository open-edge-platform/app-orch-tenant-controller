// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"context"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

// This module creates an inferface and abstraction layer for the Nexus API that allows it to easily be mocked.

type NexusOrganizationInterface interface {
	DisplayName() string
}

type NexusFolderInterface interface {
	GetParent(ctx context.Context) (NexusOrganizationInterface, error)
}

type NexusProjectInterface interface {
	GetActiveWatchers(ctx context.Context, name string) (*nexus.ProjectactivewatcherProjectActiveWatcher, error)
	AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (*nexus.ProjectactivewatcherProjectActiveWatcher, error)
	DeleteActiveWatchers(ctx context.Context, name string) error
	GetParent(ctx context.Context) (NexusFolderInterface, error)
	DisplayName() string
	GetUID() string
	IsDeleted() bool
}

// NexusFolder is a wrapper around the Nexus RuntimefolderRuntimeFolder type

type NexusFolder nexus.RuntimefolderRuntimeFolder

func (f *NexusFolder) GetParent(ctx context.Context) (NexusOrganizationInterface, error) {
	return (*nexus.RuntimefolderRuntimeFolder)(f).GetParent(ctx)
}

// NexusProject is a wrapper around the Nexus RuntimeprojectRuntimeProject type

type NexusProject nexus.RuntimeprojectRuntimeProject

func (p *NexusProject) GetActiveWatchers(ctx context.Context, name string) (*nexus.ProjectactivewatcherProjectActiveWatcher, error) {
	return (*nexus.RuntimeprojectRuntimeProject)(p).GetActiveWatchers(ctx, name)
}

func (p *NexusProject) AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (*nexus.ProjectactivewatcherProjectActiveWatcher, error) {
	return (*nexus.RuntimeprojectRuntimeProject)(p).AddActiveWatchers(ctx, watcher)
}

func (p *NexusProject) DeleteActiveWatchers(ctx context.Context, name string) error {
	return (*nexus.RuntimeprojectRuntimeProject)(p).DeleteActiveWatchers(ctx, name)
}

func (p *NexusProject) GetParent(ctx context.Context) (NexusFolderInterface, error) {
	folder, err := (*nexus.RuntimeprojectRuntimeProject)(p).GetParent(ctx)
	return (*NexusFolder)(folder), err
}

func (p *NexusProject) DisplayName() string {
	return (*nexus.RuntimeprojectRuntimeProject)(p).DisplayName()
}

func (p *NexusProject) GetUID() string {
	return string((*nexus.RuntimeprojectRuntimeProject)(p).UID)
}

func (p *NexusProject) IsDeleted() bool {
	return p.Spec.Deleted
}
