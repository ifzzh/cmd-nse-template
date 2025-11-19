# API: Nat44AddDelStaticMapping

**阶段**: P4
**用途**: 配置静态端口映射（端口转发 / Port Forwarding）
**VPP 版本**: VPP v23.10-rc0~170 / v24.10.0
**注意**: 此 API 在 govpp binapi 中标记为 `Deprecated`，但在 VPP v23.10 / v24.10 中仍可用且功能完整

## API 定义

### 请求结构

```go
type Nat44AddDelStaticMapping struct {
    IsAdd             bool                           // true = 添加静态映射, false = 删除静态映射
    Flags             nat_types.NatConfigFlags       // 标志位（0 = 端口映射，NAT_IS_ADDR_ONLY = 仅地址映射）
    LocalIPAddress    ip_types.IP4Address            // 内部服务器 IP 地址
    ExternalIPAddress ip_types.IP4Address            // 公网 IP 地址
    Protocol          uint8                          // 协议类型（6 = TCP, 17 = UDP, 1 = ICMP）
    LocalPort         uint16                         // 内部端口
    ExternalPort      uint16                         // 公网端口
    ExternalSwIfIndex interface_types.InterfaceIndex // 外部接口索引（可选，默认 0xFFFFFFFF）
    VrfID             uint32                         // VRF ID（默认 0）
    Tag               string                         // 标签（最长 64 字符，用于描述映射用途）
}
```

### 字段说明

| 字段名 | 类型 | 必填 | 描述 | 有效值 |
|--------|------|------|------|--------|
| IsAdd | bool | ✅ | 添加或删除静态映射 | true（添加）、false（删除） |
| Flags | nat_types.NatConfigFlags | ❌ | 标志位 | 0（端口映射）、NAT_IS_ADDR_ONLY（仅地址映射，不限端口） |
| LocalIPAddress | ip_types.IP4Address | ✅ | 内部服务器 IP | 有效的 IPv4 地址（4 字节数组） |
| ExternalIPAddress | ip_types.IP4Address | ✅ | 公网 IP | 必须在地址池中（P1.3 已配置） |
| Protocol | uint8 | ✅ | 协议类型 | 6（TCP）、17（UDP）、1（ICMP） |
| LocalPort | uint16 | ✅ | 内部端口 | 1-65535（ICMP 时可为 0） |
| ExternalPort | uint16 | ✅ | 公网端口 | 1-65535（ICMP 时可为 0） |
| ExternalSwIfIndex | interface_types.InterfaceIndex | ❌ | 外部接口索引 | 默认 0xFFFFFFFF（使用地址池） |
| VrfID | uint32 | ❌ | VRF ID | 默认 0 |
| Tag | string | ❌ | 描述标签 | 最长 64 字符 |

### 响应结构

```go
type Nat44AddDelStaticMappingReply struct {
    Retval int32 // 返回值：0 = 成功，负数 = 错误码
}
```

## 使用示例

### P4 实现

```go
// configureStaticMapping 配置静态端口映射
func configureStaticMapping(ctx context.Context, vppConn *govppapi.Connection, mapping StaticMapping) error {
    logger := log.FromContext(ctx).WithFields(logrus.Fields{
        "nat_server": "configure",
        "mapping":    fmt.Sprintf("%s:%d -> %s:%d", mapping.PublicIP, mapping.PublicPort, mapping.InternalIP, mapping.InternalPort),
    })

    client := nat44_ed.NewServiceClient(vppConn)

    // 1. 协议转换（字符串 -> 数字）
    var protocol uint8
    switch mapping.Protocol {
    case "tcp":
        protocol = 6
    case "udp":
        protocol = 17
    case "icmp":
        protocol = 1
    default:
        return fmt.Errorf("不支持的协议类型: %s", mapping.Protocol)
    }

    // 2. 转换 IP 地址
    publicIP4 := mapping.PublicIP.To4()
    if publicIP4 == nil {
        return fmt.Errorf("公网 IP 不是有效的 IPv4 地址: %s", mapping.PublicIP)
    }
    var externalIP ip_types.IP4Address
    copy(externalIP[:], publicIP4)

    internalIP4 := mapping.InternalIP.To4()
    if internalIP4 == nil {
        return fmt.Errorf("内部 IP 不是有效的 IPv4 地址: %s", mapping.InternalIP)
    }
    var localIP ip_types.IP4Address
    copy(localIP[:], internalIP4)

    // 3. 创建请求
    req := &nat44_ed.Nat44AddDelStaticMapping{
        IsAdd:             true,
        Flags:             0, // 端口映射模式
        LocalIPAddress:    localIP,
        ExternalIPAddress: externalIP,
        Protocol:          protocol,
        LocalPort:         mapping.InternalPort,
        ExternalPort:      mapping.PublicPort,
        ExternalSwIfIndex: 0xFFFFFFFF, // 使用地址池
        VrfID:             0,           // 默认 VRF
        Tag:               mapping.Tag,
    }

    // 4. 调用 VPP API
    reply := &nat44_ed.Nat44AddDelStaticMappingReply{}
    if err := vppConn.Invoke(ctx, req, reply); err != nil {
        logger.WithError(err).Error("VPP API 调用失败")
        return errors.Wrap(err, "VPP API Nat44AddDelStaticMapping 调用失败")
    }

    // 5. 检查返回值
    if reply.Retval != 0 {
        logger.WithField("retval", reply.Retval).Error("VPP API 返回错误")
        return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
    }

    logger.Info("NAT 静态端口映射配置成功")
    return nil
}
```

