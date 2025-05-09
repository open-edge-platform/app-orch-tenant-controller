// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"context"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	//projectwatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectwatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/rest"
)

type MockNexusProject struct {
	isDeleted bool
	displayName string
	uid string
	parent *nexus.RuntimefolderRuntimeFolder
	activeWatchers map[string]*nexus.ProjectactivewatcherProjectActiveWatcher
}

func (p *MockNexusProject) GetActiveWatchers(ctx context.Context, name string) (*nexus.ProjectactivewatcherProjectActiveWatcher, error) {
	return p.activeWatchers[name], nil
}

func (p *MockNexusProject) AddActiveWatchers(ctx context.Context, watcher *projectActiveWatcherv1.ProjectActiveWatcher) (*nexus.ProjectactivewatcherProjectActiveWatcher, error) {
    p.activeWatchers[watcher.Name] = &nexus.ProjectactivewatcherProjectActiveWatcher{ProjectActiveWatcher: watcher}
		
		//&nexus.ProjectactivewatcherProjectActiveWatcherSpec{StatusIndicator: projectActiveWatcherv1.StatusIndicationIdle, Message: "msg"}}
	return p.activeWatchers[watcher.Name], nil
}

func (p *MockNexusProject) DeleteActiveWatchers(ctx context.Context, name string) error {
	return nil
}

func (p *MockNexusProject) GetParent(ctx context.Context) (*nexus.RuntimefolderRuntimeFolder, error) {
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

func NewMockNexusProject(name string, uid string, parent *nexus.RuntimefolderRuntimeFolder) *MockNexusProject {
	return &MockNexusProject{
		isDeleted: false,
		displayName: name,
		uid: uid,
		parent: parent,
		activeWatchers: make(map[string]*nexus.ProjectactivewatcherProjectActiveWatcher),
	}
}