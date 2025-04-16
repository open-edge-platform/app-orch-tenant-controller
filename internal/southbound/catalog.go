// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	catalogv3 "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	"github.com/open-edge-platform/app-orch-catalog/pkg/wiper"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/open-edge-platform/orch-library/go/pkg/grpc/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var log = dazl.GetPackageLogger()

type CatalogClient interface {
	GetRegistry(ctx context.Context, in *catalogv3.GetRegistryRequest, opts ...grpc.CallOption) (*catalogv3.GetRegistryResponse, error)
	CreateRegistry(ctx context.Context, in *catalogv3.CreateRegistryRequest, opts ...grpc.CallOption) (*catalogv3.CreateRegistryResponse, error)
	UpdateRegistry(ctx context.Context, in *catalogv3.UpdateRegistryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	UploadCatalogEntities(ctx context.Context, in *catalogv3.UploadCatalogEntitiesRequest, opts ...grpc.CallOption) (*catalogv3.UploadCatalogEntitiesResponse, error)
	ListRegistries(ctx context.Context, in *catalogv3.ListRegistriesRequest, opts ...grpc.CallOption) (*catalogv3.ListRegistriesResponse, error)
}

type AppCatalog struct {
	config        config.Configuration
	catalogClient CatalogClient
	sessionID     string
}

var catalogClientFactory = NewCatalogClient

func NewCatalogGRPCClient(catalogGrpcHost string) (catalogv3.CatalogServiceClient, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))

	conn, err := grpc.NewClient(catalogGrpcHost, opts...)
	if err != nil {
		return nil, err
	}
	return catalogv3.NewCatalogServiceClient(conn), nil
}

func NewCatalogClient(catalogGrpcHost string) (CatalogClient, error) {
	return NewCatalogGRPCClient(catalogGrpcHost)
}

func newCatalog(config config.Configuration) (*AppCatalog, error) {
	cat := &AppCatalog{
		config: config,
	}
	var err error
	cat.catalogClient, err = catalogClientFactory(cat.config.CatalogServer)
	if err != nil {
		return nil, err
	}
	return cat, nil
}

func (c *AppCatalog) InitializeClientSecret(ctx context.Context) (string, error) {
	log.Infof("Initializing client secret")
	v, err := auth.NewVaultAuth(c.config.KeycloakServiceBase, c.config.VaultServer, c.config.ServiceAccount)
	if err != nil {
		return "", err
	}

	k8sClient, err := K8sFactory(c.config.KeycloakNamespace)
	if err != nil {
		return "", err
	}

	data, err := k8sClient.ReadSecret(ctx, c.config.KeycloakSecret)
	if err != nil {
		return "", err
	}

	p := string(data["admin-password"])
	u := "admin"
	m2m, err := v.GetM2MToken(ctx)
	if err != nil || m2m == "" {
		log.Infof("Client secret not found, creating a new one")
		return v.CreateClientSecret(ctx, u, p)
	}
	log.Infof("Client secret found")
	return m2m, nil
}

func NewAppCatalog(config config.Configuration) (*AppCatalog, error) {
	return newCatalog(config)
}

type RegistryAttributes struct {
	Name         string
	DisplayName  string
	Description  string
	Type         string
	RootURL      string
	InventoryURL string
	Username     string
	Cacerts      string
	AuthToken    string
	ProjectUUID  string
}

func (c *AppCatalog) CreateOrUpdateRegistry(ctx context.Context, attrs RegistryAttributes) error {
	log.Infof("Creating or updating registry %s url %s", attrs.Name, attrs.RootURL)
	ctx, err := getCtxForProjectID(ctx, attrs.ProjectUUID, c.config)
	if err != nil {
		return err
	}

	registry := &catalogv3.Registry{
		Name:         attrs.Name,
		DisplayName:  attrs.DisplayName,
		Description:  attrs.Description,
		Type:         attrs.Type,
		RootUrl:      attrs.RootURL,
		InventoryUrl: attrs.InventoryURL,
		Username:     attrs.Username,
		Cacerts:      attrs.Cacerts,
		AuthToken:    attrs.AuthToken,
	}

	if _, err = c.catalogClient.GetRegistry(ctx, &catalogv3.GetRegistryRequest{RegistryName: attrs.Name}); err != nil {
		if !errors.IsNotFound(errors.FromGRPC(err)) {
			return err
		}
		if _, err = c.catalogClient.CreateRegistry(ctx, &catalogv3.CreateRegistryRequest{Registry: registry}); err != nil {
			return err
		}
		log.Infof("Registry %s created", attrs.Name)
	} else {
		if _, err = c.catalogClient.UpdateRegistry(ctx, &catalogv3.UpdateRegistryRequest{RegistryName: registry.Name, Registry: registry}); err != nil {
			return err
		}
		log.Infof("Registry %s updated", attrs.Name)
	}
	return nil
}

func (c *AppCatalog) ListRegistries(ctx context.Context) error {
	ctx, err := getCtxForProjectID(ctx, "", c.config)
	if err != nil {
		return err
	}
	_, err = c.catalogClient.ListRegistries(ctx, &catalogv3.ListRegistriesRequest{})
	return err
}

func (c *AppCatalog) UploadYAMLFile(ctx context.Context, projectUUID string, fileName string, artifact []byte, lastFile bool) error {
	log.Debugf("Uploading file %s to %s last file %t", fileName, projectUUID, lastFile)
	ctx, err := getCtxForProjectID(ctx, projectUUID, c.config)
	if err != nil {
		return err
	}
	fileUpload := &catalogv3.Upload{
		FileName: fileName,
		Artifact: artifact,
	}
	catalogUpload := &catalogv3.UploadCatalogEntitiesRequest{
		SessionId:  c.sessionID,
		Upload:     fileUpload,
		LastUpload: lastFile,
	}
	resp, err := c.catalogClient.UploadCatalogEntities(ctx, catalogUpload)
	if err != nil {
		return err
	}
	c.sessionID = resp.SessionId
	return nil
}

func (c *AppCatalog) WipeProject(ctx context.Context, projectUUID string, catalogServer string) error {
	log.Infof("Wiping project %s", projectUUID)
	ctx, err := getCtxForProjectID(ctx, projectUUID, c.config)
	if err != nil {
		return err
	}
	gc, err := NewCatalogGRPCClient(catalogServer)

	if err != nil {
		return err
	}
	grpcWiper := wiper.NewGRPCWiper(gc)
	errs := grpcWiper.Wipe(ctx, projectUUID)
	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}
