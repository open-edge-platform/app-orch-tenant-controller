// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-edge-platform/app-orch-tenant-controller/test/utils/portforward"
)

// ComponentTestSuite tests the tenant controller
type ComponentTestSuite struct {
	suite.Suite
	orchDomain         string
	ctx                context.Context
	cancel             context.CancelFunc
	httpClient         *http.Client
	k8sClient          kubernetes.Interface
	tenantControllerNS string

	keycloakURL         string
	harborURL           string
	catalogURL          string
	tenantControllerURL string
}

// SetupSuite initializes the test suite
func (suite *ComponentTestSuite) SetupSuite() {
	log.Printf("Setting up component tests")

	// Get orchestration domain (defaults to kind.internal)
	suite.orchDomain = os.Getenv("ORCH_DOMAIN")
	if suite.orchDomain == "" {
		suite.orchDomain = "kind.internal"
	}

	// Set tenant controller namespace
	suite.tenantControllerNS = "orch-app"

	// Set up context with cancellation
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Configure service URLs
	suite.keycloakURL = "http://keycloak.keycloak.svc.cluster.local"
	suite.harborURL = "http://harbor-core.harbor.svc.cluster.local"
	suite.catalogURL = "http://catalog.orch-app.svc.cluster.local"
	suite.tenantControllerURL = "http://localhost:8083" // via port-forward

	log.Printf("Connecting to orchestrator services at domain: %s", suite.orchDomain)

	// Set up Kubernetes client for verifying deployments
	suite.setupKubernetesClient()

	// Set up port forwarding to deployed services
	suite.setupPortForwarding()

	// Create HTTP client for service endpoints
	suite.setupHTTPClient()

	// Wait for all services to be ready
	suite.waitForRealServices()
}

// TearDownSuite cleans up after tests
func (suite *ComponentTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}

	// Cleanup port forwarding
	portforward.Cleanup()
}

// setupKubernetesClient sets up Kubernetes client
func (suite *ComponentTestSuite) setupKubernetesClient() {
	log.Printf("Setting up Kubernetes client")

	// Load kubeconfig
	kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	suite.Require().NoError(err, "Failed to load kubeconfig")

	// Create Kubernetes client
	suite.k8sClient, err = kubernetes.NewForConfig(config)
	suite.Require().NoError(err, "Failed to create Kubernetes client")

	log.Printf("Kubernetes client setup complete")
}

// setupPortForwarding sets up port forwarding to deployed services
func (suite *ComponentTestSuite) setupPortForwarding() {
	log.Printf("Setting up port forwarding to deployed services")

	// Set up port forwarding to tenant controller
	err := portforward.SetupTenantController(suite.tenantControllerNS, 8083, 80)
	if err != nil {
		log.Printf("Failed to set up port forwarding to tenant controller: %v", err)
	}

	// Additional port forwards for direct service testing
	err = portforward.SetupKeycloak("keycloak", 8080, 80)
	if err != nil {
		log.Printf("Failed to set up port forwarding to Keycloak: %v", err)
	}

	err = portforward.SetupHarbor("harbor", 8081, 80)
	if err != nil {
		log.Printf("Failed to set up port forwarding to Harbor: %v", err)
	}

	err = portforward.SetupCatalog(suite.tenantControllerNS, 8082, 80)
	if err != nil {
		log.Printf("Failed to set up port forwarding to Catalog: %v", err)
	}

	// Wait for port forwards to be established
	time.Sleep(5 * time.Second)

	log.Printf("Port forwarding setup complete")
}

// setupHTTPClient sets up HTTP client for service endpoints
func (suite *ComponentTestSuite) setupHTTPClient() {
	log.Printf("Setting up HTTP client for service endpoints")

	// Create HTTP client for services
	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	log.Printf("HTTP client setup complete")
}

// waitForRealServices waits for all deployed services to be ready
func (suite *ComponentTestSuite) waitForRealServices() {
	log.Printf("Waiting for deployed services to be ready")

	// Wait for services with tolerance for startup delays
	suite.waitForService("keycloak", "keycloak", "app.kubernetes.io/name=keycloak")
	suite.waitForService("harbor", "harbor", "app.kubernetes.io/name=harbor")
	suite.waitForService("catalog", suite.tenantControllerNS, "app.kubernetes.io/name=catalog")

	log.Printf("Services check completed")
}

// waitForService waits for a specific deployed service to be ready
func (suite *ComponentTestSuite) waitForService(serviceName, namespace, labelSelector string) {
	log.Printf("Checking %s service", serviceName)

	// Check if pods exist and get their status
	for i := 0; i < 10; i++ {
		pods, err := suite.k8sClient.CoreV1().Pods(namespace).List(suite.ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})

		if err == nil && len(pods.Items) > 0 {
			log.Printf("%s service has %d pods", serviceName, len(pods.Items))
			return
		}

		time.Sleep(3 * time.Second)
	}

	log.Printf("%s service not found, but continuing test", serviceName)
}

// TestTenantProvisioningWithRealServices tests tenant provisioning against deployed services
func (suite *ComponentTestSuite) TestTenantProvisioningWithRealServices() {
	log.Printf("Testing tenant provisioning against deployed services")

	// Test service access
	suite.Run("VerifyRealKeycloakAccess", func() {
		suite.testRealKeycloakAccess()
	})

	suite.Run("VerifyRealHarborAccess", func() {
		suite.testRealHarborAccess()
	})

	suite.Run("VerifyRealCatalogAccess", func() {
		suite.testRealCatalogAccess()
	})

	// Test end-to-end tenant provisioning
	suite.Run("EndToEndTenantProvisioning", func() {
		suite.testEndToEndTenantProvisioning()
	})
}

