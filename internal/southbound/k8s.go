// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type K8sClient struct {
	clientset     *kubernetes.Clientset
	config        *rest.Config
	secretsClient coreV1Types.SecretInterface
	namespace     string
}

func newK8sClient(namespace string) (*K8sClient, error) {
	k8s := &K8sClient{}
	k8s.namespace = namespace
	err := k8s.initClient()
	if err != nil {
		return nil, err
	}
	return k8s, nil
}

func NewK8sClient(namespace string) (*K8sClient, error) {
	return newK8sClient(namespace)
}

func (k *K8sClient) getK8sClient() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	k.config = config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	k.clientset = clientset
	return nil
}

func (k *K8sClient) initClient() error {
	err := k.getK8sClient()
	if err != nil {
		return err
	}
	k.secretsClient = k.clientset.CoreV1().Secrets(k.namespace)
	return nil
}

func (k *K8sClient) ReadSecret(ctx context.Context, name string) (map[string][]byte, error) {
	secret, err := k.secretsClient.Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}
