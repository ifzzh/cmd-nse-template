# API: Nat44AddDelAddressRange

**阶段**: P1.3
**用途**: 配置 NAT 地址池（公网 IP 地址范围）
**VPP 版本**: VPP v23.10-rc0~170 / v24.10.0

## API 定义

### 请求结构

```go
type Nat44AddDelAddressRange struct {
    FirstIPAddress ip_types.IP4Address      // NAT 公网 IP 起始地址
    LastIPAddress  ip_types.IP4Address      // NAT 公网 IP 结束地址（单 IP 时与 FirstIPAddress 相同）
    VrfID          uint32                   // VRF ID（虚拟路由转发表 ID，默认 0）
    IsAdd          bool                     // true = 添加地址池, false = 删除地址池
    Flags          nat_types.NatConfigFlags // 标志位（默认 0，保留字段）
}
```

### 字段说明

| 字段名 | 类型 | 必填 | 描述 | 有效值 |
|--------|------|------|------|--------|
| FirstIPAddress | ip_types.IP4Address | ✅ | 公网 IP 起始地址 | 有效的 IPv4 地址（4 字节数组） |
| LastIPAddress | ip_types.IP4Address | ✅ | 公网 IP 结束地址 | 有效的 IPv4 地址，单 IP 时与 FirstIPAddress 相同 |
| VrfID | uint32 | ❌ | VRF ID | 默认 0（全局路由表） |
| IsAdd | bool | ✅ | 添加或删除地址池 | true（添加）、false（删除） |
| Flags | nat_types.NatConfigFlags | ❌ | 标志位 | 默认 0 |

### 响应结构

```go
type Nat44AddDelAddressRangeReply struct {
    Retval int32 // 返回值：0 = 成功，负数 = 错误码
}
```

## 使用示例

### P1.3 实现

```go
// configureNATAddressPool 配置 NAT 地址池
func configureNATAddressPool(ctx context.Context, vppConn *govppapi.Connection, publicIPs []net.IP) error {
    logger := log.FromContext(ctx).WithField("nat_server", "configure")

    client := nat44_ed.NewServiceClient(vppConn)

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
            logger.WithError(err).WithField("publicIP", publicIP.String()).Error("VPP API 调用失败")
            return errors.Wrapf(err, "VPP API Nat44AddDelAddressRange 调用失败（IP: %s）", publicIP)
        }

        // 4. 检查返回值
        if reply.Retval != 0 {
            logger.WithField("retval", reply.Retval).WithField("publicIP", publicIP.String()).Error("VPP API 返回错误")
            return fmt.Errorf("VPP API 返回错误: %d（IP: %s）", reply.Retval, publicIP)
        }

        logger.WithField("publicIP", publicIP.String()).Info("NAT 地址池配置成功")
    }

    return nil
}
```

### P2 配置文件集成

```go
// main.go - P2 阶段从配置文件加载公网 IP
func main() {
    // ... 加载配置 ...
    config, err := config.LoadConfig(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 创建 NAT server
    natServer := nat.NewServer(vppConn, config.NATConfig)
    // ...
}

// internal/nat/server.go - NewServer 构造函数
func NewServer(vppConn *govppapi.Connection, config *NATConfig) networkservice.NetworkServiceServer {
    server := &natServer{
        vppConn: vppConn,
        config:  config,
    }

    // P1.3 阶段在 Request() 中配置地址池
    // P2 阶段在 NewServer() 中配置地址池（服务启动时配置一次）
    if err := configureNATAddressPool(context.Background(), vppConn, config.PublicIPs); err != nil {
        log.WithError(err).Fatal("NAT 地址池配置失败")
    }

    return server
}
```

### P1.3 硬编码实现（简化版）

```go
// main.go - P1.3 阶段硬编码公网 IP
func main() {
    // ... 创建 VPP 连接 ...

    // 硬编码公网 IP 地址池
    publicIPs := []net.IP{net.ParseIP("192.168.1.100")}

    // 配置 NAT 地址池（服务启动时配置一次）
    if err := configureNATAddressPool(ctx, vppConn, publicIPs); err != nil {
        log.WithError(err).Fatal("NAT 地址池配置失败")
    }

    // 创建 NAT server
    natServer := nat.NewServer(vppConn, publicIPs)
    // ...
}
```

### 清理地址池（Close 时调用）

```go
// cleanupNATAddressPool 删除 NAT 地址池
func cleanupNATAddressPool(ctx context.Context, vppConn *govppapi.Connection, publicIPs []net.IP) error {
    logger := log.FromContext(ctx).WithField("nat_server", "cleanup")

    client := nat44_ed.NewServiceClient(vppConn)

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
            logger.WithError(err).WithField("publicIP", publicIP.String()).Error("VPP API 调用失败")
            // 继续删除其他地址，不中断
            continue
        }

        if reply.Retval != 0 {
            logger.WithField("retval", reply.Retval).WithField("publicIP", publicIP.String()).Warn("VPP API 返回错误")
            continue
        }

        logger.WithField("publicIP", publicIP.String()).Info("NAT 地址池删除成功")
    }

    return nil
}
```

## 错误处理

### 常见错误码

