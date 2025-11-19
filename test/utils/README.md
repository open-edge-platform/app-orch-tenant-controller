<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Component Test Utilities

This directory contains utility packages that support component testing following the catalog repository pattern.

## Packages

- `portforward/`: Port forwarding utilities for connecting to deployed orchestrator services
- `auth/`: Authentication utilities for obtaining tokens from deployed Keycloak  
- `types/`: Common types and constants used across component tests

These utilities enable component tests to connect to and authenticate with deployed orchestrator services, following the same patterns used by the catalog repository.