// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"context"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

/*
 * This module creates an inferface and abstraction layer for the Nexus API that allows it to easily be mocked.
 *
 * The following structs and their interfaces are defined:
 *
 * - nexus.RuntimeprojectRuntimeProject -> NexusOrganizationInterface
 * - nexus.RuntimefolderRuntimeFolder -> NexusFolder, NexusFolderInterface
 * - nexus.RuntimeprojectRuntimeProject -> NexusProject, NexusProjectInterface
 * - nexus.ProjectactivewatcherProjectActiveWatcher -> NexusProjectActiveWatcher, NexusProjectActiveWatcherInterface
 *
 */

type NexusOrganizationInterface interface { // nolint:revive
	DisplayName() string
}

type NexusFolderInterface interface { // nolint:revive
	GetParent(ctx context.Context) (NexusOrganizationInterface, error)
}

type NexusProjectActiveWatcherInterface interface { // nolint:revive
	Update(ctx context.Context) error
	GetSpec() *projectActiveWatcherv1.ProjectActiveWatcherSpec
    GetAnnotations() map[string]string
    SetAnnotations(annotations map[string]string)
	DisplayName() string
}

type NexusProjectInterface interface { // nolint:revive
	GetActiveWatchers(ctx context.Context, name string) (NexusProjectActiveWatcherInterface, error)
	AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (NexusProjectActiveWatcherInterface, error)
	DeleteActiveWatchers(ctx context.Context, name string) error
	GetParent(ctx context.Context) (NexusFolderInterface, error)
	DisplayName() string
	GetUID() string
	IsDeleted() bool
}

// NexusFolder is an abstraction around the Nexus RuntimefolderRuntimeFolder type

type NexusFolder nexus.RuntimefolderRuntimeFolder // nolint:revive

func (f *NexusFolder) GetParent(ctx context.Context) (NexusOrganizationInterface, error) {
	return (*nexus.RuntimefolderRuntimeFolder)(f).GetParent(ctx)
}

// NexusProjectActiveWatcher is an abstraction around the Nexus ProjectactivewatcherProjectActiveWatcher type

type NexusProjectActiveWatcher nexus.ProjectactivewatcherProjectActiveWatcher // nolint:revive

func (w *NexusProjectActiveWatcher) Update(ctx context.Context) error {
	return (*nexus.ProjectactivewatcherProjectActiveWatcher)(w).Update(ctx)
}

func (w *NexusProjectActiveWatcher) GetSpec() *projectActiveWatcherv1.ProjectActiveWatcherSpec {
	return &w.Spec
}

func (w *NexusProjectActiveWatcher) GetAnnotations() map[string]string {
	return w.GetAnnotations()
}

func (w *NexusProjectActiveWatcher) SetAnnotations(annotations map[string]string) {
	w.SetAnnotations(annotations)
}

func (w *NexusProjectActiveWatcher) DisplayName() string {
	return (*nexus.ProjectactivewatcherProjectActiveWatcher)(w).DisplayName()
}

// NexusProject is an abstraction around the Nexus RuntimeprojectRuntimeProject type

type NexusProject nexus.RuntimeprojectRuntimeProject // nolint:revive

func (p *NexusProject) GetActiveWatchers(ctx context.Context, name string) (NexusProjectActiveWatcherInterface, error) {
	watcherObj, err := (*nexus.RuntimeprojectRuntimeProject)(p).GetActiveWatchers(ctx, name)
	return (*NexusProjectActiveWatcher)(watcherObj), err
}

func (p *NexusProject) AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (NexusProjectActiveWatcherInterface, error) {
	watcherObj, err := (*nexus.RuntimeprojectRuntimeProject)(p).AddActiveWatchers(ctx, watcher)
	return (*NexusProjectActiveWatcher)(watcherObj), err
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
