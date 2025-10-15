// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package portforward

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

// Global registry for port forward processes
var (
	portForwardRegistry = make(map[string]*exec.Cmd)
	registryMutex       sync.Mutex
)

// SetupTenantController sets up port forwarding to deployed tenant controller
func SetupTenantController(namespace string, localPort, remotePort int) error {
	return setupPortForward("tenant-controller", namespace, "app-orch-tenant-controller", localPort, remotePort)
}

// SetupKeycloak sets up port forwarding to deployed Keycloak
func SetupKeycloak(namespace string, localPort, remotePort int) error {
	return setupPortForward("keycloak", namespace, "keycloak", localPort, remotePort)
}

// SetupHarbor sets up port forwarding to deployed Harbor
func SetupHarbor(namespace string, localPort, remotePort int) error {
	return setupPortForward("harbor", namespace, "harbor-core", localPort, remotePort)
}

// SetupCatalog sets up port forwarding to deployed Catalog
func SetupCatalog(namespace string, localPort, remotePort int) error {
	return setupPortForward("catalog", namespace, "catalog", localPort, remotePort)
}

// setupPortForward establishes kubectl port-forward to deployed service
func setupPortForward(serviceName, namespace, k8sServiceName string, localPort, remotePort int) error {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	log.Printf("Setting up port forwarding to %s service", serviceName)

	// Kill existing port forward if any
	if cmd, exists := portForwardRegistry[serviceName]; exists && cmd.Process != nil {
		_ = cmd.Process.Kill()
		delete(portForwardRegistry, serviceName)
	}

	// Create kubectl port-forward command to service
	// #nosec G204 -- This is test code with controlled input
	cmd := exec.Command("kubectl", "port-forward",
		"-n", namespace,
		fmt.Sprintf("svc/%s", k8sServiceName),
		fmt.Sprintf("%d:%d", localPort, remotePort))

	// Start port forwarding to service
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start port forwarding to %s: %v", serviceName, err)
	}

	// Register the process
	portForwardRegistry[serviceName] = cmd

	// Give time for port forwarding to establish
	time.Sleep(3 * time.Second)

	log.Printf("Port forwarding to %s established on localhost:%d", serviceName, localPort)
	return nil
}

// Cleanup kills all port forwarding processes
func Cleanup() {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	log.Printf("Cleaning up all port forwarding processes")

	for serviceName, cmd := range portForwardRegistry {
		if cmd.Process != nil {
			log.Printf("Killing port forward to %s", serviceName)
			_ = cmd.Process.Kill()
		}
	}

	// Clear registry
	portForwardRegistry = make(map[string]*exec.Cmd)
}
