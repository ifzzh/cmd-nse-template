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

// Package acl 提供 ACL（访问控制列表）防火墙功能的链式元素
// 本模块基于 networkservicemesh/sdk-vpp/pkg/networkservice/acl 实现
// 并针对本地需求进行了定制化修改
package acl

import (
	"context"
	"fmt"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/govpp/binapi/acl"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

// aclServer ACL 服务器结构体
// 负责管理 VPP ACL 规则的创建、应用和删除
//
// 字段说明:
//   - vppConn: VPP API 连接，用于与 VPP 交互
//   - aclRules: 预配置的 ACL 规则列表（从配置文件加载）
//   - aclIndices: 连接 ID 到 ACL 索引的映射（线程安全）
type aclServer struct {
	vppConn    api.Connection                      // VPP API 连接
	aclRules   []acl_types.ACLRule                 // 预配置的 ACL 规则列表
	aclIndices genericsync.Map[string, []uint32]   // 连接 ID -> ACL 索引映射（线程安全）
}

// NewServer 创建 ACL NetworkServiceServer 链式元素
//
// 功能说明:
//   - 创建一个 ACL 服务器，用于在 VPP 接口上应用 ACL 规则
//   - 作为 NSM 链式处理的一个环节，接收请求并��递给下一个处理器
//
// 参数:
//   - vppConn: VPP API 连接
//   - aclrules: 要应用的 ACL 规则列表（通常从配置文件加载）
//
// 返回:
//   - networkservice.NetworkServiceServer: NSM 网络服务服务器接口实现
//
// 使用示例:
//   aclServer := acl.NewServer(vppConn, config.ACLConfig)
func NewServer(vppConn api.Connection, aclrules []acl_types.ACLRule) networkservice.NetworkServiceServer {
	return &aclServer{
		vppConn:  vppConn,
		aclRules: aclrules,
	}
}

// Request 处理网络服务请求
//
// 功能说明:
//   1. 调用链中下一个服务器处理请求
//   2. 检查此连接是否已应用 ACL 规则
//   3. 如果未应用且配置了 ACL 规则，则创建并应用规则
//   4. 如果创建失败，自动清理连接并返回错误
//
// 处理流程:
//   Request → next.Server().Request() → 检查 ACL → 创建/跳过 → 返回连接
//
// 参数:
//   - ctx: 上下文
//   - request: 网络服务请求
//
// 返回:
//   - *networkservice.Connection: 建立的连接
//   - error: 错误信息
func (a *aclServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// 创建延迟清理上下文，用于失败时清理资源
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	// 调用链中的��一个服务器
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// 检查是否已为此连接创建 ACL
	_, loaded := a.aclIndices.Load(conn.GetId())
	if !loaded && len(a.aclRules) > 0 {
		// 创建 ACL 规则并应用到 VPP 接口
		var indices []uint32
		if indices, err = create(ctx, a.vppConn, fmt.Sprintf("%s-%s", aclTag, conn.GetId()), metadata.IsClient(a), a.aclRules); err != nil {
			// 创建失败时，使用延迟上下文清理连接
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()

			if _, closeErr := a.Close(closeCtx, conn); closeErr != nil {
				// 包装错误信息，同时报告 ACL 创建失败和连接关闭失败
				err = errors.Wrapf(err, "连接关闭时发生错误: %s", closeErr.Error())
			}

			return nil, err
		}

		// 存储 ACL 索引，用于后续清理
		a.aclIndices.Store(conn.GetId(), indices)
	}

	return conn, nil
}

// Close 关闭连接并清理 ACL 规则
//
// 功能说明:
//   1. 从映射中加载并删除此连接的 ACL 索引
//   2. 调用 VPP API 删除每个 ACL 规则
//   3. 调用链中的下一个服务器继续关闭流程
//
// 处理流程:
//   Close → 加载 ACL 索引 → 删除 VPP ACL → next.Server().Close()
//
// 参数:
//   - ctx: 上下文
//   - conn: 要关闭的连接
//
// 返回:
//   - *empty.Empty: 空响应
//   - error: 错误信息
func (a *aclServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// 加载并删除此连接的 ACL 索引
	indices, _ := a.aclIndices.LoadAndDelete(conn.GetId())

	// 删除 VPP 中的每个 ACL 规则
	for ind := range indices {
		_, err := acl.NewServiceClient(a.vppConn).ACLDel(ctx, &acl.ACLDel{ACLIndex: uint32(ind)})
		if err != nil {
			// 删除失败只记录调试日志，不中断关闭流程
			log.FromContext(ctx).Debug("ACL 服务器: 删除 ACL 规则失败")
		}
	}

	// 调用链中的下一个服务器
	return next.Server(ctx).Close(ctx, conn)
}