### P4 配置文件集成

```yaml
# /etc/nat/config.yaml - P4 阶段配置示例
public_ips:
  - 192.168.1.100

port_range:
  min: 1024
  max: 65535

static_mappings:
  - protocol: tcp
    public_ip: 192.168.1.100
    public_port: 8080
    internal_ip: 172.16.1.10
    internal_port: 80
    tag: "web-server-mapping"

  - protocol: tcp
    public_ip: 192.168.1.100
    public_port: 3306
    internal_ip: 172.16.1.20
    internal_port: 3306
    tag: "mysql-database"
```

```go
// NewServer 构造函数 - P4 阶段集成静态映射
func NewServer(vppConn *govppapi.Connection, config *NATConfig) networkservice.NetworkServiceServer {
    server := &natServer{
        vppConn: vppConn,
        config:  config,
    }

    // 1. 配置地址池（P1.3）
    if err := configureNATAddressPool(context.Background(), vppConn, config.PublicIPs); err != nil {
        log.WithError(err).Fatal("NAT 地址池配置失败")
    }

    // 2. 配置静态端口映射（P4）
    for _, mapping := range config.StaticMappings {
        if err := configureStaticMapping(context.Background(), vppConn, mapping); err != nil {
            log.WithError(err).Fatal("NAT 静态端口映射配置失败")
        }
    }

    return server
}
```

### 删除静态映射

```go
// cleanupStaticMapping 删除静态端口映射
func cleanupStaticMapping(ctx context.Context, vppConn *govppapi.Connection, mapping StaticMapping) error {
    logger := log.FromContext(ctx).WithFields(logrus.Fields{
        "nat_server": "cleanup",
        "mapping":    fmt.Sprintf("%s:%d -> %s:%d", mapping.PublicIP, mapping.PublicPort, mapping.InternalIP, mapping.InternalPort),
    })

    client := nat44_ed.NewServiceClient(vppConn)

    var protocol uint8
    switch mapping.Protocol {
    case "tcp":
        protocol = 6
    case "udp":
        protocol = 17
    case "icmp":
        protocol = 1
    default:
        return fmt.Errorf("不支持的协议类型: %s", mapping.Protocol)
    }

    publicIP4 := mapping.PublicIP.To4()
    var externalIP ip_types.IP4Address
    copy(externalIP[:], publicIP4)

    internalIP4 := mapping.InternalIP.To4()
    var localIP ip_types.IP4Address
    copy(localIP[:], internalIP4)

    req := &nat44_ed.Nat44AddDelStaticMapping{
        IsAdd:             false, // 删除静态映射
        Flags:             0,
        LocalIPAddress:    localIP,
        ExternalIPAddress: externalIP,
        Protocol:          protocol,
        LocalPort:         mapping.InternalPort,
        ExternalPort:      mapping.PublicPort,
        ExternalSwIfIndex: 0xFFFFFFFF,
        VrfID:             0,
        Tag:               mapping.Tag,
    }

    reply := &nat44_ed.Nat44AddDelStaticMappingReply{}
    if err := vppConn.Invoke(ctx, req, reply); err != nil {
        logger.WithError(err).Error("VPP API 调用失败")
        // 继续清理其他映射，不中断
        return nil
    }

    if reply.Retval != 0 {
        logger.WithField("retval", reply.Retval).Warn("VPP API 返回错误")
        return nil
    }

    logger.Info("NAT 静态端口映射删除成功")
    return nil
}
```

## 错误处理

### 常见错误码

| 错误码 | 含义 | 可能原因 | 解决方法 |
|--------|------|----------|----------|
| 0 | 成功 | - | - |
| -1 | 通用错误 | 参数无效、VPP 内部错误 | 检查参数，查看 VPP 日志 |
| -3 | 资源已存在 | 静态映射已配置 | 避免重复配置，检查现有映射 |
| -2 | 资源不存在 | 删除不存在的静态映射 | 确认映射存在后再删除 |
| -22 | 无效参数 | 端口号为 0、IP 地址格式错误 | 验证配置参数 |

### 错误日志示例

```
错误: NAT 静态端口映射配置失败: VPP API Nat44AddDelStaticMapping 调用失败: rpc error: code = Unknown desc = invalid argument
```

## VPP CLI 验证

### 验证命令

