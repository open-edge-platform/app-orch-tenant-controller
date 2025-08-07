// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/southbound"
	"oras.land/oras-go/v2/content/file"
)

const (
	HarborProjectID = 1234 // Mock project ID for testing
)

// Catalog client mock
type upload struct {
	path       string
	artifact   string
	lastUpload bool
}

type testCatalog struct {
	registries    map[string]southbound.RegistryAttributes
	uploadedFiles map[string]upload
}

var mockCatalog testCatalog

func newTestCatalog(_ config.Configuration) (Catalog, error) {
	if mockCatalog.registries == nil {
		mockCatalog = testCatalog{
			registries:    map[string]southbound.RegistryAttributes{},
			uploadedFiles: map[string]upload{},
		}
	}
	return &mockCatalog, nil
}

func (c *testCatalog) CreateOrUpdateRegistry(_ context.Context, attrs southbound.RegistryAttributes) error {
	c.registries[attrs.Name] = attrs
	return nil
}

func (c *testCatalog) ListRegistries(_ context.Context) error {
	return nil
}

func (c *testCatalog) UploadYAMLFile(_ context.Context, _ string, filePath string, artifact []byte, lastFile bool) error {
	_, fileName := filepath.Split(filePath)
	uploadedFile := upload{
		path:       fileName,
		artifact:   string(artifact),
		lastUpload: lastFile,
	}
	c.uploadedFiles[fileName] = uploadedFile
	return nil
}

func (c *testCatalog) InitializeClientSecret(_ context.Context) (string, error) {
	return "", nil
}

func (c *testCatalog) ListPublishers(_ context.Context) error { return nil }

func (c *testCatalog) WipeProject(_ context.Context, _ string, _ string) error {
	return nil
}

// Oras client mock
type testOras struct {
	dest string
}

func NewTestOras(_ string) (Oras, error) {
	return &testOras{}, nil
}

var paths = map[string]string{
	"/registry/edge-node/en/manifest:latest":       "24.11.0.yaml",
	"/registry/edge-node/dp/intel-gpu:1.0.2":       "intel-gpu_1.0.2.yaml",
	"/registry/edge-node/dp/loadbalancer:0.1.0":    "loadbalancer_0.1.0.yaml",
	"/registry/edge-node/dp/sriov:0.1.4":           "sriov_0.1.4.yaml",
	"/registry/edge-node/dp/usb:0.1.0":             "usb_0.1.0.yaml",
	"/registry/edge-node/dp/virtualization:0.2.4":  "virtualization_0.2.4.yaml",
	"/registry/edge-node/tmpl/privileged:1.3.4":    "privileged_1.3.4.json",
	"/registry/edge-node/tmpl/restricted:1.3.4":    "restricted_1.3.4.json",
	"/registry/edge-node/tmpl/baseline:1.3.4":      "baseline_1.3.4.json",
	"/registry/edge-node/dp/base-extensions:0.2.0": "base-extensions_0.2.0.yaml",
	"/registry/edge-node/dp/loadbalancer:0.2.6":    "loadbalancer_0.2.6.yaml",
	"/registry/edge-node/dp/skupper:0.1.4":         "skupper_0.1.4.yaml",
}

func (o *testOras) Load(path string, version string) error {
	var err error
	o.dest, err = os.MkdirTemp("", "repo")
	if err != nil {
		return err
	}

	fs, err := file.New(o.dest)
	if err != nil {
		return err
	}
	defer func() { _ = fs.Close() }()

	fullPath := path + ":" + version

	srcFilePath := filepath.Join("testdata", "extensions", paths[fullPath])
	destFilePath := filepath.Join(o.dest, paths[fullPath])

	// Read in test data
	data, err := os.ReadFile(srcFilePath)
	if err != nil {
		return err
	}

	// Write data to destination
	err = os.WriteFile(destFilePath, data, 0600)
	return err
}

func (o *testOras) Close() {
	_ = os.RemoveAll(o.dest)
}

func (o *testOras) Dest() string {
	return o.dest
}

// Harbor client mock
type permission struct {
	roleID    int
	groupName string
	projectID string
}

type robot struct {
	projectName string
	robotName   string
	robotID     int
}

type testHarbor struct {
	configurations  int
	createdProjects map[string]string
	permissions     []permission
	robots          map[string]robot
}

