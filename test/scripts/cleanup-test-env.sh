#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Cleaning up component test environment...${NC}"

CLUSTER_NAME=${KIND_CLUSTER_NAME:-"tenant-controller-test"}

# Only delete the specific test cluster, not any existing clusters
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${YELLOW}Deleting test-specific KIND cluster: ${CLUSTER_NAME}${NC}"
    kind delete cluster --name "$CLUSTER_NAME"
else
    echo -e "${YELLOW}Test cluster ${CLUSTER_NAME} not found, skipping deletion${NC}"
fi

# Clean up any leftover processes
echo -e "${YELLOW}Cleaning up any remaining processes...${NC}"
pkill -f "kind.*${CLUSTER_NAME}" || true
pkill -f "kubectl.*port-forward" || true

# Restore original kubectl context
if [ -f /tmp/original-kubectl-context ]; then
    ORIGINAL_CONTEXT=$(cat /tmp/original-kubectl-context)
    if [ -n "$ORIGINAL_CONTEXT" ] && [ "$ORIGINAL_CONTEXT" != "" ]; then
        echo -e "${YELLOW}Restoring original kubectl context: ${ORIGINAL_CONTEXT}${NC}"
        kubectl config use-context "$ORIGINAL_CONTEXT" || {
            echo -e "${YELLOW}Warning: Could not restore original context ${ORIGINAL_CONTEXT}${NC}"
            echo -e "${YELLOW}Available contexts:${NC}"
            kubectl config get-contexts || true
        }
    else
        echo -e "${YELLOW}No original kubectl context to restore${NC}"
    fi
    rm -f /tmp/original-kubectl-context
else
    echo -e "${YELLOW}No original kubectl context file found${NC}"
fi

echo -e "${GREEN}Component test environment cleanup complete!${NC}"