```bash
vppctl show nat44 static mappings
```

### 预期输出

```
NAT44 static mappings:
  tcp local 172.16.1.10:80 external 192.168.1.100:8080 vrf 0
  tcp local 172.16.1.20:3306 external 192.168.1.100:3306 vrf 0
```

### UDP 映射示例

```
NAT44 static mappings:
  udp local 172.16.1.30:53 external 192.168.1.100:53 vrf 0
```

## 测试用例

### 正常场景

```go
// 测试 TCP 静态端口映射
func TestConfigureStaticMapping_TCP_Success(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    mapping := StaticMapping{
        Protocol:     "tcp",
        PublicIP:     net.ParseIP("192.168.1.100"),
        PublicPort:   8080,
        InternalIP:   net.ParseIP("172.16.1.10"),
        InternalPort: 80,
        Tag:          "web-server",
    }

    err := configureStaticMapping(ctx, vppConn, mapping)
    assert.NoError(t, err)

    // 验证 VPP 配置
    output := execVPPCLI(t, "show nat44 static mappings")
    assert.Contains(t, output, "tcp local 172.16.1.10:80 external 192.168.1.100:8080")
}
```

### 边界场景

```go
// 测试重复配置
func TestConfigureStaticMapping_Duplicate(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    mapping := StaticMapping{
        Protocol:     "tcp",
        PublicIP:     net.ParseIP("192.168.1.100"),
        PublicPort:   8080,
        InternalIP:   net.ParseIP("172.16.1.10"),
        InternalPort: 80,
        Tag:          "web-server",
    }

    // 第一次配置成功
    err := configureStaticMapping(ctx, vppConn, mapping)
    assert.NoError(t, err)

    // 第二次配置可能返回"资源已存在"错误
    err = configureStaticMapping(ctx, vppConn, mapping)
    // 根据 VPP 实际行为调整断言
}
```

### 错误场景

```go
// 测试无效协议
func TestConfigureStaticMapping_InvalidProtocol(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    mapping := StaticMapping{
        Protocol:     "invalid",
        PublicIP:     net.ParseIP("192.168.1.100"),
        PublicPort:   8080,
        InternalIP:   net.ParseIP("172.16.1.10"),
        InternalPort: 80,
        Tag:          "test",
    }

    err := configureStaticMapping(ctx, vppConn, mapping)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "不支持的协议类型")
}
```

## 端到端测试

### P4 完整测试流程

```bash
# 1. 部署 NAT NSE（包含静态映射配置）
kubectl apply -f deployments/nse-nat-p4.yaml

# 2. 部署内部 Web 服务器（NSC）
kubectl apply -f deployments/internal-web-server.yaml

# 3. 验证静态映射配置
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 static mappings
# 预期输出：
# NAT44 static mappings:
#   tcp local 172.16.1.10:80 external 192.168.1.100:8080 vrf 0

# 4. 从外部客户端访问公网 IP:8080
curl http://192.168.1.100:8080
# 预期：成功返回 Web 服务器响应

# 5. 查看 NAT 会话（验证静态映射生效）
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 sessions
# 预期输出：
# NAT44 ED sessions:
#   o2i 192.168.1.100:8080 <- <external-client-ip>:<random-port> [protocol TCP]
#       internal: 172.16.1.10:80

# 6. 在内部 Web 服务器查看请求
kubectl logs <web-server-pod>
# 预期：日志显示请求来自外部客户端
```

## 与动态 NAT 共存

### 场景说明

静态端口映射（P4）与动态 SNAT（P1）可以共存：
- **动态 SNAT**: 出站流量（NSC → 外部服务器），自动分配端口
- **静态映射**: 入站流量（外部客户端 → 内部服务器），固定端口映射

### 共存示例

```
┌─────────────────────────────────────────────────────────────┐
│                      NAT NSE                                │
│                                                             │
│  动态 SNAT（P1）:                                           │
│  NSC 172.16.1.1:12345 → 192.168.1.100:54321 → 8.8.8.8:80   │
│                                                             │
│  静态映射（P4）:                                            │
│  外部客户端:随机端口 → 192.168.1.100:8080 → 172.16.1.10:80 │
└─────────────────────────────────────────────────────────────┘
```

## 依赖

- **binapi 模块**: nat44_ed, nat_types, ip_types, interface_types
- **前置条件**:
  - P1.3 地址池配置完成（公网 IP 必须在地址池中）
  - P1.2 接口角色配置完成（inside/outside）
- **后续操作**: 静态端口映射生效，外部客户端可以访问内部服务

## 参考

- VPP 官方文档: https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- govpp binapi: `internal/binapi_nat44_ed/nat44_ed.ba.go` (P3.2 本地化后)
- Data Model: [data-model.md](../data-model.md) - 静态端口映射实体定义
- Research: [research.md](../research.md) - API 稳定性验证

---

**更新记录**:
- 2025-01-15: 初始版本
