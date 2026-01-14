// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Utility package
package southbound

import (
	"context"
	"github.com/open-edge-platform/app-orch-tenant-controller/internal/config"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"google.golang.org/grpc/metadata"
)

func getCtxForProjectID(ctx context.Context, projectUUID string, config config.Configuration) (context.Context, error) {
	vaultAuthClient, err := auth.NewVaultAuth(config.KeycloakServiceBase, config.VaultServer, config.ServiceAccount)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	token, err := vaultAuthClient.GetM2MToken(ctx)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return ctx, nil
	}

	outCtx := metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+token,
		"ActiveProjectID", projectUUID,
	)
	err = vaultAuthClient.Logout(ctx)
	return outCtx, err
}
