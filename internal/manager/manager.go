// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	nexushook "github.com/open-edge-platform/app-orch-tenant-controller/internal/nexus"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/plugins"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"os"
	"time"
)

var log = dazl.GetPackageLogger()

// NewManager creates a new manager
func NewManager(config config.Configuration) *Manager {
	return &Manager{
		Config: config,
	}
}

// Manager single point of entry for the config provisioner
type Manager struct {
	Config    config.Configuration
	NexusHook *nexushook.Hook
	eventChan chan plugins.Event
}

// Run starts the provisioner server manager
func (m *Manager) Run() {
	workingDirectory := os.TempDir()
	log.Infof("Starting Manager in %s", workingDirectory)
	_ = os.Chdir(workingDirectory)
	if err := m.Start(); err != nil {
		log.Errorf("Unable to run Manager %v", err)
	}
}

// Start starts the provisioner server manager
func (m *Manager) Start() error {
	log.Info("Starting Manager with config:")
	config.DumpConfig(m.Config)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
	defer cancel()

	harborPlugin, err := plugins.NewHarborProvisionerPlugin(ctx, m.Config.HarborServer, m.Config.KeycloakServer, m.Config.HarborNamespace, m.Config.HarborAdminCredential)
	if err != nil {
		return err
	}

	log.Infof("Edge Node manifest path %s%s:%s", m.Config.ReleaseServiceBase, m.Config.ManifestPath, m.Config.ManifestTag)
	catalogPlugin, err := plugins.NewCatalogProvisionerPlugin(m.Config)
	if err != nil {
		return err
	}

	extensionsPlugin, err := plugins.NewExtensionsProvisionerPlugin(m.Config)
	if err != nil {
		return err
	}

	plugins.Register(harborPlugin)
	plugins.Register(catalogPlugin)
	plugins.Register(extensionsPlugin)

	err = plugins.Initialize(context.Background())
	if err != nil {
		return err
	}

	// Create a new Nexus hook.
	m.NexusHook = nexushook.NewNexusHook(m)
	err = m.NexusHook.Subscribe()
	if err != nil {
		log.Errorf("Unable to subscribe to Nexus hook %v", err)
	}

	// set up event handling workers
	m.eventChan = make(chan plugins.Event)

	for i := 0; i < m.Config.NumberWorkerThreads; i++ {
		go m.eventWorker(i)
	}

	// Wait until interrupted
	ready := make(chan os.Signal, 1)
	<-ready
	return nil
}

func (m *Manager) eventWorker(id int) {
	for event := range m.eventChan {
		start := time.Now()
		log.Infof("Event worker %d found work on for project %s", id, event.Name)
		err := m.handleProjectEvent(event)
		if err != nil {
			log.Errorf("Unable to handle project event: %v", err)
			err = m.NexusHook.SetWatcherStatusError(event.Project, err.Error())
			if err != nil {
				log.Errorf("Unable to set watcher error status: %v", err)
			}
		} else {
			// After processing, set the status to IDLE.
			setStatusErr := m.NexusHook.SetWatcherStatusIdle(event.Project)
			if setStatusErr != nil {
				log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", setStatusErr)
				return
			}
			if event.EventType == "delete" {
				// free up the project watcher
				if event.Project != nil && m.NexusHook != nil {
					m.NexusHook.StopWatchingProject(event.Project)
				}
			}
		}
		elapsed := time.Since(start)
		log.Infof("Done with %s on worker %d for project %s elapsed time %d seconds", event.EventType, id, event.Name, int(elapsed.Seconds()))
	}
}

func (m *Manager) handleProjectEvent(event plugins.Event) error {
	startTime := time.Now()
	maxTimeout := m.Config.InitialSleepInterval * 10 * time.Second
	sleepInterval := m.Config.InitialSleepInterval

	var err error

	for {
		ctx, cancel := context.WithTimeout(context.Background(), maxTimeout)

		// dispatch the event
		err = plugins.Dispatch(ctx, event, m.NexusHook)

		cancel()

		if err == nil {
			return err
		}
		log.Infof("Error processing event, retrying: %+v", err)

		// Check if the maximum wait time has been exceeded
		if time.Since(startTime) > m.Config.MaxWaitTime {
			log.Errorf("Failed to handle event %s within the maximum wait time\n", event.Name)
			break
		}

		err = m.NexusHook.SetWatcherStatusInProgress(event.Project, fmt.Sprintf("Retry backoff for project %s. Last error was %s", event.Name, err.Error()))
		if err != nil {
			return err
		}
		log.Infof("Retrying in %d seconds", int(sleepInterval.Seconds()))
		time.Sleep(sleepInterval)
	}
	return err
}

func (m *Manager) CreateProject(organizationName string, projectName string, projectUUID string, project nexushook.NexusProjectInterface) {
	log.Debugf("Creating project with organizationName=%s; projectName=%s; projectUUID=%s", organizationName, projectName, projectUUID)
	e := plugins.Event{
		EventType:    "create",
		Organization: organizationName,
		Name:         projectName,
		UUID:         projectUUID,
		Project:      project,
	}
	m.eventChan <- e
}

func (m *Manager) DeleteProject(organizationName string, projectName string, projectUUID string, project nexushook.NexusProjectInterface) {
	log.Debugf("Deleting project with organizationName=%s; projectName=%s; projectUUID=%s", organizationName, projectName, projectUUID)
	e := plugins.Event{
		EventType:    "delete",
		Organization: organizationName,
		Name:         projectName,
		UUID:         projectUUID,
		Project:      project,
	}
	m.eventChan <- e
}

// Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
	close(m.eventChan)
}

func (m *Manager) ManifestTag() string {
	return m.Config.ManifestTag
}

// HealthCheck is a struct receiver implementing onos northbound Register interface.
type HealthCheck struct{}

// Register is a method to register a health check gRPC service.
func (h HealthCheck) Register(s *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
}
