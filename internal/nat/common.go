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

// Package nat 提供 NAT44 功能的通用工具和辅助函数
package nat

import (
	"context"
	"fmt"
	"net"

	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip_types"
	"github.com/networkservicemesh/govpp/binapi/nat44_ed"
	"github.com/networkservicemesh/govpp/binapi/nat_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// NATInterfaceRole NAT 接口角色类型
type NATInterfaceRole string

const (
	// NATRoleInside 内部接口（NSC 侧，源地址转换）
	NATRoleInside NATInterfaceRole = "inside"
	// NATRoleOutside 外部接口（下游 NSE 侧，转换后的地址）
	NATRoleOutside NATInterfaceRole = "outside"
)

// enableNAT44Plugin 启用 VPP NAT44 ED 插件
//
// 功能说明:
//   1. 调用 VPP API Nat44EdPluginEnableDisable 启用插件
//   2. 配置会话数等参数
//   3. 检查返回值，确认启用成功
//
// 注意事项:
//   - 必须在配置地址池和接口之前调用此函数
//   - VPP 21.01+ 版本中 NAT44 插件默认禁用，必须显式启用
//   - 如果插件已启用，某些 VPP 版本可能返回非零值（可忽略）
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//
// 返回:
//   - error: 错误信息
func enableNAT44Plugin(ctx context.Context, vppConn api.Connection) error {
	logger := log.FromContext(ctx).WithField("nat_server", "enable_plugin")

	// 创建请求
	req := &nat44_ed.Nat44EdPluginEnableDisable{
		Enable:   true,
		Sessions: 10000, // 默认最大会话数 10000，可根据需要调整
		// InsideVrf 和 OutsideVrf 默认为 0（默认 VRF）
	}

	// 调用 VPP API
	reply := &nat44_ed.Nat44EdPluginEnableDisableReply{}
	if err := vppConn.Invoke(ctx, req, reply); err != nil {
		logger.Errorf("VPP API 调用失败: %v", err)
		return errors.Wrap(err, "VPP API Nat44EdPluginEnableDisable 调用失败")
	}

	// 检查返回值
	if reply.Retval != 0 {
		logger.Errorf("VPP API 返回错误: retval=%d", reply.Retval)
		return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
	}

	logger.Infof("NAT44 ED 插件启用成功")
	return nil
}

// configureNATInterface 配置 NAT 接口角色（inside/outside）
//
// 功能说明:
//   1. 创建 NAT44 API 客户端
//   2. 根据角色确定 NAT 标志（NAT_IS_INSIDE 或 NAT_IS_OUTSIDE）
//   3. 调用 VPP API Nat44InterfaceAddDelFeature 启用 NAT 功能
//   4. 检查返回值，确认配置成功
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - swIfIndex: VPP 接口索引
//   - role: NAT 接口角色（inside/outside）
//
// 返回:
//   - error: 错误信息
func configureNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, role NATInterfaceRole) error {
	logger := log.FromContext(ctx).WithField("nat_server", "configure")

	// 确定 NAT 角色标志
	var flags nat_types.NatConfigFlags
	if role == NATRoleInside {
		flags = nat_types.NAT_IS_INSIDE
	} else {
		flags = nat_types.NAT_IS_OUTSIDE
	}

	// 创建请求
	req := &nat44_ed.Nat44InterfaceAddDelFeature{
		IsAdd:     true,
		Flags:     flags,
		SwIfIndex: swIfIndex,
	}

	// 调用 VPP API
	reply := &nat44_ed.Nat44InterfaceAddDelFeatureReply{}
	if err := vppConn.Invoke(ctx, req, reply); err != nil {
		logger.Errorf("VPP API 调用失败: %v", err)
		return errors.Wrap(err, "VPP API Nat44InterfaceAddDelFeature 调用失败")
	}

	// 检查返回值
	if reply.Retval != 0 {
		logger.Errorf("VPP API 返回错误: retval=%d", reply.Retval)
		return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
	}

	logger.Infof("配置 NAT 接口成功: swIfIndex=%d, role=%s", swIfIndex, role)
	return nil
}

