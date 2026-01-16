// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Internal package
package nexus

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/orch-library/go/dazl"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	projectwatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectwatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"strings"
	"time"
)

var log = dazl.GetPackageLogger()

const (
	appName = "config-provisioner"

	// Allow only certain time for interacting with Nexus server
	nexusTimeout = 5 * time.Second

	// Some reasonable limits for names that come from Nexus events, to guard against attack vector on event
	// handling. Note that there is no guarantee the plugins will be able to correctly process names at this
	// length.
	MaxOrganizationNameLength = 63 // Same limit as used in tenant data model
	MaxProjectNameLength      = 63 // Same limit as used in tenant data model
	MaxProjectUUIDLength      = 36
	// manifest tag annotation key
	ManifestTagAnnotationKey = "app-orch-tenant-controller/manifest-tag"
)

type ProjectManager interface {
	CreateProject(orgName string, projectName string, projectUUID string, project NexusProjectInterface)
	DeleteProject(orgName string, projectName string, projectUUID string, project NexusProjectInterface)
	ManifestTag() string
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
	if _, err := h.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").RegisterAddCallback(h.projectCreatedCallback); err != nil {
		log.Errorf("Unable to register project creation callback: %+v", err)
		return err
	}

	if _, err := h.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").RegisterUpdateCallback(h.projectUpdatedCallback); err != nil {
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

func (h *Hook) setProjWatcherStatus(watcherObj NexusProjectActiveWatcherInterface, statusInd projectActiveWatcherv1.ActiveWatcherStatus, status string) error {
	watcherObj.GetSpec().StatusIndicator = statusInd
	watcherObj.GetSpec().Message = status
	watcherObj.GetSpec().TimeStamp = h.safeUnixTime()
	log.Debugf("ProjWatcher object to update: %+v", watcherObj)

	err := watcherObj.Update(context.Background())
	if err != nil {
		log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", err)
		return err
	}
	return nil
}

