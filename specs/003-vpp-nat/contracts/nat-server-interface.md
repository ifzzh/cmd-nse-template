# NAT Server Interface Contract

**文件**: `internal/nat/server.go`
**包**: `github.com/ifzzh/cmd-nse-template/internal/nat`

## 接口定义

### natServer 结构体

```go
// natServer 实现 NAT 网络服务端点
// 符合 networkservice.NetworkServiceServer 接口
type natServer struct {
    vppConn      api.Connection  // VPP API 连接
    publicIPPool []net.IP        // NAT 公网 IP 地址池
    natIndices   sync.Map        // 连接ID -> natConfig 的映射表
}
```

**职责**:
- 管理 VPP NAT 配置的生命周期
- 配置 VPP 接口为 NAT inside/outside 角色
- 配置 NAT 公网 IP 地址池
- 清理 NAT 配置和会话

## 公共方法

### NewServer

```go
// NewServer 创建一个新的 NAT 服务器
//
// 参数:
//   - ctx: 上下文（用于日志记录）
//   - vppConn: VPP API 连接
//   - publicIPPool: NAT 公网 IP 地址池（从配置加载）
//
// 返回:
//   - networkservice.NetworkServiceServer: NAT 服务器实例
//
// 约束:
//   - vppConn 不能为 nil
//   - publicIPPool 不能为空（至少包含一个 IP）
//
// 错误:
//   - 如果 publicIPPool 为空，记录警告日志但不返回错误
//     （P1 阶段可硬编码默认 IP，P2 阶段从配置加载）
func NewServer(ctx context.Context, vppConn api.Connection, publicIPPool []net.IP) networkservice.NetworkServiceServer
```

**设计决策**:
- 返回 `networkservice.NetworkServiceServer` 接口而非具体类型，便于链式调用
- 接受 `publicIPPool` 参数，支持 P2 配置管理
- 使用 `ctx` 参数传递日志上下文

### Request

```go
// Request 处理 NSC 连接请求，配置 VPP NAT
//
// 执行流程:
//  1. 调用链中的下一个 server（创建 VPP 接口）
//  2. 从上下文获取接口索引 (swIfIndex)
//  3. 检查是否已为该连接配置 NAT（通过 natIndices 查询）
//  4. 如果未配置，调用 configureNATInterface() 配置接口角色
//  5. 调用 configureNATAddressPool() 配置地址池
//  6. 存储配置到 natIndices
//  7. 返回连接
//
// 参数:
//   - ctx: 上下文（包含接口索引等元数据）
//   - request: NSM 连接请求
//
// 返回:
//   - *networkservice.Connection: 连接对象
//   - error: 错误信息
//
// 错误场景:
//   - 接口索引未找到（ifindex.Load 返回 false）
//   - VPP API 调用失败（Nat44InterfaceAddDelFeature 或 Nat44AddDelAddressRange）
//   - 下一个 server 的 Request() 失败
//
// 错误处理:
//   - 任何错误发生时，调用 Close() 清理已创建的连接
//   - 使用 postpone.ContextWithValues 创建延迟清理上下文
//
// 幂等性:
//   - 重复调用不会重复配置 NAT（通过 natIndices 检查）
//   - 如果已配置，直接返回连接
//
// 日志:
//   - Info: "配置 NAT 接口" + swIfIndex + 接口角色
//   - Info: "配置 NAT 地址池" + IP 范围
//   - Error: 错误详情（包含 swIfIndex、连接 ID）
func (n *natServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error)
```

**设计决策**:
- 遵循 NSM server chain 模式，先调用 `next.Server(ctx).Request()`
- 使用 `metadata.IsClient(n)` 判断接口角色（server 端 = inside）
- 错误时必须清理资源，避免 VPP 配置泄漏
- 幂等设计，支持 NSM 的重试机制

### Close