// disableNATInterface 禁用 NAT 接口功能
//
// 功能说明:
//   1. 调用 VPP API Nat44InterfaceAddDelFeature 禁用 NAT 功能（IsAdd=false）
//   2. 用于资源清理，不阻断关闭流程
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - swIfIndex: VPP 接口索引
//   - role: NAT 接口角色（用于日志）
//
// 返回:
//   - error: 错误信息
func disableNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, role NATInterfaceRole) error {
	logger := log.FromContext(ctx).WithField("nat_server", "cleanup")

	// 确定 NAT 角色标志
	var flags nat_types.NatConfigFlags
	if role == NATRoleInside {
		flags = nat_types.NAT_IS_INSIDE
	} else {
		flags = nat_types.NAT_IS_OUTSIDE
	}

	// 创建请求（IsAdd=false 表示禁用）
	req := &nat44_ed.Nat44InterfaceAddDelFeature{
		IsAdd:     false,
		Flags:     flags,
		SwIfIndex: swIfIndex,
	}

	// 调用 VPP API
	reply := &nat44_ed.Nat44InterfaceAddDelFeatureReply{}
	if err := vppConn.Invoke(ctx, req, reply); err != nil {
		logger.Errorf("VPP API 调用失败: %v", err)
		return errors.Wrap(err, "VPP API Nat44InterfaceAddDelFeature 调用失败")
	}

	// 检查返回值
	if reply.Retval != 0 {
		logger.Errorf("VPP API 返回错误: retval=%d", reply.Retval)
		return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
	}

	logger.Infof("禁用 NAT 接口成功: swIfIndex=%d, role=%s", swIfIndex, role)
	return nil
}

// configureNATAddressPool 配置 NAT 地址池
//
// 功能说明:
//   1. 遍历公网 IP 列表
//   2. 将 net.IP 转换为 ip_types.IP4Address
//   3. 调用 VPP API Nat44AddDelAddressRange 添加地址池
//   4. 检查返回值，确认配置成功
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - publicIPs: 公网 IP 地址列表
//
// 返回:
//   - error: 错误信息
func configureNATAddressPool(ctx context.Context, vppConn api.Connection, publicIPs []net.IP) error {
	logger := log.FromContext(ctx).WithField("nat_server", "configure_pool")

	for _, publicIP := range publicIPs {
		// 1. 转换 net.IP 为 ip_types.IP4Address
		ip4 := publicIP.To4()
		if ip4 == nil {
			return fmt.Errorf("公网 IP 不是有效的 IPv4 地址: %s", publicIP)
		}

		var firstIP ip_types.IP4Address
		copy(firstIP[:], ip4)
		lastIP := firstIP // 单个 IP 时，起始和结束地址相同

		// 2. 创建请求
		req := &nat44_ed.Nat44AddDelAddressRange{
			FirstIPAddress: firstIP,
			LastIPAddress:  lastIP,
			VrfID:          0,    // 默认 VRF
			IsAdd:          true, // 添加地址池
			Flags:          0,    // 默认标志
		}

		// 3. 调用 VPP API
		reply := &nat44_ed.Nat44AddDelAddressRangeReply{}
		if err := vppConn.Invoke(ctx, req, reply); err != nil {
			logger.Errorf("VPP API 调用失败: publicIP=%s, error=%v", publicIP.String(), err)
			return errors.Wrapf(err, "VPP API Nat44AddDelAddressRange 调用失败（IP: %s）", publicIP)
		}

		// 4. 检查返回值
		if reply.Retval != 0 {
			logger.Errorf("VPP API 返回错误: publicIP=%s, retval=%d", publicIP.String(), reply.Retval)
			return fmt.Errorf("VPP API 返回错误: %d（IP: %s）", reply.Retval, publicIP)
		}

		logger.Infof("配置 NAT 地址池成功: %s", publicIP.String())
	}

	return nil
}

// cleanupNATAddressPool 删除 NAT 地址池
//
// 功能说明:
//   1. 调用 VPP API Nat44AddDelAddressRange 删除地址池（IsAdd=false）
//   2. 用于资源清理，不阻断关闭流程
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - publicIPs: 公网 IP 地址列表
//
// 返回:
//   - error: 错误信息
func cleanupNATAddressPool(ctx context.Context, vppConn api.Connection, publicIPs []net.IP) error {
	logger := log.FromContext(ctx).WithField("nat_server", "cleanup_pool")

	for _, publicIP := range publicIPs {
		ip4 := publicIP.To4()
		if ip4 == nil {
			continue
		}

		var firstIP ip_types.IP4Address
		copy(firstIP[:], ip4)
		lastIP := firstIP

		req := &nat44_ed.Nat44AddDelAddressRange{
			FirstIPAddress: firstIP,
			LastIPAddress:  lastIP,
			VrfID:          0,
			IsAdd:          false, // 删除地址池
			Flags:          0,
		}

		reply := &nat44_ed.Nat44AddDelAddressRangeReply{}
		if err := vppConn.Invoke(ctx, req, reply); err != nil {
			logger.Errorf("VPP API 调用失败: publicIP=%s, error=%v", publicIP.String(), err)
			// 继续删除其他地址，不中断
			continue
		}

		if reply.Retval != 0 {
			logger.Errorf("VPP API 返回错误: publicIP=%s, retval=%d", publicIP.String(), reply.Retval)
			continue
		}

		logger.Infof("删除 NAT 地址池成功: %s", publicIP.String())
	}

	return nil
}