func (h *Hook) SetWatcherStatusIdle(proj NexusProjectInterface) error {
	watcherObj, err := proj.GetActiveWatchers(context.Background(), appName)
	if err == nil && watcherObj != nil {
		// If watcher exists and is IDLE, simply return.
		if watcherObj.GetSpec().StatusIndicator == projectActiveWatcherv1.StatusIndicationIdle {
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

func (h *Hook) SetWatcherStatusError(proj NexusProjectInterface, message string) error {
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

func (h *Hook) SetWatcherStatusInProgress(proj NexusProjectInterface, message string) error {
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

func (h *Hook) UpdateProjectManifestTag(proj NexusProjectInterface) error {
	log.Infof("Setting watcher manifest tag for project %s to %s", proj.DisplayName(), h.dispatcher.ManifestTag())
	watcherObj, err := proj.GetActiveWatchers(context.Background(), appName)
	if err != nil {
		return err
	}
	if watcherObj != nil {
		log.Debug("Setting watcher annotations")
		annotations := make(map[string]string)
		annotations[ManifestTagAnnotationKey] = h.dispatcher.ManifestTag()
		watcherObj.SetAnnotations(annotations)
		return watcherObj.Update(context.Background())
	}
	return err
}

func (h *Hook) StopWatchingProject(project NexusProjectInterface) {
	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

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

func (h *Hook) deleteProject(project NexusProjectInterface) {
	log.Infof("Project: %+v marked for deletion", project.DisplayName())

	organizationName := h.getOrganizationName(project)
	h.dispatcher.DeleteProject(organizationName, project.DisplayName(), project.GetUID(), project)
}

func (h *Hook) validateArgs(project NexusProjectInterface, organizationName string, projectName string, projectUUID string) error {
	if len(organizationName) == 0 {
		err := h.SetWatcherStatusError(project, "organization name is empty")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Organization name is empty")
	}
	if strings.Contains(organizationName, "\n") {
		err := h.SetWatcherStatusError(project, "Organization name contains illegal characters")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Organization name contains illegal characters")
	}
	if len(organizationName) > MaxOrganizationNameLength {
		err := h.SetWatcherStatusError(project, "Organization name is too long")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Organization name is too long")
	}
	if len(projectName) == 0 {
		err := h.SetWatcherStatusError(project, "project name is empty")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Project name is empty")
	}
	if strings.Contains(projectName, "\n") {
		err := h.SetWatcherStatusError(project, "project name contains illegal characters")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Project name contains illegal characters")
	}
	if len(projectName) > MaxProjectNameLength {
		err := h.SetWatcherStatusError(project, "Project name is too long")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Project name is too long")
	}
	if len(projectUUID) == 0 {
		err := h.SetWatcherStatusError(project, "project UUID is empty")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Project UUID is empty")
	}
	if len(projectUUID) > MaxProjectUUIDLength {
		err := h.SetWatcherStatusError(project, "project UUID is too long")
		if err != nil {
			log.Errorf("Unable to set watcher error status: %v", err)
		}
		return fmt.Errorf("Project UUID is too long")
	}
	return nil
}

func (h *Hook) projectCreatedCallback(nexusProject *nexus.RuntimeprojectRuntimeProject) {
	project := (*NexusProject)(nexusProject)
	err := h.projectCreated(project)
	if err != nil {
		// We're inside a callback, so there's no caller to pass the error up to.
		// We've also potentially failed to create the watcher, so error status won't be reported there either.
		// Just log it and give up.
		log.Errorf("Error in projectCreatedCallback: %v", err)
	}
}

// Callback function to be invoked when Project is added.
func (h *Hook) projectCreated(project NexusProjectInterface) error {
	log.Infof("Runtime Project: %+v created", project.DisplayName())

	if project.IsDeleted() {
		log.Info("Created event for deleted project, dispatching delete event")
		h.deleteProject(project)
		return nil
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

	if err != nil {
		log.Errorf("Failed to create ProjectActiveWatcher object with an error: %v", err)
		return err
	}

	var action string

	if watcherObj.GetSpec().StatusIndicator == projectActiveWatcherv1.StatusIndicationIdle && watcherObj.GetSpec().Message == "Created" {
		// This is a rerun of an event we already processed - check for update
		log.Infof("Watch %s for project %s already provisioned", watcherObj.DisplayName(), project.DisplayName())
		log.Debugf("existing watcher annotations are: %+v", watcherObj.GetAnnotations())
		annotations := watcherObj.GetAnnotations()
		if annotations[ManifestTagAnnotationKey] == h.dispatcher.ManifestTag() {
			// Manifest tag is correct
			log.Infof("Manifest tag is correct, no need to update")
			return nil
		}
		log.Infof("Manifest tag is not correct, updating. Have %s, want %s", annotations[ManifestTagAnnotationKey], h.dispatcher.ManifestTag())
		action = "update"
	} else {
		action = "created"
	}

	if nexus.IsAlreadyExists(err) {
		log.Infof("Watch %s already exists for project %s", watcherObj.DisplayName(), project.DisplayName())
	} else if err != nil {
		// NOTE: This will permantently fail project creation -- there is no recovery if we cannot create the watcher.
		return fmt.Errorf("Error %+v while creating watch %s for project %s", err, appName, project.DisplayName())
	}

	// handle the creation of the project
	organizationName := h.getOrganizationName(project)
	err = h.validateArgs(project, organizationName, project.DisplayName(), project.GetUID())
	if err != nil {
		// If there is an error, validateArgs() will also set the watcher status appropriately.
		return err
	}
	h.dispatcher.CreateProject(organizationName, project.DisplayName(), project.GetUID(), project)

	log.Infof("Active watcher %s %s created for Project %s", watcherObj.DisplayName(), action, project.DisplayName())

	return nil
}

func (h *Hook) getOrganizationName(project NexusProjectInterface) string {
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
func (h *Hook) projectUpdatedCallback(_, nexusProject *nexus.RuntimeprojectRuntimeProject) {
	project := (*NexusProject)(nexusProject)
	h.projectUpdated(project)
}

func (h *Hook) projectUpdated(project NexusProjectInterface) {
	if project.IsDeleted() {
		h.deleteProject(project)
	}
}
