// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package portforward

import (
	"fmt"
	"os/exec"
	"time"
)

const (
	// Service and namespace for port forwarding to tenant controller
	PortForwardServiceNamespace = "orch-app"
	PortForwardService          = "svc/app-orch-tenant-controller"
	PortForwardLocalPort        = "8081"
	PortForwardRemotePort       = "8081"
	PortForwardAddress          = "0.0.0.0"
)

// KillPortForwardToTenantController kills the port forwarding process to tenant controller service
func KillPortForwardToTenantController(cmd *exec.Cmd) error {
	fmt.Println("Killing port forward process to app-orch-tenant-controller")
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

// ToTenantController sets up port forwarding to deployed tenant controller service
// This follows the VIP pattern for component testing
func ToTenantController() (*exec.Cmd, error) {
	fmt.Println("Setting up port forward to app-orch-tenant-controller")

	// #nosec G204 - command arguments are safe constants defined in types package
	cmd := exec.Command("kubectl", "port-forward", "-n", PortForwardServiceNamespace, PortForwardService,
		fmt.Sprintf("%s:%s", PortForwardLocalPort, PortForwardRemotePort),
		"--address", PortForwardAddress)

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start port forwarding: %v", err)
	}

	// Give time for port forwarding to establish
	time.Sleep(5 * time.Second)

	return cmd, nil
}
