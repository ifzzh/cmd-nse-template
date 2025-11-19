package nat

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

// natClient 实现 NetworkServiceClient 接口
// 用于在客户端链中配置 NAT 接口
type natClient struct {
	vppConn api.Connection
}

// NewClient 创建新的 NAT 客户端
// 参数:
//   - vppConn: VPP 连接
//
// 返回:
//   - networkservice.NetworkServiceClient: NAT 客户端实例
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	return &natClient{
		vppConn: vppConn,
	}
}

// Request 处理客户端请求
// 功能说明:
//   1. 调用链中下一个客户端处理请求（创建 VPP 接口）
//   2. 获取接口索引
//   3. 确定接口角色（client 端为 outside）
//   4. 配置 NAT 接口
//
// 参数:
//   - ctx: 上下文
//   - request: 网络服务请求
//   - opts: gRPC 调用选项
//
// 返回:
//   - *networkservice.Connection: 建立的连接
//   - error: 错误信息
func (n *natClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("nat_client", "request")

	// 创建延迟清理上下文，用于失败时清理资源
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	// 1. 调用链中的下一个客户端（创建 VPP 接口）
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
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

	// 3. 确定接口角色（客户端链始终配置为 outside）
	role := NATRoleOutside
	logger.Infof("客户端链接口角色: %s, swIfIndex=%d", role, swIfIndex)

	// 4. 配置 NAT 接口
	if err := configureNATInterface(ctx, n.vppConn, swIfIndex, role); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if _, closeErr := n.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "连接关闭时发生错误: %s", closeErr.Error())
		}
		return nil, err
	}

	return conn, nil
}

// Close 关闭连接并清理资源
// 功能说明:
//   1. 获取接口索引
//   2. 禁用 NAT 接口
//   3. 调用链中的下一个客户端关闭连接
//
// 参数:
//   - ctx: 上下文
//   - conn: 要关闭的连接
//   - opts: gRPC 调用选项
//
// 返回:
//   - *empty.Empty: 空响应
//   - error: 错误信息
func (n *natClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("nat_client", "close")

	// 1. 获取接口索引
	isClient := metadata.IsClient(n)
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if ok {
		// 2. 禁用 NAT 接口
		role := NATRoleOutside
		if err := disableNATInterface(ctx, n.vppConn, swIfIndex, role); err != nil {
			logger.Warnf("禁用 NAT 接口失败: %v (swIfIndex=%d, role=%s)", err, swIfIndex, role)
		} else {
			logger.Infof("禁用 NAT 接口成功: swIfIndex=%d, role=%s", swIfIndex, role)
		}
	}

	// 3. 调用链中的下一个客户端关闭连接
	return next.Client(ctx).Close(ctx, conn, opts...)
}
