<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Application Orchestrator Tenant Controller

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

The Application Orchestrator Tenant Controller is a Kubernetes Deployment of a Go server that handles lifecycle
management for multi-tenacy **Projects** in the **[Application Catalog]**, **[Harbor]**, **[Cluster Template Manager]**
and **[App Deployment Manager]**.

## Get Started

### Workflow

When a multi-tenancy Project is created, the Tenant Controller performs these operations:

- in the Orchestrator Harbor
  - creates the `catalog-apps` Project in the Orchestrator Harbor for the project
  - creates members in this Harbor project
  - creates robot accounts in this Harbor project
- in the Application Catalog the following registries are created for the project:
  - `harbor-helm` registry to point at the Orchestrator Harbor for Helm Charts
  - `harbor-docker` registry to point at the Orchestrator Harbor for Images
  - `intel-rs-helm` registry to point at the Release Service OCI Registry for Helm Charts
  - `intel-rs-image` registry to point at the Release Service OCI Registry for Images
- in the Application Catalog apps and packages are created for extensions:
  - download from the Release Service the manifest of LPKE deployment packages
  - load them in to the Application Catalog one by one
- in the Cluster Template Manager cluster templates are created:
  - download from the Release Service the manifest of LPKE cluster templates
  - load them in to the Cluster Template Manager one by one
- in the Application Deployment Manager, deployments are created for extension packages:
  - download from the Release Service the manifest of LPKE deployments
  - for each deployment in the list, create a deployment in ADM

When a project is deleted, the tenant controller performs these operations:

- in the Orchestrator Harbor, the project specific `catalog-apps` project is deleted
- in the Application Catalog, all entities fot the project are deleted
- deletion of cluster templates is handled by the Cluster Template Manager
- deletion of deployments is handled by the App Deployment Manager

### Method of Operation

The tenant controller listens for Project `create` and `delete` events coming from the multi-tenancy data model and
dispatches these events to `plugins` that handle the application catalog, Harbor, cluster templates, and extensions
packages. The plugins utilize `southbound` implementations to communicate with the app catalog server, Harbor server,
CTM server, and ADM server.

### Input Variables

The Application Orchestrator Tenant Controller Deployment is loaded as a [Docker Image](../build/Dockerfile) and
referred to by a [Helm Chart](../deploy/charts/config-provisioner).

The values given to the helm chart drive the behavior. The values are:

- harborServer:
  - default `http://harbor-oci-core.orch-harbor.svc.cluster.local:80`
  - the internally accessible Orchestrator Harbor service URL
  - Env var: `HARBOR_SERVER, REGISTRY_HOST`
- catalogServer:
  - default `catalog-service-grpc-server.orch-app.svc.cluster.local:8080`
  - the internally accessible App Catalog REST proxy service URL
  - Env var: `CATALOG_SERVER`
- releaseServiceBase:
  - default `rs-proxy.rs-proxy.svc.cluster.local:8081`
  - the internally accessible Release Proxy service URL
  - Env var: `RELEASE_SERVICE_BASE`
- keycloakServiceBase:
  - default `"http://platform-keycloak.orch-platform.svc.cluster.local:8080"`
  - the internally accessible Keycloak service URL
  - Env var: `KEYCLOAK_SERVICE_BASE`
- admServer:
  - default `app-deployment-api-grpc-server.orch-app.svc.cluster.local:8080`
  - the internally accessible ADM service URL
  - Env var: `ADM_SERVER`
- keycloakSecret:
  - default `"platform-keycloak"`
  - the name of the kubernetes secret that holds keycloak credentials
  - Env var: `KEYCLOAK_SECRET`
- serviceAccount:
  - default `orch-svc`
  - the service account used for the deployment
  - Env var: `SERVICE_ACCOUNT`
- vaultServer:
  - default `"http://vault.orch-platform.svc.cluster.local:8200"`
  - the internally accessible vault service URL
  - Env var: `VAULT_SERVER`
- keycloakServer:
  - default - must be overridden with cluster specific FQDN
  - the externally accessible Keycloak service URL
  - Env var: `KEYCLOAK_SERVER`
- harborServerExternal:
  - default - must be overridden with cluster specific FQDN
  - the externally accessible Orchestrator Harbor service URL
  - Env var: `REGISTRY_HOST_EXTERNAL`
- releaseServiceRootUrl:
  - default - must be overridden with cluster specific FQDN
  - The externally accessible URL of the Release Service
  - Env var: `RS_ROOT_URL`
- releaseServiceProxyRootUrl:
  - default `oci://rs-proxy.rs-proxy.svc.cluster.local:8443`
  - The internally accessible URL of the Release Service Proxy
  - Env var: `RS_PROXY_ROOT_URL`
- manifestPath:
  - default `"/edge-orch/en/files/manifest"`
  - path to use when fetching the Release Server manifest
  - Env var: `MANIFEST_PATH`
- manifestTag:
  - default `latest`
  - version tag to use when fetching the Release Server manifest
  - Env var: `MANIFEST_TAG`
- namespace:
  - default `orch-system`
  - The namespace where the Application Orchestrator Tenant Controller resides
- keycloakNamespace:
  - default `orch-platform`
  - The namespace where the Keycloak service resides
  - Env var: `KEYCLOAK_NAMESPACE`
- harborNamespace:
  - default `orch-harbor`
  - The namespace where the Harbor service resides
  - Env var: `HARBOR_NAMESPACE`
- platformNamespace:
  - default `orch-platform`
  - The namespace where the Platform services reside
- numberWorkerThreads:
  - default `2`
  - defines the number of simultaneous workers that are available to process events
  - Env var: `NUMBER_WORKER_THREADS`
- initialSleepInterval:
  - default `60`
  - number of seconds to wait for an event to be processed
  - Env var: `INITIAL_SLEEP_INTERVAL`
- maxWaitTime:
  - default `600`
  - maximum number of seconds to wait for an event to be processed
  - Env var: `MAX_WAIT_TIME`

## Develop

To develop a new plugin add to the package `internal/plugins`. The plugin must implement the `Plugin` interface
in [plugin.go](internal/plugins/plugin.go) with the methods:

- Name() string
- Initialize(context.Context) error
- CreateEvent(context.Context, Event, PluginData) error
- DeleteEvent(context.Context, Event, PluginData) error

Each plugin must have its own set of unit tests in the `internal/plugins` package.

To add a new plugin to the controller, create a struct for your plugin and call the `plugins.Register()` function
in [manager.go](internal/manager/manager.go).

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed
and merged into the main. We encourage you to add appropriate unit tests and e2e tests if your contribution introduces
a new feature. See the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information.

Additionally, ensure the following commands are successful:

```shell
make test
make lint
make license
```

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community](https://github.com/open-edge-platform).
For support, start with [Troubleshooting](https://github.com/open-edge-platform) or [contact us](https://github.com/open-edge-platform).

## License

Application Orchestration Tenant Controller is licensed under Apache 2.0.

[Application Catalog]: https://github.com/open-edge-platform/app-orch-catalog
[App Deployment Manager]: https://github.com/open-edge-platform/app-orch-deployment/tree/main/app-deployment-manager
[Cluster Template Manager]: https://github.com/open-edge-platform/cluster-manager
[Harbor]: https://goharbor.io
