// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus

import (
	"context"
	"github.com/labstack/gommon/log"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	projectwatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectwatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"time"
)

const (
	appName = "config-provisioner"

	// Allow only certain time for interacting with Nexus server
	nexusTimeout = 5 * time.Second
)

type ProjectManager interface {
	CreateProject(orgName string, projectName string, projectUUID string, project *nexus.RuntimeprojectRuntimeProject)
	DeleteProject(orgName string, projectName string, projectUUID string, project *nexus.RuntimeprojectRuntimeProject)
}

type Hook struct {
	dispatcher  ProjectManager
	nexusClient *nexus.Clientset
}

// NewNexusHook creates a new hook for receiving project lifecycle events from Nexus.
func NewNexusHook(dispatcher ProjectManager) *Hook {
	return &Hook{dispatcher: dispatcher}
}

// Subscribe issues all required subscriptions for receiving project lifecycle events.
func (h *Hook) Subscribe() error {
	// Initialize Nexus SDK, by pointing it to the K8s API endpoint where CRD's are to be stored.
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("Unable to load in-cluster configuration: %+v", err)
		return err
	}

	h.nexusClient, err = nexus.NewForConfig(cfg)
	if err != nil {
		log.Errorf("Unable to create Nexus configuration: %+v", err)
		return err
	}

	// Register the configuration provisioner watcher node in the configuration subtree.
	if err := h.setupConfigProvisionerWatcherConfig(); err != nil {
		return err
	}

	// Subscribe to Multi-Tenancy graph.
	// Subscribe() api empowers subscription to objects from datamodel.
	// What subscription does is to keep the local cache in sync with datamodel changes.
	// This sync is done in the background.
	h.nexusClient.SubscribeAll()

	// API to subscribe and register a callback function that is invoked when a Project is added in the datamodel.
	// Register*Callback() has the effect of subscription and also invoking a callback to the application code
	// when there are datamodel changes to the objects of interest.
	if _, err := h.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").RegisterAddCallback(h.projectCreated); err != nil {
		log.Errorf("Unable to register project creation callback: %+v", err)
		return err
	}

	if _, err := h.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").RegisterUpdateCallback(h.projectUpdated); err != nil {
		log.Errorf("Unable to register project deletion callback: %+v", err)
		return err
	}
	log.Info("Nexus hook successfully subscribed")
	return nil
}

func (h *Hook) setupConfigProvisionerWatcherConfig() error {
	tenancy := h.nexusClient.TenancyMultiTenancy()

	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	projWatcher, err := tenancy.Config().AddProjectWatchers(ctx, &projectwatcherv1.ProjectWatcher{ObjectMeta: metav1.ObjectMeta{
		Name: appName,
	}})
	if nexus.IsAlreadyExists(err) {
		log.Warnf("Project watcher already exist: appName=%s, projWatcher=%v", appName, projWatcher)
	} else if err != nil {
		log.Errorf("Failed to create project watcher: appName=%s", appName)
		return err
	}
	log.Infof("Created project watcher: appName=%s, projWatcher=%v", appName, projWatcher)
	return nil
}

func (h *Hook) safeUnixTime() uint64 {
	t := time.Now().Unix()
	if t < 0 {
		return 0
	}
	return uint64(t)
}

func (h *Hook) setProjWatcherStatus(watcherObj *nexus.ProjectactivewatcherProjectActiveWatcher, statusInd projectActiveWatcherv1.ActiveWatcherStatus, status string) error {
	watcherObj.Spec.StatusIndicator = statusInd
	watcherObj.Spec.Message = status
	watcherObj.Spec.TimeStamp = h.safeUnixTime()
	log.Debugf("ProjWatcher object to update: %+v", watcherObj)

	err := watcherObj.Update(context.Background())
	if err != nil {
		log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", err)
		return err
	}
	return nil
}

func (h *Hook) SetWatcherStatusIdle(proj *nexus.RuntimeprojectRuntimeProject) error {
	watcherObj, err := proj.GetActiveWatchers(context.Background(), appName)
	if err == nil && watcherObj != nil {
		// If watcher exists and is IDLE, simply return.
		if watcherObj.Spec.StatusIndicator == projectActiveWatcherv1.StatusIndicationIdle {
			log.Infof("Skipping processing of projectactivewatcher %v as it is already created and set to IDLE", appName)
			return nil
		}

		// If watcher exists and is not IDLE, mark it as idle
		setStatusErr := h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationIdle, "Created")
		if setStatusErr != nil {
			log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", setStatusErr)
			return setStatusErr
		}
		return nil
	}
	return err
}

