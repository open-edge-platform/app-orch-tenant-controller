# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2@sha256:d9db32125db0c3a680cfb7a1afcaefb89c898a075ec148fdc2f0f646cc2ed509 AS build

RUN mkdir /build
WORKDIR /build

COPY go.mod go.sum ./
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./vendor ./vendor
ARG TARGETPLATFORM

RUN if [ "${TARGETPLATFORM}" = "linux/amd64" ] ; then \
        CGO_ENABLED=0 go build -mod=vendor -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -trimpath -o provisioner ./cmd/provisioner ; \
    else  \
        CGO_ENABLED=0 go build -mod=vendor -trimpath -o provisioner ./cmd/provisioner ; \
    fi

FROM gcr.io/distroless/static:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc
USER nonroot

COPY --from=build /build/provisioner /usr/local/bin/provisioner

USER nobody

ENTRYPOINT ["/usr/local/bin/provisioner"]
