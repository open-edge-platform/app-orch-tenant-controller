# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.1@sha256:595c7847cff97c9a9e76f015083c481d26078f961c9c8dca3923132f51fe12f1 AS build

RUN mkdir /build
WORKDIR /build

COPY go.mod go.sum ./
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./vendor ./vendor
ARG TARGETPLATFORM

RUN if [ "${TARGETPLATFORM}" = "linux/amd64" ] ; then \
        CGO_ENABLED=0 go build -mod=vendor -gcflags="all=-spectre=all -N -l" -asmflags="-spectre=all" -trimpath -o provisioner ./cmd/provisioner ; \
    else  \
        CGO_ENABLED=0 go build -mod=vendor -trimpath -o provisioner ./cmd/provisioner ; \
    fi

FROM gcr.io/distroless/static:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39
USER nonroot

COPY --from=build /build/provisioner /usr/local/bin/provisioner

USER nobody

ENTRYPOINT ["/usr/local/bin/provisioner"]