```go
// Close 处理 NSC 断开连接请求，清理 VPP NAT 配置
//
// 执行流程:
//  1. 从 natIndices 查找配置
//  2. 如果找到配置，调用 disableNATInterface() 禁用接口 NAT
//  3. 调用 removeNATAddressPool() 删除地址池
//  4. 从 natIndices 删除配置记录
//  5. 调用链中的下一个 server 的 Close()（删除 VPP 接口）
//
// 参数:
//   - ctx: 上下文
//   - conn: 连接对象
//
// 返回:
//   - *networkservice.Connection: 空连接对象
//   - error: 错误信息
//
// 错误场景:
//   - VPP API 调用失败（Nat44InterfaceAddDelFeature 或 Nat44AddDelAddressRange）
//   - 下一个 server 的 Close() 失败
//
// 错误处理:
//   - VPP API 调用失败时，记录错误日志但继续执行后续清理
//   - 最终返回下一个 server 的 Close() 的结果
//
// 健壮性:
//   - 如果 natIndices 中找不到配置，直接调用下一个 Close()，不报错
//   - 部分清理失败不阻塞完整清理流程
//
// 日志:
//   - Info: "清理 NAT 配置" + 连接 ID
//   - Warn: VPP API 清理失败（但继续执行）
//   - Error: 下一个 server 的 Close() 失败
func (n *natServer) Close(ctx context.Context, conn *networkservice.Connection) (*networkservice.Connection, error)
```

**设计决策**:
- 采用"最大努力清理"策略，部分失败不阻塞完整流程
- 先清理 NAT 配置，再调用下一个 Close()（与 Request 顺序相反）
- 即使找不到配置记录，也要调用下一个 Close()，确保链式清理完整

## 私有方法契约

### configureNATInterface

```go
// configureNATInterface 配置 VPP 接口为 NAT inside 或 outside
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - swIfIndex: VPP 软件接口索引
//   - isInside: true = inside（内部网络侧），false = outside（外部网络侧）
//
// 返回:
//   - error: VPP API 调用失败时返回错误
//
// VPP API: Nat44InterfaceAddDelFeature
//
// 实现细节:
//   - IsAdd = true（启用 NAT）
//   - Flags = NAT_IS_INSIDE (32) 或 NAT_IS_OUTSIDE (16)
//   - SwIfIndex = 从参数传入
//
// 约束:
//   - swIfIndex 必须是有效的接口索引（非零）
//   - 同一接口不能同时是 inside 和 outside
//
// 错误:
//   - VPP API 调用失败（返回非零 retval）
//   - 使用 errors.Wrap 包装错误，提供上下文
func configureNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex uint32, isInside bool) error
```

### disableNATInterface

```go
// disableNATInterface 禁用 VPP 接口的 NAT 功能
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - swIfIndex: VPP 软件接口索引
//   - isInside: true = inside，false = outside（需要与配置时一致）
//
// 返回:
//   - error: VPP API 调用失败时返回错误
//
// VPP API: Nat44InterfaceAddDelFeature
//
// 实现细节:
//   - IsAdd = false（禁用 NAT）
//   - Flags = NAT_IS_INSIDE (32) 或 NAT_IS_OUTSIDE (16)（与配置时一致）
//   - SwIfIndex = 从参数传入
//
// 错误:
//   - VPP API 调用失败（返回非零 retval）
//   - 使用 errors.Wrap 包装错误，提供上下文
func disableNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex uint32, isInside bool) error
```

### configureNATAddressPool

```go
// configureNATAddressPool 配置 NAT 公网 IP 地址池
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - publicIP: 公网 IP 地址（单个 IP）
//
// 返回:
//   - error: VPP API 调用失败时返回错误
//
// VPP API: Nat44AddDelAddressRange
//
// 实现细节:
//   - FirstIPAddress = publicIP
//   - LastIPAddress = publicIP（单个 IP 时起始和结束相同）
//   - VrfID = 0（默认 VRF）
//   - IsAdd = true（添加地址池）
//   - Flags = 0（无特殊标志）
//
// 约束:
//   - publicIP 必须是合法的 IPv4 地址
//   - 同一个 IP 不能重复配置
//
// 错误:
//   - VPP API 调用失败（返回非零 retval）
//   - 使用 errors.Wrap 包装错误，提供上下文
func configureNATAddressPool(ctx context.Context, vppConn api.Connection, publicIP net.IP) error
```

### removeNATAddressPool