func (h *Hook) SetWatcherStatusError(proj *nexus.RuntimeprojectRuntimeProject, message string) error {
	watcherObj, err := proj.GetActiveWatchers(context.Background(), appName)
	if err == nil && watcherObj != nil {
		setStatusErr := h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationError, message)
		if setStatusErr != nil {
			log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", setStatusErr)
			return setStatusErr
		}
		return nil
	}
	return err
}

func (h *Hook) SetWatcherStatusInProgress(proj *nexus.RuntimeprojectRuntimeProject, message string) error {
	watcherObj, err := proj.GetActiveWatchers(context.Background(), appName)
	log.Infof("Setting watcher status to InProgress for project %s to %s", proj.DisplayName(), message)
	if err == nil && watcherObj != nil {
		setStatusErr := h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationInProgress, message)
		if setStatusErr != nil {
			log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", setStatusErr)
			return setStatusErr
		}
		return nil
	}
	return err
}

func (h *Hook) deleteProject(project *nexus.RuntimeprojectRuntimeProject) {
	log.Infof("Project: %+v marked for deletion", project.DisplayName())

	organizationName := h.getOrganizationName(project)
	h.dispatcher.DeleteProject(organizationName, project.DisplayName(), string(project.UID), project)

	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	// After processing, set the status to IDLE.
	setStatusErr := h.SetWatcherStatusIdle(project)
	if setStatusErr != nil {
		log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", setStatusErr)
		return
	}

	// Stop watching the project as it is marked for deletion.
	err := project.DeleteActiveWatchers(ctx, appName)
	if nexus.IsChildNotFound(err) {
		// This app has already stopped watching the project.
		log.Warnf("App %s DOES NOT watch project %s", appName, project.DisplayName())
		return
	} else if err != nil {
		log.Errorf("Error %+v while deleting watch %s for project %s", err, appName, project.DisplayName())
		return
	}
	log.Infof("Active watcher %s deleted for project %s", appName, project.DisplayName())
}

// Callback function to be invoked when Project is added.
func (h *Hook) projectCreated(project *nexus.RuntimeprojectRuntimeProject) {
	log.Infof("Runtime Project: %+v created", *project)

	if project.Spec.Deleted {
		log.Info("Created event for deleted project, dispatching delete event")
		h.deleteProject(project)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	// Register this app as an active watcher for this project.
	watcherObj, err := project.AddActiveWatchers(ctx, &projectActiveWatcherv1.ProjectActiveWatcher{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
		Spec: projectActiveWatcherv1.ProjectActiveWatcherSpec{
			StatusIndicator: projectActiveWatcherv1.StatusIndicationInProgress,
			Message:         "Creating",
			TimeStamp:       h.safeUnixTime(),
		},
	})
	if watcherObj.Spec.StatusIndicator == projectActiveWatcherv1.StatusIndicationIdle && watcherObj.Spec.Message == "Created" {
		// This is a rerun of an event we already processed - no more processing required
		log.Infof("Watch %s for project %s already provisioned", watcherObj.DisplayName(), project.DisplayName())
		return
	}

	if nexus.IsAlreadyExists(err) {
		log.Warnf("Watch %s already exists for project %s", watcherObj.DisplayName(), project.DisplayName())
	} else if err != nil {
		log.Errorf("Error %+v while creating watch %s for project %s", err, appName, project.DisplayName())
	}

	// handle the creation of the project
	organizationName := h.getOrganizationName(project)
	h.dispatcher.CreateProject(organizationName, project.DisplayName(), string(project.UID), project)

	log.Infof("Active watcher %s created for Project %s", watcherObj.DisplayName(), project.DisplayName())
}

func (h *Hook) getOrganizationName(project *nexus.RuntimeprojectRuntimeProject) string {
	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	// TODO: Revisit this. For now, demote errors to mere warnings.
	folderOrgs, err := project.GetParent(ctx)
	if err != nil {
		log.Warnf("Unable to get project parent folder: %v", err)
		return ""
	}

	organization, err := folderOrgs.GetParent(context.Background())
	if err != nil {
		log.Warnf("Unable to get parent folder organization: %v", err)
		return ""
	}
	return organization.DisplayName()
}

// Callback function to be invoked when Project is deleted.
func (h *Hook) projectUpdated(_, project *nexus.RuntimeprojectRuntimeProject) {
	if project.Spec.Deleted {
		h.deleteProject(project)
	}
}
