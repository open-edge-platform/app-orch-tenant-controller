// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestComponentTests is the main test runner for component tests
func TestComponentTests(t *testing.T) {
	t.Log("🎯 Running Component Tests for App Orchestration Tenant Controller")
	t.Log("")
	t.Log("Component tests validate:")
	t.Log("  ✓ Plugin integration (Harbor, Catalog, Extensions)")
	t.Log("  ✓ Manager event handling and project lifecycle")
	t.Log("  ✓ Nexus hook integration and watcher management")
	t.Log("  ✓ Southbound service communications")
	t.Log("  ✓ Error handling and recovery scenarios")
	t.Log("  ✓ Concurrent operations and thread safety")
	t.Log("")

	// Run plugin component tests
	t.Run("PluginComponents", func(t *testing.T) {
		suite.Run(t, new(PluginComponentTests))
	})

	// Run manager component tests
	t.Run("ManagerComponents", func(t *testing.T) {
		suite.Run(t, new(ManagerComponentTests))
	})

	// Run nexus hook component tests
	t.Run("NexusHookComponents", func(t *testing.T) {
		suite.Run(t, new(NexusHookComponentTests))
	})

	// Run southbound component tests
	t.Run("SouthboundComponents", func(t *testing.T) {
		suite.Run(t, new(SouthboundComponentTests))
	})

	t.Log("")
	t.Log("🎉 Component Test Suite Complete")
}
