// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2024 OpenInfra Foundation Europe. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package internal

import (
	"context"
	"net/url"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryauthorize "github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"google.golang.org/grpc"
)

// NewRegistryClient 创建NSM Registry客户端
// 配置了客户端URL、拨号选项和授权策略
func NewRegistryClient(ctx context.Context, connectTo *url.URL, clientOptions []grpc.DialOption, registryPolicies []string) registryapi.NetworkServiceEndpointRegistryClient {
	return registryclient.NewNetworkServiceEndpointRegistryClient(
		ctx,
		registryclient.WithClientURL(connectTo),
		registryclient.WithDialOptions(clientOptions...),
		registryclient.WithNSEAdditionalFunctionality(
			clientinfo.NewNetworkServiceEndpointRegistryClient(),
			registrysendfd.NewNetworkServiceEndpointRegistryClient()),
		registryclient.WithAuthorizeNSERegistryClient(registryauthorize.NewNetworkServiceEndpointRegistryClient(
			registryauthorize.WithPolicies(registryPolicies...))),
	)
}

// RegisterEndpoint 向NSM Manager注册网络服务端点
// 构造NetworkServiceEndpoint对象并发起注册请求
func RegisterEndpoint(ctx context.Context, client registryapi.NetworkServiceEndpointRegistryClient, name string, serviceName string, labels map[string]string, listenURL string) (*registryapi.NetworkServiceEndpoint, error) {
	return client.Register(ctx, &registryapi.NetworkServiceEndpoint{
		Name:                name,
		NetworkServiceNames: []string{serviceName},
		NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
			serviceName: {
				Labels: labels,
			},
		},
		Url: listenURL,
	})
}
