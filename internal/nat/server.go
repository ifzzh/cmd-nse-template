// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package nat 提供 NAT44（网络地址转换）功能的链式元素
// 本模块基于 VPP NAT44 插件实现，用于网络服务端点的地址转换
package nat

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// natServer NAT 服务器结构体
// 负责管理 VPP NAT44 规则的创建、应用和删除
//
// 字段说明:
//   - vppConn: VPP API 连接，用于与 VPP 交互
type natServer struct {
	vppConn api.Connection // VPP API 连接
}

// NewServer 创建 NAT NetworkServiceServer 链式元素
//
// 功能说明:
//   - 创建一个 NAT 服务器，用于在 VPP 接口上应用 NAT44 规则
//   - 作为 NSM 链式处理的一个环节，接收请求并传递给下一个处理器
//
// 参数:
//   - vppConn: VPP API 连接
//
// 返回:
//   - networkservice.NetworkServiceServer: NSM 网络服务服务器接口实现
//
// 使用示例:
//   natServer := nat.NewServer(vppConn)
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &natServer{
		vppConn: vppConn,
	}
}

// Request 处理网络服务请求
//
// 功能说明:
//   1. 调用链中下一个服务器处理请求（当前为空实现）
//
// 参数:
//   - ctx: 上下文
//   - request: 网络服务请求
//
// 返回:
//   - *networkservice.Connection: 建立的连接
//   - error: 错误信息
func (n *natServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// 当前为空实现，仅调用下一个服务器
	return next.Server(ctx).Request(ctx, request)
}

// Close 关闭连接
//
// 功能说明:
//   1. 调用链中的下一个服务器继续关闭流程（当前为空实现）
//
// 参数:
//   - ctx: 上下文
//   - conn: 要关闭的连接
//
// 返回:
//   - *empty.Empty: 空响应
//   - error: 错误信息
func (n *natServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// 当前为空实现，仅调用下一个服务器
	return next.Server(ctx).Close(ctx, conn)
}
