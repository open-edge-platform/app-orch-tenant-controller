// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"context"
	"errors"
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

type MockNexusProjectActiveWatcher struct {
	nexus.ProjectactivewatcherProjectActiveWatcher
	annotations map[string]string
}

func (w *MockNexusProjectActiveWatcher) Update(ctx context.Context) error {
	_ = ctx
	return nil
}

func (w *MockNexusProjectActiveWatcher) GetSpec() *projectActiveWatcherv1.ProjectActiveWatcherSpec {
	return &w.Spec
}

func (w *MockNexusProjectActiveWatcher) GetAnnotations() map[string]string {
	if w.annotations == nil {
		w.annotations = make(map[string]string)
	}
	return w.annotations
}

func (w *MockNexusProjectActiveWatcher) SetAnnotations(annotations map[string]string) {
	w.annotations = annotations
}

func (w *MockNexusProjectActiveWatcher) DisplayName() string {
	return w.Name
}

type MockNexusProject struct {
	isDeleted      bool
	displayName    string
	uid            string
	parent         *MockNexusFolder
	activeWatchers map[string]*MockNexusProjectActiveWatcher
}

func (p *MockNexusProject) GetActiveWatchers(ctx context.Context, name string) (NexusProjectActiveWatcherInterface, error) {
	_ = ctx
	return p.activeWatchers[name], nil
}

func (p *MockNexusProject) AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (NexusProjectActiveWatcherInterface, error) {
	_ = ctx
	
	// Check if watcher already exists
	if existingWatcher, exists := p.activeWatchers[watcher.Name]; exists {
		return existingWatcher, errors.New("already exists")
	}
	
	mockWatcher := &MockNexusProjectActiveWatcher{
		ProjectactivewatcherProjectActiveWatcher: nexus.ProjectactivewatcherProjectActiveWatcher{ProjectActiveWatcher: watcher},
		annotations: make(map[string]string),
	}
	p.activeWatchers[watcher.Name] = mockWatcher
	return mockWatcher, nil
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
		activeWatchers: make(map[string]*MockNexusProjectActiveWatcher),
	}
}
