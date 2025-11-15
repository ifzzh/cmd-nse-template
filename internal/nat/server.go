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

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

// natInterfaceState NAT 接口状态记录
type natInterfaceState struct {
	swIfIndex  interface_types.InterfaceIndex // VPP 接口索引
	role       NATInterfaceRole                // NAT 角色（inside/outside）
	configured bool                            // 是否已配置
}

// natServer NAT 服务器结构体
// 负责管理 VPP NAT44 规则的创建、应用和删除
//
// 字段说明:
//   - vppConn: VPP API 连接，用于与 VPP 交互
//   - interfaceStates: 连接 ID 到接口状态的映射（线程安全）
type natServer struct {
	vppConn         api.Connection                               // VPP API 连接
	interfaceStates genericsync.Map[string, *natInterfaceState] // 连接 ID -> 接口状态映射
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
//   1. 调用链中下一个服务器处理请求（创建 VPP 接口）
//   2. 获取接口索引
//   3. 确定接口角色（client 端为 outside，server 端为 inside）
//   4. 配置 NAT 接口
//   5. 记录接口状态
//
// 参数:
//   - ctx: 上下文
//   - request: 网络服务请求
//
// 返回:
//   - *networkservice.Connection: 建立的连接
//   - error: 错误信息
func (n *natServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// 创建延迟清理上下文，用于失败时清理资源
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	// 1. 调用链中的下一个服务器（创建 VPP 接口）
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// 2. 获取接口索引
	isClient := metadata.IsClient(n)
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if _, closeErr := n.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "连接关闭时发生错误: %s", closeErr.Error())
		}
		return nil, errors.New("未找到接口索引")
	}

	// 3. 确定接口角色
	var role NATInterfaceRole
	if isClient {
		role = NATRoleOutside // client 端连接下游 NSE，配置为 outside
	} else {
		role = NATRoleInside // server 端连接 NSC，配置为 inside
	}

	// 4. 配置 NAT 接口
	if err := configureNATInterface(ctx, n.vppConn, swIfIndex, role); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if _, closeErr := n.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "连接关闭时发生错误: %s", closeErr.Error())
		}
		return nil, err
	}

	// 5. 记录接口状态
	n.interfaceStates.Store(conn.GetId(), &natInterfaceState{
		swIfIndex:  swIfIndex,
		role:       role,
		configured: true,
	})

	return conn, nil
}

// Close 关闭连接并清理 NAT 配置
//
// 功能说明:
//   1. 从映射中加载并删除此连接的接口状态
//   2. 禁用 NAT 接口功能
//   3. 调用链中的下一个服务器继续关闭流程
//
// 参数:
//   - ctx: 上下文
//   - conn: 要关闭的连接
//
// 返回:
//   - *empty.Empty: 空响应
//   - error: 错误信息
func (n *natServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// 1. 获取接口状态
	state, loaded := n.interfaceStates.LoadAndDelete(conn.GetId())
	if loaded && state.configured {
		// 2. 禁用 NAT 接口（IsAdd = false）
		if err := disableNATInterface(ctx, n.vppConn, state.swIfIndex, state.role); err != nil {
			// 禁用失败只记录错误，不阻断关闭流程
			// log.FromContext(ctx) 已在 disableNATInterface 中记录
		}
	}

	// 3. 调用下一个服务器
	return next.Server(ctx).Close(ctx, conn)
}
