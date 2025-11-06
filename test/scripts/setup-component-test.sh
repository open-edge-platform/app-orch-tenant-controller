#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Setup script for tenant controller component tests
# This script assumes orchestrator is DEPLOYED
# Validate connectivity to existing services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}🚀 Tenant Controller Component Test Setup${NC}"
echo -e "${BLUE}Validating connection to deployed orchestrator services...${NC}"

# Orchestrator already be deployed - just verify connectivity
ORCH_DOMAIN=${ORCH_DOMAIN:-"kind.internal"}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}📋 Checking prerequisites...${NC}"
    
    local missing_tools=()
    
    # Check required tools
    for tool in kubectl go; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        echo -e "${RED}❌ Missing required tools: ${missing_tools[*]}${NC}"
        echo -e "${YELLOW}Please install the missing tools and try again.${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ All prerequisites met${NC}"
}

# Verify orchestrator is deployed and accessible
verify_orchestrator() {
    echo -e "${BLUE}� Verifying orchestrator deployment...${NC}"
    
    # Check if kubectl can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
        echo -e "${YELLOW}Make sure kubectl is configured correctly${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Kubernetes cluster accessible${NC}"
    
    # Check for key orchestrator namespaces
    local required_namespaces=("orch-app" "orch-system")
    for ns in "${required_namespaces[@]}"; do
        if ! kubectl get namespace "$ns" &> /dev/null; then
            echo -e "${YELLOW}⚠️  Namespace $ns not found - orchestrator may not be fully deployed${NC}"
        else
            echo -e "${GREEN}✅ Namespace $ns exists${NC}"
        fi
    done
    
    # Check if app-orch-tenant-controller deployment exists
    if kubectl get deployment app-orch-tenant-controller -n orch-app &> /dev/null; then
        echo -e "${GREEN}✅ Tenant controller deployment found${NC}"
        
        # Get pod status
        local pod_status=$(kubectl get pods -n orch-app -l app.kubernetes.io/instance=app-orch-tenant-controller -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "NotFound")
        echo -e "${BLUE}📋 Tenant controller pod status: $pod_status${NC}"
    else
        echo -e "${YELLOW}⚠️  Tenant controller deployment not found${NC}"
    fi
    
    echo -e "${GREEN}✅ Orchestrator verification completed${NC}"
}

# Print test environment information
print_environment_info() {
    echo ""
    echo -e "${GREEN}🎉 Component test environment ready!${NC}"
    echo ""
    echo -e "${BLUE}📋 Environment Information:${NC}"
    echo -e "  Domain: ${ORCH_DOMAIN}"
    echo -e "  Cluster: $(kubectl config current-context)"
    echo ""
    echo -e "${BLUE}📊 Key Services Status:${NC}"
    kubectl get pods -n orch-app -l app.kubernetes.io/instance=app-orch-tenant-controller 2>/dev/null || echo "  Tenant controller: Not found in orch-app"
    kubectl get pods -n orch-system -l app.kubernetes.io/name=keycloak 2>/dev/null | tail -n +2 || echo "  Keycloak: Not found in orch-system"
    echo ""
    echo -e "${BLUE}🧪 Ready to run component tests${NC}"
    echo ""
}

# Main execution flow
main() {
    check_prerequisites
    verify_orchestrator
    print_environment_info
}

# Execute main function
main