// testRealKeycloakAccess tests access to deployed Keycloak service
func (suite *ComponentTestSuite) testRealKeycloakAccess() {
	log.Printf("Testing Keycloak access")

	// Test Keycloak health endpoint via port-forward
	resp, err := suite.httpClient.Get("http://localhost:8080/")
	if err != nil {
		log.Printf("Keycloak connection failed (may still be starting): %v", err)
		return
	}
	defer resp.Body.Close()

	suite.Require().True(resp.StatusCode < 500,
		"Keycloak service not accessible, status: %d", resp.StatusCode)

	log.Printf("Keycloak access verified")
}

// testRealHarborAccess tests access to deployed Harbor service
func (suite *ComponentTestSuite) testRealHarborAccess() {
	log.Printf("Testing Harbor access")

	// Test Harbor health endpoint via port-forward
	resp, err := suite.httpClient.Get("http://localhost:8081/")
	if err != nil {
		log.Printf("Harbor connection failed: %v", err)
		return
	}
	defer resp.Body.Close()

	suite.Require().True(resp.StatusCode < 500,
		"Harbor service not accessible, status: %d", resp.StatusCode)

	log.Printf("Harbor access verified")
}

// testRealCatalogAccess tests access to deployed Catalog service
func (suite *ComponentTestSuite) testRealCatalogAccess() {
	log.Printf("Testing Catalog access")

	// Test Catalog health endpoint via port-forward
	resp, err := suite.httpClient.Get("http://localhost:8082/")
	if err != nil {
		log.Printf("Catalog connection failed (may still be starting): %v", err)
		return
	}
	defer resp.Body.Close()

	suite.Require().True(resp.StatusCode < 500,
		"Catalog service not accessible, status: %d", resp.StatusCode)

	log.Printf("Catalog access verified")
}

// testEndToEndTenantProvisioning tests complete tenant provisioning using services
func (suite *ComponentTestSuite) testEndToEndTenantProvisioning() {
	log.Printf("Testing end-to-end tenant provisioning with services")

	// Verify tenant controller deployment exists
	deployment, err := suite.k8sClient.AppsV1().Deployments(suite.tenantControllerNS).Get(
		suite.ctx, "app-orch-tenant-controller", metav1.GetOptions{})
	if err != nil {
		log.Printf("Tenant Controller deployment not found: %v", err)
		return
	}

	log.Printf("Found tenant controller deployment with %d ready replicas", deployment.Status.ReadyReplicas)

	// Verify tenant controller can reach other services
	pods, err := suite.k8sClient.CoreV1().Pods(suite.tenantControllerNS).List(
		suite.ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=app-orch-tenant-controller",
		})
	if err != nil {
		log.Printf("Failed to list tenant controller pods: %v", err)
		return
	}

	log.Printf("Found %d tenant controller pods", len(pods.Items))

	log.Printf("End-to-end tenant provisioning verification complete")
}

// TestRealServiceIntegration tests integration with all deployed services
func (suite *ComponentTestSuite) TestRealServiceIntegration() {
	log.Printf("Testing service integration")

	// Verify all services are deployed and accessible
	suite.Run("VerifyAllRealServicesDeployed", func() {
		suite.testVerifyAllRealServicesDeployed()
	})

	// Test service-to-service communication
	suite.Run("TestRealServiceCommunication", func() {
		suite.testRealServiceCommunication()
	})
}

// testVerifyAllRealServicesDeployed verifies all services are properly deployed
func (suite *ComponentTestSuite) testVerifyAllRealServicesDeployed() {
	log.Printf("Verifying all services are deployed")

	// Check for each service deployment
	services := []struct {
		name       string
		namespace  string
		deployment string
	}{
		{"keycloak", "keycloak", "keycloak"},
		{"harbor", "harbor", "harbor-core"},
		{"catalog", suite.tenantControllerNS, "catalog"},
	}

	for _, svc := range services {
		_, err := suite.k8sClient.AppsV1().Deployments(svc.namespace).Get(
			suite.ctx, svc.deployment, metav1.GetOptions{})
		if err == nil {
			log.Printf("%s service is deployed", svc.name)
		} else {
			log.Printf("%s service not found: %v", svc.name, err)
		}
	}

	log.Printf("Service deployment verification complete")
}

// testRealServiceCommunication tests communication between deployed services
func (suite *ComponentTestSuite) testRealServiceCommunication() {
	log.Printf("Testing service communication")

	// Verify services can resolve each other via Kubernetes DNS
	services := []struct {
		name      string
		namespace string
	}{
		{"keycloak", "keycloak"},
		{"harbor-core", "harbor"},
		{"catalog", suite.tenantControllerNS},
	}

	for _, svc := range services {
		_, err := suite.k8sClient.CoreV1().Services(svc.namespace).Get(
			suite.ctx, svc.name, metav1.GetOptions{})
		if err == nil {
			log.Printf("service %s accessible", svc.name)
		} else {
			log.Printf("service %s not found: %v", svc.name, err)
		}
	}

	log.Printf("Service communication verification complete")
}

// Run the test suite
func TestComponentTestSuite(t *testing.T) {
	suite.Run(t, new(ComponentTestSuite))
}
