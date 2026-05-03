// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Internal package
package manager

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/plugins"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/tenancy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var log = dazl.GetPackageLogger()

const controllerName = "app-orch-tenant-controller"

// NewManager creates a new manager
func NewManager(config config.Configuration) *Manager {
	return &Manager{
		Config: config,
	}
}

// Manager single point of entry for the config provisioner
type Manager struct {
	Config config.Configuration
	cancel context.CancelFunc
}

// Run starts the provisioner server manager
func (m *Manager) Run() {
	workingDirectory := os.TempDir()
	log.Infof("Starting Manager in %s", workingDirectory)
	_ = os.Chdir(workingDirectory)
	if err := m.Start(); err != nil {
		log.Errorf("Unable to run Manager %v", err)
		log.Fatal("Manager failed to start - exiting to allow Kubernetes restart")
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

	// Start tenancy poller.
	tenantManagerURL := os.Getenv("TENANT_MANAGER_URL")
	if tenantManagerURL == "" {
		tenantManagerURL = "http://tenancy-manager.orch-iam:8080"
	}

	handler := &tenancyHandler{manager: m}
	poller := tenancy.NewPoller(tenantManagerURL, controllerName, handler,
		func(cfg *tenancy.PollerConfig) {
			cfg.OnError = func(err error, msg string) {
				log.Errorf("%s: %v", msg, err)
			}
		},
	)

	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	m.cancel = pollerCancel

	go func() {
		if err := poller.Run(pollerCtx); err != nil && pollerCtx.Err() == nil {
			log.Errorf("poller stopped with error: %v", err)
		}
	}()

	// Wait for a termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("Received shutdown signal, exiting")
	pollerCancel()
	return nil
}

func (m *Manager) handleProjectEvent(event plugins.Event) error {
	startTime := time.Now()
	maxTimeout := m.Config.InitialSleepInterval * 10 * time.Second
	sleepInterval := m.Config.InitialSleepInterval

	var err error

	for {
		ctx, ctxCancel := context.WithTimeout(context.Background(), maxTimeout)
		err = plugins.Dispatch(ctx, event, nil)
		ctxCancel()

		if err == nil {
			return nil
		}
		log.Infof("Error processing event, retrying: %+v", err)

		if time.Since(startTime) > m.Config.MaxWaitTime {
			log.Errorf("Failed to handle event %s within the maximum wait time", event.Name)
			break
		}

		log.Infof("Retrying in %d seconds", int(sleepInterval.Seconds()))
		time.Sleep(sleepInterval)
	}
	return err
}

// Close cancels the poller and cleans up manager resources.
func (m *Manager) Close() {
	log.Info("Closing Manager")
	if m.cancel != nil {
		m.cancel()
	}
}

// HealthCheck is a struct receiver implementing onos northbound Register interface.
type HealthCheck struct{}

// Register is a method to register a health check gRPC service.
func (h HealthCheck) Register(s *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
}

// tenancyHandler implements tenancy.Handler, calling the plugin pipeline
// synchronously so the Poller can accurately report status.
type tenancyHandler struct {
	manager *Manager
}

func (h *tenancyHandler) HandleEvent(_ context.Context, event tenancy.Event) error {
	if event.ResourceType != "project" {
		return nil // app-orch only handles project events
	}

	orgName := ""
	if event.OrgName != nil {
		orgName = *event.OrgName
	}

	e := plugins.Event{
		Organization: orgName,
		Name:         event.ResourceName,
		UUID:         event.ResourceID.String(),
	}

	switch event.EventType {
	case "created":
		e.EventType = "create"
	case "deleted":
		e.EventType = "delete"
	default:
		return nil
	}

	// Run the plugin pipeline synchronously. The Poller sets in_progress
	// before calling HandleEvent and writes completed/error after it
	// returns, so the status accurately reflects the outcome.
	return h.manager.handleProjectEvent(e)
}
