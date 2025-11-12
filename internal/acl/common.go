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

package acl

import (
	"context"
	"time"

	"github.com/networkservicemesh/govpp/binapi/acl"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	// aclTag ACL 标签，用于标识从配置文件加载的 ACL 规则
	aclTag = "nsm-acl-from-config"
)

// create 在 VPP 接口上创建并应用 ACL 规则
//
// 功能说明:
//   1. 获取软件接口索引 (swIfIndex)
//   2. 创建入站 (ingress) ACL 规则
//   3. 创建出站 (egress) ACL 规则
//   4. 将 ACL 规则列表应用到 VPP 接口
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - tag: ACL 标签（用于标识）
//   - isClient: 是否为客户端模式
//   - aRules: ACL 规则列表
//
// 返回:
//   - []uint32: 创建的 ACL 索引列表
//   - error: 错误信息
func create(ctx context.Context, vppConn api.Connection, tag string, isClient bool, aRules []acl_types.ACLRule) ([]uint32, error) {
	logger := log.FromContext(ctx).WithField("acl_server", "create")

	// 获取软件接口索引
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return nil, errors.New("未找到软件接口索引 (swIfIndex)")
	}
	logger.Debugf("软件接口索引 swIfIndex=%v", swIfIndex)

	// 构造 ACL 接口列表
	interfaceACLList := &acl.ACLInterfaceSetACLList{
		SwIfIndex: swIfIndex,
	}

	// 添加入站 (ingress) ACL 规则
	var err error
	interfaceACLList.Acls, err = addACLToACLList(ctx, vppConn, tag, false, aRules)
	if err != nil {
		logger.Debug("添加入站 ACL 规则到列表失败")
		return nil, err
	}
	interfaceACLList.NInput = uint8(len(interfaceACLList.Acls))

	// 添加出站 (egress) ACL 规则
	egressACLIndeces, err := addACLToACLList(ctx, vppConn, tag, true, aRules)
	if err != nil {
		logger.Debug("添加出站 ACL 规则到列表失败")
		return nil, err
	}
	interfaceACLList.Acls = append(interfaceACLList.Acls, egressACLIndeces...)
	interfaceACLList.Count = uint8(len(interfaceACLList.Acls))

	// 将 ACL 列表应用到 VPP 接口
	_, err = acl.NewServiceClient(vppConn).ACLInterfaceSetACLList(ctx, interfaceACLList)
	if err != nil {
		return nil, errors.Wrap(err, "VPP API ACLInterfaceSetACLList 调用失败")
	}
	return interfaceACLList.Acls, nil
}

// addACLToACLList 添加 ACL 规则到 ACL 列表
//
// 功能说明:
//   - 调用 VPP API ACLAddReplace 创建 ACL 规则
//   - 记录操作耗时和索引
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - tag: ACL 标签
//   - egress: true 表示出站规则，false 表示入站规则
//   - aRules: ACL 规则列表
//
// 返回:
//   - []uint32: ACL 索引列表
//   - error: 错误信息
func addACLToACLList(ctx context.Context, vppConn api.Connection, tag string, egress bool, aRules []acl_types.ACLRule) ([]uint32, error) {
	var ACLIndeces []uint32

	now := time.Now()
	rsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, aclAdd(tag, egress, aRules))
	if err != nil {
		return nil, errors.Wrap(err, "VPP API ACLAddReplace 调用失败")
	}
	log.FromContext(ctx).
		WithField("aclIndices", rsp.ACLIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLAddReplace").Debug("ACL 规则创建完成")
	ACLIndeces = append([]uint32{rsp.ACLIndex}, ACLIndeces...)

	return ACLIndeces, nil
}

// aclAdd 构造 ACL 添加/替换请求
//
// 功能说明:
//   - 复制 ACL 规则列表，避免修改原始数据
//   - 对于出站规则，交换源/目标 IP 和端口，实现双向过滤
//
// 技术细节:
//   VPP ACL 规则默认是入站方向，出站规则需要反转匹配条件：
//   - 交换源/目标 IP 前缀 (SrcPrefix ↔ DstPrefix)
//   - 交换源/目标端口 (SrcportOrIcmptypeFirst ↔ DstportOrIcmpcodeFirst)
//   - 交换源/目标端口范围 (SrcportOrIcmptypeLast ↔ DstportOrIcmpcodeLast)
//
// 参数:
//   - tag: ACL 标签
//   - egress: true 表示出站规则，false 表示入站规则
//   - aRules: ACL 规则列表
//
// 返回:
//   - *acl.ACLAddReplace: ACL 添加/替换请求
func aclAdd(tag string, egress bool, aRules []acl_types.ACLRule) *acl.ACLAddReplace {
	// 复制规则列表，避免修改原始数据
	aRulesCopy := make([]acl_types.ACLRule, len(aRules))
	copy(aRulesCopy, aRules)

	aclAddReplace := &acl.ACLAddReplace{
		ACLIndex: ^uint32(0), // 0xFFFFFFFF 表示创建新 ACL
		Tag:      tag,
		Count:    uint32(len(aRulesCopy)),
		R:        aRulesCopy,
	}

	// 对于出站规则，交换源/目标信息
	if egress {
		for i := range aclAddReplace.R {
			// 交换源/目标 IP 前缀
			aclAddReplace.R[i].SrcPrefix, aclAddReplace.R[i].DstPrefix =
				aclAddReplace.R[i].DstPrefix, aclAddReplace.R[i].SrcPrefix

			// 交换源/目标端口（或 ICMP 类型/代码）
			aclAddReplace.R[i].SrcportOrIcmptypeFirst, aclAddReplace.R[i].DstportOrIcmpcodeFirst =
				aclAddReplace.R[i].DstportOrIcmpcodeFirst, aclAddReplace.R[i].SrcportOrIcmptypeFirst

			aclAddReplace.R[i].SrcportOrIcmptypeLast, aclAddReplace.R[i].DstportOrIcmpcodeLast =
				aclAddReplace.R[i].DstportOrIcmpcodeLast, aclAddReplace.R[i].SrcportOrIcmptypeLast
		}
	}
	return aclAddReplace
}
