# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.11@sha256:54528d189affe7ce60840cad093d53360705d47d71ea74d62eef72c476242fe9 AS build

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
