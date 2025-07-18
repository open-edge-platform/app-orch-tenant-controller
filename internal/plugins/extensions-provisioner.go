// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/southbound"
	yaml "gopkg.in/yaml.v2"
)

const (
	DesiredStatePresent = "present"
	DesiredStateAbsent  = "absent"
)

type Manifest struct {
	Metadata struct {
		SchemaVersion string `yaml:"schemaVersion"`
		Release       string `yaml:"release"`
	} `yaml:"metadata"`
	Lpke struct {
		DeploymentPackages []struct {
			Dpkg         string `yaml:"dpkg"`
			Version      string `yaml:"version"`
			DesiredState string `yaml:"desiredState"` // if unspecified, defaults to "present"
		} `yaml:"deploymentPackages"`
		DeploymentList []struct {
			DpName               string `yaml:"dpName"`
			DisplayName          string `yaml:"displayName"`
			DpProfileName        string `yaml:"dpProfileName"`
			DpVersion            string `yaml:"dpVersion"`
			AllAppTargetClusters []struct {
				Key string `yaml:"key"`
				Val string `yaml:"val"`
			} `yaml:"allAppTargetClusters"`
			DesiredState string `yaml:"desiredState"` // if unspecified, defaults to "present"
		} `yaml:"deploymentList"`
	} `yaml:"lpke"`
}

type AppDeployment interface {
	ListDeploymentNames(ctx context.Context, projectID string) (map[string]string, error)
	CreateDeployment(ctx context.Context, dpName string, displayName string, version string, profileName string, projectID string, labels map[string]string) error
	DeleteDeployment(ctx context.Context, dpName string, displayName string, version string, profileName string, projectID string, missingOkay bool) error
}

func NewAppDeployment(configuration config.Configuration) (AppDeployment, error) {
	return southbound.NewAppDeployment(configuration)
}

var AppDeploymentFactory = NewAppDeployment

type Oras interface {
	Load(string, string) error
	Dest() string
	Close()
}

var OrasFactory = NewOras

func NewOras(registry string) (Oras, error) {
	oras, err := southbound.NewOras(registry)
	if err != nil {
		return nil, err
	}
	return &oras, nil
}

type ExtensionsProvisionerPlugin struct {
	configuration config.Configuration
}

func NewExtensionsProvisionerPlugin(configuration config.Configuration) (*ExtensionsProvisionerPlugin, error) {
	plugin := &ExtensionsProvisionerPlugin{
		configuration: configuration,
	}
	return plugin, nil
}

func (p *ExtensionsProvisionerPlugin) waitForADM(ctx context.Context) {
	if p.configuration.AdmServer == "" {
		log.Info("No admServer is set, skipping wait")
		return
	}

	log.Infof("Waiting for app deployment manager %s", p.configuration.AdmServer)
	ad, _ := AppDeploymentFactory(p.configuration)

	for {
		var cancel context.CancelFunc
		var lctx context.Context
		lctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		_, err := ad.ListDeploymentNames(lctx, "")
		cancel()
		if err == nil || strings.Contains(err.Error(), "Unauthenticated") {
			break
		}
		time.Sleep(5 * time.Second)
	}
	log.Info("App deployment manager ready")
}

func (p *ExtensionsProvisionerPlugin) Initialize(_ context.Context, _ PluginData) error {
	p.waitForADM(context.Background())
	return nil
}

func (p *ExtensionsProvisionerPlugin) CreateEvent(ctx context.Context, event Event, _ PluginData) error {
	var err error

	var yamlBytes []byte

	if p.configuration.UseLocalManifest != "" {
		log.Info("Using local manifest")
		yamlBytes = []byte(p.configuration.UseLocalManifest)
	} else {
		log.Infof("Using remote manifest directory %s%s:%s", p.configuration.ReleaseServiceBase, p.configuration.ManifestPath, p.configuration.ManifestTag)

		manifestOras, err := OrasFactory(p.configuration.ReleaseServiceBase)
		if err != nil {
			return err
		}
		defer manifestOras.Close()

		err = manifestOras.Load(p.configuration.ManifestPath, p.configuration.ManifestTag)
		if err != nil {
			return err
		}

		manifestDir := manifestOras.Dest()

		entries, err := os.ReadDir(manifestDir)
		if err != nil {
			return err
		}

		yamlBytes, err = os.ReadFile(manifestOras.Dest() + "/" + entries[0].Name())
		if err != nil {
			return err
		}
	}

	cat, err := CatalogFactory(p.configuration)
	if err != nil {
		return err
	}

	manifest := Manifest{}

	decoder := yaml.NewDecoder(strings.NewReader(string(yamlBytes)))
	err = decoder.Decode(&manifest)
	if err != nil {
		return err
	}

	log.Infof("Manifest release %s", manifest.Metadata.Release)
	pkgOras, err := OrasFactory(p.configuration.ReleaseServiceBase)
	if err != nil {
		return err
	}
	defer pkgOras.Close()
	for _, dp := range manifest.Lpke.DeploymentPackages {
		if dp.DesiredState == DesiredStateAbsent {
			// TODO: implement deletion of deployment packages. We need to do this _after_ the deployments are deleted.
			log.Infof("Skipping deployment package %s version %s as desiredState is %s", dp.Dpkg, dp.Version, dp.DesiredState)
			continue
		}

		err = pkgOras.Load(`/`+dp.Dpkg, dp.Version)
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(pkgOras.Dest())
		if err != nil {
			return err
		}
		for i, entry := range entries {
			var artifact []byte
			fileName := pkgOras.Dest() + "/" + entry.Name()
			artifact, err = os.ReadFile(fileName)
			if err != nil {
				return err
			}

			lastUpload := i == len(entries)-1
			err = cat.UploadYAMLFile(ctx, event.UUID, fileName, artifact, lastUpload)
			if err != nil {
				return err
			}
		}
	}

	if p.configuration.AdmServer == "" {
		log.Info("No admServer is set, skipping deployments")
	} else {
		uuid := event.UUID
		ad, _ := AppDeploymentFactory(p.configuration)

		existingDisplayNames, err := ad.ListDeploymentNames(ctx, uuid)
		if err != nil {
			log.Info("Not able to list deployments, skipping deployments")
			return err
		}

		for _, dl := range manifest.Lpke.DeploymentList {
			log.Infof("displayName: %s", dl.DisplayName)
			if dl.DesiredState == DesiredStateAbsent {
				err = ad.DeleteDeployment(ctx, dl.DpName, dl.DisplayName, dl.DpVersion, dl.DpProfileName, uuid, true)
				if err != nil {
					return err
				}
			} else {
				if _, exists := existingDisplayNames[dl.DisplayName]; exists {
					log.Infof("Deployment with displayName %s already exists, skipping creation", dl.DisplayName)
					continue
				}

				labels := map[string]string{}
				for _, appTargetCluster := range dl.AllAppTargetClusters {
					labels[appTargetCluster.Key] = appTargetCluster.Val
				}
				err = ad.CreateDeployment(ctx, dl.DpName, dl.DisplayName, dl.DpVersion, dl.DpProfileName, uuid, labels)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *ExtensionsProvisionerPlugin) DeleteEvent(_ context.Context, _ Event, _ PluginData) error {
	return nil
}

func (p *ExtensionsProvisionerPlugin) Name() string {
	return "Extensions Provisioner"
}