| 错误码 | 含义 | 可能原因 | 解决方法 |
|--------|------|----------|----------|
| 0 | 成功 | - | - |
| -1 | 通用错误 | IP 地址格式错误、VPP 内部错误 | 检查 IP 地址格式，查看 VPP 日志 |
| -3 | 资源已存在 | 地址池已配置 | 避免重复配置，检查是否已添加 |
| -2 | 资源不存在 | 删除不存在的地址池 | 确认地址池存在后再删除 |

### 错误日志示例

```
错误: NAT 地址池配置失败: VPP API Nat44AddDelAddressRange 调用失败（IP: 192.168.1.100）: rpc error: code = Unknown desc = invalid argument
```

## VPP CLI 验证

### 验证命令

```bash
vppctl show nat44 addresses
```

### 预期输出

```
NAT44 pool addresses:
  192.168.1.100
    tenant VRF independent
    0 busy udp ports
    0 busy tcp ports
    0 busy icmp ports
```

### 多 IP 地址池示例

```
NAT44 pool addresses:
  192.168.1.100
    tenant VRF independent
    5 busy udp ports
    3 busy tcp ports
    1 busy icmp ports
  192.168.1.101
    tenant VRF independent
    2 busy udp ports
    0 busy tcp ports
    0 busy icmp ports
```

## 测试用例

### 正常场景

```go
// 测试单个公网 IP 配置
func TestConfigureNATAddressPool_SingleIP_Success(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    publicIPs := []net.IP{net.ParseIP("192.168.1.100")}
    err := configureNATAddressPool(ctx, vppConn, publicIPs)
    assert.NoError(t, err)

    // 验证 VPP 配置
    output := execVPPCLI(t, "show nat44 addresses")
    assert.Contains(t, output, "192.168.1.100")
}

// 测试多个公网 IP 配置
func TestConfigureNATAddressPool_MultipleIPs_Success(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    publicIPs := []net.IP{
        net.ParseIP("192.168.1.100"),
        net.ParseIP("192.168.1.101"),
    }
    err := configureNATAddressPool(ctx, vppConn, publicIPs)
    assert.NoError(t, err)

    output := execVPPCLI(t, "show nat44 addresses")
    assert.Contains(t, output, "192.168.1.100")
    assert.Contains(t, output, "192.168.1.101")
}
```

### 边界场景

```go
// 测试空地址池
func TestConfigureNATAddressPool_EmptyList(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    publicIPs := []net.IP{}
    err := configureNATAddressPool(ctx, vppConn, publicIPs)
    assert.NoError(t, err) // 空列表不报错，直接跳过
}

// 测试重复配置
func TestConfigureNATAddressPool_Duplicate(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    publicIPs := []net.IP{net.ParseIP("192.168.1.100")}

    // 第一次配置成功
    err := configureNATAddressPool(ctx, vppConn, publicIPs)
    assert.NoError(t, err)

    // 第二次配置可能返回"资源已存在"错误
    err = configureNATAddressPool(ctx, vppConn, publicIPs)
    // 根据 VPP 实际行为调整断言
}
```

### 错误场景

```go
// 测试无效 IP 地址
func TestConfigureNATAddressPool_InvalidIP(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    // 无效 IPv4 地址（IPv6 地址）
    publicIPs := []net.IP{net.ParseIP("2001:db8::1")}
    err := configureNATAddressPool(ctx, vppConn, publicIPs)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "不是有效的 IPv4 地址")
}
```

## 端到端测试

### P1.3 完整测试流程

```bash
# 1. 部署 NAT NSE
kubectl apply -f deployments/nse-nat.yaml

# 2. 部署外部测试服务器
kubectl apply -f deployments/test-server.yaml

# 3. 部署 NSC
kubectl apply -f deployments/client.yaml

# 4. 进入 NAT NSE 容器，验证地址池配置
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 addresses
# 预期输出：192.168.1.100

# 5. 从 NSC 发起 ping 测试
kubectl exec <nsc-pod> -- ping -c 3 <test-server-ip>
# 预期：成功接收响应

# 6. 查看 NAT 会话
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 sessions
# 预期输出：
# NAT44 ED sessions:
#   i2o 172.16.1.1:12345 -> 192.168.1.100:12345 [protocol ICMP]

# 7. 在测试服务器查看接收到的源 IP
kubectl exec <test-server-pod> -- tcpdump -i eth0 -n icmp
# 预期：源 IP 是 192.168.1.100（公网 IP），而非 172.16.1.1
```

## 依赖

- **binapi 模块**: nat44_ed, nat_types, ip_types
- **前置条件**: P1.2 接口角色配置完成（Nat44InterfaceAddDelFeature）
- **后续操作**: NAT 地址转换功能生效，NSC 可以通过公网 IP 访问外部网络

## 参考

- VPP 官方文档: https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- govpp binapi: `internal/binapi_nat44_ed/nat44_ed.ba.go` (P3.2 本地化后)
- Data Model: [data-model.md](../data-model.md) - NAT 配置实体定义
- Configuration: [config.yaml](../../../deployments/config.yaml) (P2 阶段示例配置)

---

**更新记录**:
- 2025-01-15: 初始版本
