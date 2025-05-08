// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	adm "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/orch-library/go/pkg/grpc/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type AdmClient interface {
	ListDeployments(ctx context.Context, in *adm.ListDeploymentsRequest, opts ...grpc.CallOption) (*adm.ListDeploymentsResponse, error)
	CreateDeployment(ctx context.Context, in *adm.CreateDeploymentRequest, opts ...grpc.CallOption) (*adm.CreateDeploymentResponse, error)
}

type AppDeployment struct {
	configuration config.Configuration
	admClient     AdmClient
}

var admClientFactory = NewAdmClient

func NewAppDeployment(configuration config.Configuration) (*AppDeployment, error) {
	return newADM(configuration)
}

func NewAppDeploymentGRPCClient(admGrpcHost string) (adm.DeploymentServiceClient, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))

	conn, err := grpc.NewClient(admGrpcHost, opts...)
	if err != nil {
		return nil, err
	}
	return adm.NewDeploymentServiceClient(conn), nil
}

func NewAdmClient(admGrpcHost string) (AdmClient, error) {
	return NewAppDeploymentGRPCClient(admGrpcHost)
}

func newADM(configuration config.Configuration) (*AppDeployment, error) {
	ad := &AppDeployment{
		configuration: configuration,
	}
	var err error
	ad.admClient, err = admClientFactory(configuration.AdmServer)
	if err != nil {
		return nil, err
	}
	return ad, nil
}

func (a *AppDeployment) ListDeployments(ctx context.Context) error {
	ctx, err := getCtxForProjectID(ctx, "", a.configuration)
	if err != nil {
		return err
	}
	_, err = a.admClient.ListDeployments(ctx, &adm.ListDeploymentsRequest{})
	return err
}

func (a *AppDeployment) CreateDeployment(ctx context.Context,
	dpName string, displayName string, version string, profileName string,
	projectID string, labels map[string]string) error {
	log.Infof("ADM Create Deployment DP name:%s display name:%s version:%s profileName:%s project ID:%s labels:%v", dpName, displayName, version, profileName, projectID, labels)
	deployment := &adm.CreateDeploymentRequest{
		Deployment: &adm.Deployment{
			DisplayName:    displayName,
			AppName:        dpName,
			AppVersion:     version,
			ProfileName:    profileName,
			DeploymentType: "auto-scaling",
			AllAppTargetClusters: &adm.TargetClusters{
				Labels: labels,
			},
		},
	}

	lctx, err := getCtxForProjectID(ctx, projectID, a.configuration)
	if err != nil {
		return err
	}
	resp, err := a.admClient.CreateDeployment(lctx, deployment)
	if e, ok := status.FromError(err); ok {
		if e.Code() == codes.AlreadyExists {
			return nil
		}
	}
	if err != nil {
		return err
	}
	log.Infof("ADM Created deployment %s", resp.DeploymentId)
	return nil
}
