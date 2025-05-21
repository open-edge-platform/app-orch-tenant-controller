// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	"fmt"
	adm "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"testing"
	"time"
)

// Suite of catalog southbound tests
type AppDeploymentTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *AppDeploymentTestSuite) SetupSuite() {
}

func (s *AppDeploymentTestSuite) TearDownSuite() {
}

func (s *AppDeploymentTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 1*time.Minute)
	admClientFactory = NewTestAdmClient
	K8sFactory = NewTestK8s
	mockClient := MockCatalogClient{}
	_ = mockClient
	deployments = make(map[string]*adm.Deployment)
}

func (s *AppDeploymentTestSuite) TearDownTest() {
	s.cancel()
}

func TestAppDeployment(t *testing.T) {
	suite.Run(t, &AppDeploymentTestSuite{})
}

type testAdmClient struct {
}

var deployments map[string]*adm.Deployment

func (c *testAdmClient) ListDeployments(_ context.Context, _ *adm.ListDeploymentsRequest, _ ...grpc.CallOption) (*adm.ListDeploymentsResponse, error) {
	var resp adm.ListDeploymentsResponse
	for _, deployment := range deployments {
		resp.Deployments = append(resp.Deployments, deployment)
	}
	return &resp, nil
}

func (c *testAdmClient) CreateDeployment(_ context.Context, in *adm.CreateDeploymentRequest, _ ...grpc.CallOption) (*adm.CreateDeploymentResponse, error) {
	deployments[in.Deployment.Name] = in.Deployment

	resp := adm.CreateDeploymentResponse{}
	resp.DeploymentId = in.Deployment.Name
	return &resp, nil
}

func NewTestAdmClient(_ string) (AdmClient, error) {
	testClient := &testAdmClient{}
	return testClient, nil
}

func (s *AppDeploymentTestSuite) TestClientCreation() {
	c, err := NewAppDeploymentGRPCClient("http://localhost:1234")
	s.NoError(err)
	s.NotNil(c)
}

func (s *AppDeploymentTestSuite) TestAppDeployment() {
	var err error
	ADM, err := newADM(config.Configuration{AdmServer: ""})
	s.NoError(err)

	_, err = ADM.ListDeploymentNames(s.ctx, "")
	s.NoError(err)

	labels1 := map[string]string{
		"l1": "l1",
	}
	err = ADM.CreateDeployment(s.ctx, "deployment1", "Deployment 1", "1.1.1",
		"profile", "uuid", labels1)
	s.NoError(err)

	s.Len(deployments, 1)
}

func NewAdmClientWithError(_ string) (AdmClient, error) {
	return nil, fmt.Errorf("no client here")
}

func (s *AppDeploymentTestSuite) TestNewAppDeploymentError() {
	var err error
	savedFactory := admClientFactory
	admClientFactory = NewAdmClientWithError
	defer func() { admClientFactory = savedFactory }()
	adm, err := newADM(config.Configuration{AdmServer: "abc.def"})
	s.Error(err)
	s.Nil(adm)
}
