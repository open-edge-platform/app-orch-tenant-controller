// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Internal package
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
	"google.golang.org/protobuf/types/known/emptypb"
)

type AdmClient interface {
	ListDeployments(ctx context.Context, in *adm.ListDeploymentsRequest, opts ...grpc.CallOption) (*adm.ListDeploymentsResponse, error)
	CreateDeployment(ctx context.Context, in *adm.CreateDeploymentRequest, opts ...grpc.CallOption) (*adm.CreateDeploymentResponse, error)
	DeleteDeployment(ctx context.Context, in *adm.DeleteDeploymentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
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

func (a *AppDeployment) ListDeploymentNames(ctx context.Context, projectID string) (map[string]string, error) {
	ctx, err := getCtxForProjectID(ctx, projectID, a.configuration)
	if err != nil {
		return nil, err
	}
	admResp, err := a.admClient.ListDeployments(ctx, &adm.ListDeploymentsRequest{})
	if err != nil {
		return nil, err
	}

	existingDeployments := admResp.GetDeployments()
	existingDisplayNames := make(map[string]string)
	for _, dep := range existingDeployments {
		log.Infof("displayName : %s", dep.DisplayName)
		existingDisplayNames[dep.DisplayName] = dep.DisplayName
	}

	log.Infof("display name list size : %d", len(existingDisplayNames))
	return existingDisplayNames, nil
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

func (a *AppDeployment) DeleteDeployment(ctx context.Context,
	dpName string, displayName string, version string, profileName string,
	projectID string, missingOkay bool) error {
	log.Infof("ADM Delete Deployment DP name:%s display name:%s version:%s profileName:%s project ID:%s", dpName, displayName, version, profileName, projectID)

	lctx, err := getCtxForProjectID(ctx, projectID, a.configuration)
	if err != nil {
		return err
	}

	listDeploymentsRequest := &adm.ListDeploymentsRequest{
		// TODO: add Filter
	}
	resp, err := a.admClient.ListDeployments(lctx, listDeploymentsRequest)
	if err != nil {
		return err
	}
	deplID := ""
	for _, dep := range resp.GetDeployments() {
		if dep.DisplayName == displayName && dep.AppName == dpName && dep.AppVersion == version && dep.ProfileName == profileName {
			deplID = dep.DeployId
			log.Infof("Found deployment %s with ID %s", displayName, deplID)
			break
		}
	}
	if deplID == "" {
		if missingOkay {
			log.Infof("Deployment %s not found, skipping deletion", displayName)
			return nil
		}
		return status.Errorf(codes.NotFound, "Deployment %s not found", displayName)
	}

	deleteDeploymentRequest := &adm.DeleteDeploymentRequest{
		DeplId:     deplID,
		DeleteType: adm.DeleteType_PARENT_ONLY,
	}

	_, err = a.admClient.DeleteDeployment(lctx, deleteDeploymentRequest)
	if err != nil {
		return err
	}
	log.Info("ADM Deleted Deployment")
	return nil
}