var testHarborInstance *testHarbor

func NewTestHarbor(_ context.Context, _ string, _ string, _ string, _ string) (Harbor, error) {
	if testHarborInstance == nil {
		testHarborInstance = &testHarbor{
			configurations:  0,
			createdProjects: map[string]string{},
			permissions:     []permission{},
			robots:          map[string]robot{},
		}
	}
	return testHarborInstance, nil
}

func (t *testHarbor) Configurations(_ context.Context) error {
	t.configurations++
	return nil
}

func (t *testHarbor) CreateProject(_ context.Context, org string, displayName string) error {
	name := org + "-" + displayName
	t.createdProjects[name] = name
	return nil
}

func (t *testHarbor) SetMemberPermissions(_ context.Context, roleID int, _ string, displayName string, groupName string) error {
	t.permissions = append(t.permissions, permission{roleID: roleID, groupName: groupName, projectID: displayName})
	return nil
}

func (t *testHarbor) Ping(_ context.Context) error {
	return nil
}

func (t *testHarbor) GetProjectID(_ context.Context, _ string, _ string) (int, error) {
	return HarborProjectID, nil
}

var nextRobotID = 1

func (t *testHarbor) CreateRobot(_ context.Context, robotName string, org string, displayName string) (string, string, error) {
	// robot$catalog-apps-coke-proj1+catalog-apps-read-write
	robotName = fmt.Sprintf("robot$catalog-apps-%s-%s+%s", org, displayName, robotName)
	t.robots[robotName] = robot{
		projectName: displayName,
		robotName:   robotName,
		robotID:     nextRobotID,
	}
	nextRobotID++
	return "name", "secret", nil
}

func (t *testHarbor) GetRobot(_ context.Context, _ string, _ string, robotName string, projectID int) (*southbound.HarborRobot, error) {
	if projectID != HarborProjectID {
		return nil, fmt.Errorf("robot %s projectID %d not found", robotName, projectID)
	}
	r, ok := t.robots[robotName]
	if !ok {
		return nil, fmt.Errorf("robot %s not found", robotName)
	}
	return &southbound.HarborRobot{Name: r.robotName, ID: r.robotID}, nil
}

func (t *testHarbor) DeleteRobot(_ context.Context, robotID int) error {
	for _, r := range t.robots {
		if r.robotID == robotID {
			delete(t.robots, r.robotName)
			return nil
		}
	}
	return fmt.Errorf("delete robot %d not found", robotID)
}

func (t *testHarbor) DeleteProject(_ context.Context, org string, displayName string) error {
	delete(t.createdProjects, org+"-"+displayName)
	return nil
}

// ADM client mock
type testADM struct {
}

var mockADM *testADM

var mockDeployments = map[string]*mockDeployment{}

func newTestADM(_ config.Configuration) (AppDeployment, error) {
	if mockADM == nil {
		mockADM = &testADM{}
	}
	return mockADM, nil
}

func (t *testADM) ListDeploymentNames(_ context.Context, _ string) (map[string]string, error) {
	displayName := make(map[string]string)
	for _, md := range mockDeployments {
		if md.name != "" {
			displayName[md.name] = md.version  // Return version instead of name
		}
	}
	return displayName, nil
}

type mockDeployment struct {
	name        string
	version     string
	profileName string
	projectID   string
	labels      map[string]string
}

func (t *testADM) CreateDeployment(_ context.Context, name string, _ string, version string, profileName string, projectID string, labels map[string]string) error {
	md := &mockDeployment{
		name:        name,
		version:     version,
		profileName: profileName,
		projectID:   projectID,
		labels:      labels,
	}
	mdKey := fmt.Sprintf("%s-%s-%s", md.name, md.version, md.profileName)
	mockDeployments[mdKey] = md
	return nil
}

func (t *testADM) DeleteDeployment(_ context.Context, name string, _ string, version string, profileName string, projectID string, missingOkay bool) error {
	_ = projectID
	// TODO: implement project ID test
	mdKey := fmt.Sprintf("%s-%s-%s", name, version, profileName)
	if _, exists := mockDeployments[mdKey]; !exists {
		if !missingOkay {
			return fmt.Errorf("deployment %s not found", mdKey)
		}
		return nil
	}
	delete(mockDeployments, mdKey)
	return nil
}
