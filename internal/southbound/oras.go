// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
	"os"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

type Oras struct {
	dest     string
	registry string
}

func NewOras(registry string) (Oras, error) {
	o := Oras{}
	dest, err := os.MkdirTemp("", "repo")
	if err != nil {
		return o, err
	}
	o = Oras{
		dest: dest,
	}
	o.registry = registry
	return o, nil
}

func (o *Oras) Load(manifestPath string, manifestTag string) error {
	var err error

	o.dest, err = os.MkdirTemp("", "repo")
	if err != nil {
		return err
	}

	fs, err := file.New(o.dest)
	if err != nil {
		return err
	}
	defer fs.Close()

	ctx := context.Background()
	orasPath := o.registry + manifestPath
	log.Infof("ORAS request base URL %s", orasPath)

	repo, err := remote.NewRepository(orasPath)
	if err != nil {
		return err
	}
	repo.PlainHTTP = true

	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
	}

	tag := manifestTag
	_, err = oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return err
	}
	return nil
}

func (o *Oras) Dest() string {
	return o.dest
}

func (o *Oras) Close() {
	_ = os.RemoveAll(o.dest)
	o.dest = ""
}
