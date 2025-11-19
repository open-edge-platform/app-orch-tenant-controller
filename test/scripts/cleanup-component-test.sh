#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${YELLOW}🧹 Cleaning up component test environment...${NC}"

# Configuration
CLUSTER_NAME=${KIND_CLUSTER_NAME:-"tenant-controller-test"}

# Delete KIND cluster
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${BLUE}🗑️  Deleting KIND cluster: ${CLUSTER_NAME}${NC}"
    kind delete cluster --name "$CLUSTER_NAME"
    echo -e "${GREEN}✅ Cluster deleted successfully${NC}"
else
    echo -e "${YELLOW}⚠️  Cluster ${CLUSTER_NAME} not found${NC}"
fi

# Clean up any leftover processes
echo -e "${BLUE}🧹 Cleaning up processes...${NC}"
pkill -f "kubectl port-forward" 2>/dev/null || true

# Clean up temporary files
rm -f /tmp/kind-config-vip.yaml
rm -f /tmp/keycloak-deployment.yaml
rm -f /tmp/harbor-deployment.yaml
rm -f /tmp/catalog-deployment.yaml
rm -f /tmp/tenant-controller-rbac.yaml

echo -e "${GREEN}✅ environment cleanup completed${NC}"