// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Internal package
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/open-edge-platform/orch-library/go/dazl"
)

var log = dazl.GetPackageLogger()

// Configuration is a manager configuration
type Configuration struct {
	// service addresses. These are all addresses internal to the cluster

	// catalog service - gRPC
	CatalogServer string

	// harbor core - REST
	HarborServer string

	// harbor namespace name
	HarborNamespace string

	// harbor credential name
	HarborAdminCredential string

	// keycloak server for external use - REST
	KeycloakServer string

	// keycloak service - REST
	KeycloakServiceBase string

	// vault server - REST
	VaultServer string

	// Service account name
	ServiceAccount string

	// keycloak server namespace
	KeycloakNamespace string

	// keycloak server secret name
	KeycloakSecret string

	// app deployment manager - gRPC
	AdmServer string

	// release service proxy - HTTP
	ReleaseServiceBase string

	// release service configurations

	// harbor REST API external to cluster
	HarborServerExternal string

	// release service root URL - used for Docker registry on release service
	ReleaseServiceRootURL string

	// release service proxy root URL - used for Helm registry on release service
	ReleaseServiceProxyRootURL string

	// path to manifest repo
	ManifestPath string

	// tag to use in manifest repo
	ManifestTag string

	// on retry, initial delay
	InitialSleepInterval time.Duration

	// maximum wait on retry
	MaxWaitTime time.Duration

	// number of worker threads
	NumberWorkerThreads int

	// if this string is nonempty, provisioner will use a local manifest contianed in the string instead of using manifest from remote release service
	UseLocalManifest string
}

func DumpConfig(config Configuration) {
	log.Info("Creating Manager with config:")

	log.Infof("   manifestPath: %s", config.ManifestPath)
	log.Infof("   manifestTag: %s", config.ManifestTag)
	log.Infof("   releaseServiceRootURL: %s", config.ReleaseServiceRootURL)
	log.Infof("   releaseServiceProxyRootURL: %s", config.ReleaseServiceProxyRootURL)
	log.Infof("   harborServer: %s", config.HarborServer)
	log.Infof("   harborNamespce: %s", config.HarborNamespace)
	log.Infof("   harborAdminCredential: %s", config.HarborAdminCredential)
	log.Infof("   vaultServer: %s", config.VaultServer)
	log.Infof("   serviceAccount: %s", config.ServiceAccount)
	log.Infof("   harborServerExternal: %s", config.HarborServerExternal)
	log.Infof("   catalogServer: %s", config.CatalogServer)
	log.Infof("   keycloakServer: %s", config.KeycloakServer)
	log.Infof("   keycloakServiceBase: %s", config.KeycloakServiceBase)
	log.Infof("   keycloakNamespace: %s", config.KeycloakNamespace)
	log.Infof("   keycloakSecret: %s", config.KeycloakSecret)
	log.Infof("   admServer: %s", config.AdmServer)
	log.Infof("   releaseServiceBase: %s", config.ReleaseServiceBase)
	log.Infof("   initialSleepInterval: %s", config.InitialSleepInterval)
	log.Infof("   maxWaitTime: %s", config.MaxWaitTime)
	log.Infof("   numberWorkerThreads: %d", config.NumberWorkerThreads)
	log.Infof("   useLocalManifest: %s", config.UseLocalManifest)
}

func InitConfig() (Configuration, error) {
	config := Configuration{}
	config.ReleaseServiceRootURL = os.Getenv("RS_ROOT_URL")
	config.ReleaseServiceProxyRootURL = os.Getenv("RS_PROXY_ROOT_URL")
	config.ManifestPath = os.Getenv("MANIFEST_PATH")
	config.ManifestTag = os.Getenv("MANIFEST_TAG")
	config.HarborServerExternal = os.Getenv("REGISTRY_HOST_EXTERNAL")
	config.CatalogServer = os.Getenv("CATALOG_SERVER")
	config.HarborServer = os.Getenv("HARBOR_SERVER")
	config.HarborNamespace = os.Getenv("HARBOR_NAMESPACE")
	config.HarborAdminCredential = os.Getenv("HARBOR_ADMIN_CREDENTIAL")
	config.KeycloakServer = os.Getenv("KEYCLOAK_SERVER")
	config.KeycloakServiceBase = os.Getenv("KEYCLOAK_SERVICE_BASE")
	config.KeycloakNamespace = os.Getenv("KEYCLOAK_NAMESPACE")
	config.KeycloakSecret = os.Getenv("KEYCLOAK_SECRET")
	config.AdmServer = os.Getenv("ADM_SERVER")
	config.VaultServer = os.Getenv("VAULT_SERVER")
	config.ReleaseServiceBase = os.Getenv("RELEASE_SERVICE_BASE")
	config.ServiceAccount = os.Getenv("SERVICE_ACCOUNT")
	config.UseLocalManifest = os.Getenv("USE_LOCAL_MANIFEST")

	var err error

	initialSleepIntervalString := os.Getenv("INITIAL_SLEEP_INTERVAL")
	initialSleepInterval, err := strconv.Atoi(initialSleepIntervalString)
	if err != nil {
		log.Errorf("Invalid sleep interval %s", initialSleepIntervalString)
		return config, err
	}
	config.InitialSleepInterval = time.Duration(initialSleepInterval) * time.Second

	maxWaitTimeString := os.Getenv("MAX_WAIT_TIME")
	maxWaitTime, err := strconv.Atoi(maxWaitTimeString)
	if err != nil {
		log.Errorf("Invalid max wait string %s", maxWaitTimeString)
		return config, err
	}
	config.MaxWaitTime = time.Duration(maxWaitTime) * time.Second

	numberWorkerThreadsString := os.Getenv("NUMBER_WORKER_THREADS")
	numberWorkerThreads, err := strconv.Atoi(numberWorkerThreadsString)
	if err != nil {
		log.Errorf("Invalid number of worker threads string %s", numberWorkerThreadsString)
		return config, err
	}
	config.NumberWorkerThreads = numberWorkerThreads

	if config.InitialSleepInterval > config.MaxWaitTime {
		log.Errorf("Sleep interval %d must be less than max wait time %d", config.InitialSleepInterval, config.MaxWaitTime)
		return config, fmt.Errorf("invlaid sleep interval %d must be less than max wait time %d", config.InitialSleepInterval, config.MaxWaitTime)
	}
	return config, nil
}
