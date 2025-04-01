// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
	"time"
)

// Suite of manager tests
type ManagerTestSuite struct {
	suite.Suite
}

func (s *ManagerTestSuite) SetupSuite() {
}

func (s *ManagerTestSuite) TearDownSuite() {
}

func (s *ManagerTestSuite) SetupTest() {
}

func (s *ManagerTestSuite) TearDownTest() {
}

func TestManager(t *testing.T) {
	suite.Run(t, &ManagerTestSuite{})
}

func (s *ManagerTestSuite) clearEnvironment() {
	_ = os.Unsetenv("RS_ROOT_URL")
	_ = os.Unsetenv("RS_PROXY_ROOT_URL")
	_ = os.Unsetenv("MANIFEST_PATH")
	_ = os.Unsetenv("MANIFEST_TAG")
	_ = os.Unsetenv("REGISTRY_HOST_EXTERNAL")
	_ = os.Unsetenv("CATALOG_SERVER")
	_ = os.Unsetenv("HARBOR_SERVER")
	_ = os.Unsetenv("HARBOR_NAMESPACE")
	_ = os.Unsetenv("HARBOR_ADMIN_CREDENTIAL")
	_ = os.Unsetenv("KEYCLOAK_SERVER")
	_ = os.Unsetenv("VAULT_SERVER")
	_ = os.Unsetenv("SERVICE_ACCOUNT")
	_ = os.Unsetenv("KEYCLOAK_NAMESPACE")
	_ = os.Unsetenv("KEYCLOAK_SECRET")
	_ = os.Unsetenv("ADM_SERVER")
	_ = os.Unsetenv("RELEASE_SERVICE_BASE")
	_ = os.Unsetenv("INITIAL_SLEEP_INTERVAL")
	_ = os.Unsetenv("MAX_WAIT_TIME")
}

func (s *ManagerTestSuite) TestInit() {
	s.clearEnvironment()
	_ = os.Setenv("RS_ROOT_URL", "RS_ROOT_URL")
	_ = os.Setenv("RS_PROXY_ROOT_URL", "RS_PROXY_ROOT_URL")
	_ = os.Setenv("MANIFEST_PATH", "MANIFEST_PATH")
	_ = os.Setenv("MANIFEST_TAG", "MANIFEST_TAG")
	_ = os.Setenv("REGISTRY_HOST_EXTERNAL", "REGISTRY_HOST_EXTERNAL")
	_ = os.Setenv("CATALOG_SERVER", "CATALOG_SERVER")
	_ = os.Setenv("HARBOR_SERVER", "HARBOR_SERVER")
	_ = os.Setenv("HARBOR_NAMESPACE", "HARBOR_NAMESPACE")
	_ = os.Setenv("HARBOR_ADMIN_CREDENTIAL", "HARBOR_ADMIN_CREDENTIAL")
	_ = os.Setenv("KEYCLOAK_SERVER", "KEYCLOAK_SERVER")
	_ = os.Setenv("KEYCLOAK_SERVICE_BASE", "KEYCLOAK_SERVICE_BASE")
	_ = os.Setenv("VAULT_SERVER", "VAULT_SERVER")
	_ = os.Setenv("SERVICE_ACCOUNT", "SERVICE_ACCOUNT")
	_ = os.Setenv("KEYCLOAK_NAMESPACE", "KEYCLOAK_NAMESPACE")
	_ = os.Setenv("KEYCLOAK_SECRET", "KEYCLOAK_SECRET")
	_ = os.Setenv("ADM_SERVER", "ADM_SERVER")
	_ = os.Setenv("RELEASE_SERVICE_BASE", "RELEASE_SERVICE_BASE")
	_ = os.Setenv("INITIAL_SLEEP_INTERVAL", "11")
	_ = os.Setenv("MAX_WAIT_TIME", "22")
	_ = os.Setenv("NUMBER_WORKER_THREADS", "33")

	conf, err := config.InitConfig()
	s.NoError(err)

	s.Equal("RS_ROOT_URL", conf.ReleaseServiceRootURL)
	s.Equal("RS_PROXY_ROOT_URL", conf.ReleaseServiceProxyRootURL)
	s.Equal("MANIFEST_PATH", conf.ManifestPath)
	s.Equal("MANIFEST_TAG", conf.ManifestTag)
	s.Equal("REGISTRY_HOST_EXTERNAL", conf.HarborServerExternal)
	s.Equal("CATALOG_SERVER", conf.CatalogServer)
	s.Equal("HARBOR_SERVER", conf.HarborServer)
	s.Equal("HARBOR_NAMESPACE", conf.HarborNamespace)
	s.Equal("HARBOR_ADMIN_CREDENTIAL", conf.HarborAdminCredential)
	s.Equal("KEYCLOAK_SERVICE_BASE", conf.KeycloakServiceBase)
	s.Equal("KEYCLOAK_SERVER", conf.KeycloakServer)
	s.Equal("VAULT_SERVER", conf.VaultServer)
	s.Equal("SERVICE_ACCOUNT", conf.ServiceAccount)
	s.Equal("KEYCLOAK_NAMESPACE", conf.KeycloakNamespace)
	s.Equal("KEYCLOAK_SECRET", conf.KeycloakSecret)
	s.Equal("ADM_SERVER", conf.AdmServer)
	s.Equal("RELEASE_SERVICE_BASE", conf.ReleaseServiceBase)
	s.Equal(11*time.Second, conf.InitialSleepInterval)
	s.Equal(22*time.Second, conf.MaxWaitTime)
	s.Equal(33, conf.NumberWorkerThreads)
}

func (s *ManagerTestSuite) TestBadInterval() {
	s.clearEnvironment()
	_ = os.Setenv("INITIAL_SLEEP_INTERVAL", "I am not a number!")
	_ = os.Setenv("MAX_WAIT_TIME", "100")
	_ = os.Setenv("NUMBER_WORKER_THREADS", "2")

	_, err := config.InitConfig()
	s.Error(err)
	s.Contains(err.Error(), "invalid syntax")
}

func (s *ManagerTestSuite) TestBadMaxWait() {
	s.clearEnvironment()
	_ = os.Setenv("MAX_WAIT_TIME", "I am a free man!")
	_ = os.Setenv("INITIAL_SLEEP_INTERVAL", "100")
	_ = os.Setenv("NUMBER_WORKER_THREADS", "2")

	_, err := config.InitConfig()

	s.Error(err)
	s.Contains(err.Error(), "invalid syntax")
}

func (s *ManagerTestSuite) TestIntervalLargerThanWait() {
	s.clearEnvironment()
	_ = os.Setenv("MAX_WAIT_TIME", "100")
	_ = os.Setenv("INITIAL_SLEEP_INTERVAL", "1000")
	_ = os.Setenv("NUMBER_WORKER_THREADS", "2")

	_, err := config.InitConfig()
	s.Error(err)
	s.Contains(err.Error(), "must be less than")
}