```go
// removeNATAddressPool 删除 NAT 公网 IP 地址池
//
// 参数:
//   - ctx: 上下文
//   - vppConn: VPP API 连接
//   - publicIP: 公网 IP 地址（需要与配置时一致）
//
// 返回:
//   - error: VPP API 调用失败时返回错误
//
// VPP API: Nat44AddDelAddressRange
//
// 实现细节:
//   - FirstIPAddress = publicIP
//   - LastIPAddress = publicIP
//   - VrfID = 0（默认 VRF）
//   - IsAdd = false（删除地址池）
//   - Flags = 0（无特殊标志）
//
// 错误:
//   - VPP API 调用失败（返回非零 retval）
//   - 使用 errors.Wrap 包装错误，提供上下文
func removeNATAddressPool(ctx context.Context, vppConn api.Connection, publicIP net.IP) error
```

## 依赖项

### 外部依赖

```go
import (
    "context"
    "net"
    "sync"
    "time"

    "github.com/networkservicemesh/api/pkg/api/networkservice"
    "github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
    "github.com/networkservicemesh/sdk/pkg/tools/log"
    "github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
    "github.com/pkg/errors"
    "go.fd.io/govpp/api"

    // P3 阶段本地化后的导入路径
    "github.com/ifzzh/cmd-nse-template/internal/binapi_nat_types"
    "github.com/ifzzh/cmd-nse-template/internal/binapi_nat44_ed"
)
```

### 内部依赖

- `internal/binapi_nat_types/`: NAT 类型定义（P3.1 本地化）
- `internal/binapi_nat44_ed/`: NAT44 ED API（P3.2 本地化）

## 契约保证

### 线程安全性

- `natIndices sync.Map` 保证并发读写安全
- `Request()` 和 `Close()` 方法可并发调用
- VPP API 调用通过 govpp 的连接池管理并发

### 资源清理保证

- `Request()` 失败时，保证调用 `Close()` 清理资源
- `Close()` 失败时，记录日志但不抛出错误
- NAT 会话由 VPP 自动清理（超时机制）

### 幂等性保证

- 重复调用 `Request()` 不会重复配置 NAT
- `Close()` 可以安全调用多次（找不到配置时跳过）

### 错误传播

- 所有错误使用 `errors.Wrap` 包装，提供清晰的上下文
- 错误日志使用简体中文，包含关键参数（swIfIndex、连接 ID）

## 测试契约

### 单元测试覆盖

- `NewServer()`: 验证参数校验和初始化逻辑
- `Request()`: Mock VPP API，验证配置流程
- `Close()`: Mock VPP API，验证清理流程
- `configureNATInterface()`: 验证 VPP API 参数正确性
- `configureNATAddressPool()`: 验证 IP 地址转换正确性

### 集成测试覆盖

- 部署 NAT NSE 到 Kubernetes
- NSC 连接 → 验证 VPP 接口配置（`show nat44 interfaces`）
- NSC 断开 → 验证 VPP 配置清理
- 并发多个 NSC 连接 → 验证并发安全性

### VPP CLI 验证

```bash
# 检查接口配置
vpp# show nat44 interfaces
# 预期输出: Interface A = inside, Interface B = outside

# 检查地址池
vpp# show nat44 addresses
# 预期输出: 公网 IP 地址列表

# 检查会话
vpp# show nat44 sessions
# 预期输出: 内部IP:端口 → 公网IP:端口 映射
```

## 版本兼容性

- **Go 版本**: ≥1.23.8
- **VPP 版本**: ≥24.10.0（包含 NAT44 ED 插件）
- **govpp 版本**: v0.0.0-20240328101142-8a444680fbba
- **NSM SDK 版本**: ≥v1.15.0-rc.1

## 变更日志

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| v1.0.1 | 2025-01-13 | P1.1 - 创建空框架 |
| v1.0.2 | 2025-01-13 | P1.2 - 实现接口角色配置 |
| v1.0.3 | 2025-01-13 | P1.3 - 实现地址池配置 |
| v1.1.0 | 2025-01-13 | P3.1 - 本地化 nat_types |
| v1.1.1 | 2025-01-13 | P3.2 - 本地化 nat44_ed |
