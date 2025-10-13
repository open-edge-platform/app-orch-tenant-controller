#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up component test environment...${NC}"

# Check if KIND is available
if ! command -v kind &> /dev/null; then
    echo -e "${RED}KIND is not installed. Please install KIND first.${NC}"
    exit 1
fi

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}kubectl is not installed. Please install kubectl first.${NC}"
    exit 1
fi

# Save current kubectl context
ORIGINAL_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "")
if [ -n "$ORIGINAL_CONTEXT" ]; then
    echo -e "${YELLOW}Saving current kubectl context: ${ORIGINAL_CONTEXT}${NC}"
    echo "$ORIGINAL_CONTEXT" > /tmp/original-kubectl-context
else
    echo -e "${YELLOW}No current kubectl context found${NC}"
    echo "" > /tmp/original-kubectl-context
fi

# Check if our test cluster already exists
CLUSTER_NAME=${KIND_CLUSTER_NAME:-"tenant-controller-test"}
CONFIG_FILE=${KIND_CONFIG_FILE:-".github/workflows/kind-config.yaml"}

if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${YELLOW}Test cluster ${CLUSTER_NAME} already exists, checking context...${NC}"
    # Check if the context exists, if not recreate it
    if ! kubectl config get-contexts -o name | grep -q "kind-${CLUSTER_NAME}"; then
        echo -e "${YELLOW}Context for ${CLUSTER_NAME} missing, recreating...${NC}"
        kind delete cluster --name "$CLUSTER_NAME"
        echo -e "${YELLOW}Creating KIND cluster: ${CLUSTER_NAME}${NC}"
        if [ -f "$CONFIG_FILE" ]; then
            kind create cluster --name "$CLUSTER_NAME" --config "$CONFIG_FILE" --wait 300s
        else
            echo -e "${YELLOW}Config file $CONFIG_FILE not found, creating cluster with default settings${NC}"
            kind create cluster --name "$CLUSTER_NAME" --wait 300s
        fi
    else
        echo -e "${GREEN}Test cluster and context already exist, using existing setup${NC}"
    fi
else
    echo -e "${YELLOW}Creating KIND cluster: ${CLUSTER_NAME}${NC}"
    if [ -f "$CONFIG_FILE" ]; then
        kind create cluster --name "$CLUSTER_NAME" --config "$CONFIG_FILE" --wait 300s
    else
        echo -e "${YELLOW}Config file $CONFIG_FILE not found, creating cluster with default settings${NC}"
        kind create cluster --name "$CLUSTER_NAME" --wait 300s
    fi
fi

# Set kubectl context to our test cluster
kubectl config use-context "kind-${CLUSTER_NAME}"

# Create namespaces
echo -e "${YELLOW}Creating test namespaces...${NC}"
kubectl create namespace harbor --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace keycloak --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace orch-app --dry-run=client -o yaml | kubectl apply -f -

# Deploy mock services
echo -e "${YELLOW}Deploying mock services...${NC}"
kubectl apply -f .github/workflows/test-services.yaml

# Wait for services to be ready
echo -e "${YELLOW}Waiting for mock services to be ready...${NC}"

# Function to wait for deployment
wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-300}
    
    echo "Waiting for deployment $deployment in namespace $namespace..."
    kubectl wait --for=condition=available --timeout=${timeout}s deployment/$deployment -n $namespace
}

# Wait for all deployments
wait_for_deployment harbor mock-harbor
wait_for_deployment keycloak mock-keycloak
wait_for_deployment orch-app mock-catalog

# Test service connectivity
echo -e "${YELLOW}Testing service connectivity...${NC}"

# Function to test service endpoint via kubectl port-forward with timeout
test_service_via_kubectl() {
    local name=$1
    local namespace=$2
    local service=$3
    local health_path=$4
    local service_port=80
    local local_port
    
    # Use different local ports to avoid conflicts
    case $name in
        "Harbor") local_port=8080 ;;
        "Keycloak") local_port=8081 ;;
        "Catalog") local_port=8082 ;;
    esac
    
    echo "Testing $name service via kubectl port-forward..."
    
    # Kill any existing port forward on this port
    pkill -f "kubectl.*port-forward.*$service" 2>/dev/null || true
    pkill -f ":$local_port" 2>/dev/null || true
    sleep 1
    
    # Start port forward in background with timeout
    timeout 30s kubectl port-forward -n $namespace svc/$service $local_port:$service_port &
    local pf_pid=$!
    
    # Give port forward time to start
    sleep 3
    
    # Test the endpoint with shorter timeout
    local max_attempts=5
    local attempt=1
    local success=false
    
    while [ $attempt -le $max_attempts ]; do
        if timeout 5s curl -f -s http://localhost:$local_port$health_path > /dev/null 2>&1; then
            echo -e "${GREEN}✓ $name service is ready${NC}"
            success=true
            break
        fi
        echo "Attempt $attempt/$max_attempts for $name service, retrying in 2 seconds..."
        sleep 2
        ((attempt++))
    done
    
    # Clean up port forward
    kill $pf_pid 2>/dev/null || true
    wait $pf_pid 2>/dev/null || true
    
    if [ "$success" = false ]; then
        echo -e "${YELLOW}⚠ $name service test timed out, but continuing (pods should be ready)${NC}"
    fi
}

# Test all services via kubectl port-forward
test_service_via_kubectl "Harbor" "harbor" "mock-harbor" "/api/v2.0/health"
test_service_via_kubectl "Keycloak" "keycloak" "mock-keycloak" "/health"
test_service_via_kubectl "Catalog" "orch-app" "mock-catalog" "/health"

echo -e "${GREEN}Component test environment setup complete!${NC}"
echo -e "${GREEN}Services are deployed and accessible via kubectl port-forward${NC}"
echo -e "  Harbor:   kubectl port-forward -n harbor svc/mock-harbor 8080:80"
echo -e "  Keycloak: kubectl port-forward -n keycloak svc/mock-keycloak 8081:80"
echo -e "  Catalog:  kubectl port-forward -n orch-app svc/mock-catalog 8082:80"
echo ""
echo -e "${GREEN}To run component tests:${NC}"
echo -e "  make component-test"
echo ""
echo -e "${GREEN}To cleanup:${NC}"
echo -e "  ./test/scripts/cleanup-test-env.sh"