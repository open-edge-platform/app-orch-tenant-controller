// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	catalogv3 "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"testing"
	"time"
)

// Suite of catalog southbound tests
type CatalogTestSuite struct {
	suite.Suite
	ctx           context.Context
	cancel        context.CancelFunc
	configuration config.Configuration
}

func (s *CatalogTestSuite) SetupSuite() {
}

func (s *CatalogTestSuite) TearDownSuite() {
}

func (s *CatalogTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 1*time.Minute)
	catalogClientFactory = NewTestCatalogClient
	K8sFactory = NewTestK8s
	mockClient := MockCatalogClient{}
	_ = mockClient
	s.configuration = config.Configuration{}
}

func (s *CatalogTestSuite) TearDownTest() {
	s.cancel()
}

func TestCatalog(t *testing.T) {
	suite.Run(t, &CatalogTestSuite{})
}

type MockCatalogClient struct {
	GetRegistry           func(ctx context.Context, in *catalogv3.GetRegistryRequest, opts ...grpc.CallOption) (*catalogv3.GetRegistryResponse, error)
	CreateRegistry        func(ctx context.Context, in *catalogv3.CreateRegistryRequest, opts ...grpc.CallOption) (*catalogv3.CreateRegistryResponse, error)
	UpdateRegistry        func(ctx context.Context, in *catalogv3.UpdateRegistryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	UploadCatalogEntities func(ctx context.Context, in *catalogv3.UploadCatalogEntitiesRequest, opts ...grpc.CallOption) (*catalogv3.UploadCatalogEntitiesResponse, error)
	ListRegistries        func(ctx context.Context, in *catalogv3.ListRegistriesRequest, opts ...grpc.CallOption) (*catalogv3.ListRegistriesResponse, error)
}

func (t *MockCatalogClient) WithGetRegistry(getRegistryHandler func(ctx context.Context, in *catalogv3.GetRegistryRequest, opts ...grpc.CallOption) (*catalogv3.GetRegistryResponse, error)) *MockCatalogClient {
	t.GetRegistry = getRegistryHandler
	return t
}

func (t *MockCatalogClient) WithCreateRegistry(createRegistryHandler func(ctx context.Context, in *catalogv3.CreateRegistryRequest, opts ...grpc.CallOption) (*catalogv3.CreateRegistryResponse, error)) *MockCatalogClient {
	t.CreateRegistry = createRegistryHandler
	return t
}

func (t *MockCatalogClient) WithUpdateRegistry(updateRegistryHandler func(ctx context.Context, in *catalogv3.UpdateRegistryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)) *MockCatalogClient {
	t.UpdateRegistry = updateRegistryHandler
	return t
}

func (t *MockCatalogClient) WithUploadCatalogEntities(uploadCatalogEntitiesHandler func(ctx context.Context, in *catalogv3.UploadCatalogEntitiesRequest, opts ...grpc.CallOption) (*catalogv3.UploadCatalogEntitiesResponse, error)) *MockCatalogClient {
	t.UploadCatalogEntities = uploadCatalogEntitiesHandler
	return t
}

func (t *MockCatalogClient) WithListRegistries(listRegistriesHandler func(ctx context.Context, in *catalogv3.ListRegistriesRequest, opts ...grpc.CallOption) (*catalogv3.ListRegistriesResponse, error)) *MockCatalogClient {
	t.ListRegistries = listRegistriesHandler
	return t
}

type testCatalogClient struct {
}

var registries = map[string]*catalogv3.Registry{}

func (c *testCatalogClient) GetRegistry(_ context.Context, in *catalogv3.GetRegistryRequest, _ ...grpc.CallOption) (*catalogv3.GetRegistryResponse, error) {
	reg := registries[in.RegistryName]
	if reg == nil {
		err := status.Errorf(codes.NotFound, "registry %s not found", in.RegistryName)
		return nil, err
	}
	return &catalogv3.GetRegistryResponse{Registry: reg}, nil
}

func (c *testCatalogClient) CreateRegistry(_ context.Context, in *catalogv3.CreateRegistryRequest, _ ...grpc.CallOption) (*catalogv3.CreateRegistryResponse, error) {
	registries[in.Registry.Name] = in.Registry
	return &catalogv3.CreateRegistryResponse{}, nil
}

func (c *testCatalogClient) UpdateRegistry(_ context.Context, in *catalogv3.UpdateRegistryRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	registries[in.Registry.Name] = in.Registry
	return nil, nil
}

var uploads = map[string]*catalogv3.UploadCatalogEntitiesRequest{}

func (c *testCatalogClient) UploadCatalogEntities(_ context.Context, in *catalogv3.UploadCatalogEntitiesRequest, _ ...grpc.CallOption) (*catalogv3.UploadCatalogEntitiesResponse, error) {
	uploads[in.Upload.FileName] = in
	return &catalogv3.UploadCatalogEntitiesResponse{}, nil
}

func (c *testCatalogClient) ListRegistries(_ context.Context, _ *catalogv3.ListRegistriesRequest, _ ...grpc.CallOption) (*catalogv3.ListRegistriesResponse, error) {
	return nil, nil
}

func NewTestCatalogClient(_ string) (CatalogClient, error) {
	testClient := &testCatalogClient{}
	return testClient, nil
}

func (s *CatalogTestSuite) TestRegistryCreation() {
	var err error
	cat, err := newCatalog(s.configuration)
	s.NoError(err)

	// make new registry
	err = cat.CreateOrUpdateRegistry(s.ctx, RegistryAttributes{Name: "r", RootURL: "https://root1"})
	s.NoError(err)

	// check it
	s.Len(registries, 1)
	s.Equal("r", registries["r"].Name)
	s.Equal("https://root1", registries["r"].RootUrl)

	// update registry
	err = cat.CreateOrUpdateRegistry(s.ctx, RegistryAttributes{Name: "r", RootURL: "https://root2"})
	s.NoError(err)

	// check it
	s.Len(registries, 1)
	s.Equal("r", registries["r"].Name)
	s.Equal("https://root2", registries["r"].RootUrl)
}

func (s *CatalogTestSuite) TestRegistryList() {
	var err error
	cat, err := newCatalog(s.configuration)
	s.NoError(err)

	err = cat.ListRegistries(s.ctx)
	s.NoError(err)
}

func (s *CatalogTestSuite) TestYAMLUpload() {
	var err error
	cat, err := newCatalog(s.configuration)
	s.NoError(err)

	artifact := []byte("abc")
	err = cat.UploadYAMLFile(s.ctx, "project", "file-name1", artifact, false)
	s.NoError(err)
	s.Len(uploads, 1)

	err = cat.UploadYAMLFile(s.ctx, "project", "file-name2", artifact, true)
	s.NoError(err)
	s.Len(uploads, 2)

	s.Equal(false, uploads["file-name1"].LastUpload)
	s.Equal(true, uploads["file-name2"].LastUpload)
}

func (s *CatalogTestSuite) TestSecret() {
	var err error
	cat, err := newCatalog(s.configuration)
	s.NoError(err)

	secret, err := cat.InitializeClientSecret(s.ctx)
	s.NoError(err)
	s.Equal("", secret)
}
