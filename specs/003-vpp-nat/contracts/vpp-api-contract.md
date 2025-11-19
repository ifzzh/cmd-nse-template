# VPP NAT44 API Contract

**VPP 版本**: 23.10-rc0~170-g6f1548434
**govpp 版本**: v0.0.0-20240328101142-8a444680fbba
**API 文件**: `plugins/nat44-ed/nat44_ed.api`

## API 概述

本文档定义 NAT 模块与 VPP NAT44 ED 插件之间的 Binary API 契约，确保 Go 代码正确调用 VPP 功能。

## 核心 API 调用

### 1. Nat44InterfaceAddDelFeature

**用途**: 配置或禁用 VPP 接口的 NAT inside/outside 角色

**VPP CLI 等价命令**:
```bash
vpp# set interface nat44 in <interface>   # IsAdd=true, Flags=NAT_IS_INSIDE
vpp# set interface nat44 out <interface>  # IsAdd=true, Flags=NAT_IS_OUTSIDE
```

**Go 结构体定义** (来自 `binapi/nat44_ed/nat44_ed.ba.go:1879-1918`):
```go
type Nat44InterfaceAddDelFeature struct {
    IsAdd     bool                           `binapi:"bool,name=is_add"`
    Flags     nat_types.NatConfigFlags       `binapi:"nat_config_flags,name=flags"`
    SwIfIndex interface_types.InterfaceIndex `binapi:"interface_index,name=sw_if_index"`
}

type Nat44InterfaceAddDelFeatureReply struct {
    Retval int32 `binapi:"i32,name=retval"`
}
```

**参数契约**:

| 字段 | 类型 | 取值 | 说明 |
|------|------|------|------|
| `IsAdd` | bool | `true` / `false` | `true` = 启用 NAT，`false` = 禁用 NAT |
| `Flags` | NatConfigFlags | `NAT_IS_INSIDE` (32) / `NAT_IS_OUTSIDE` (16) | 接口角色 |
| `SwIfIndex` | InterfaceIndex | 有效接口索引 | 从 `ifindex.Load(ctx, isClient)` 获取 |

**返回值契约**:

| 字段 | 类型 | 成功值 | 错误值 | 说明 |
|------|------|-------|-------|------|
| `Retval` | int32 | `0` | 非零 | 0 = 成功，非零 = 错误码 |

**调用示例**:
```go
func configureNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex uint32, isInside bool) error {
    flags := nat_types.NAT_IS_OUTSIDE
    if isInside {
        flags = nat_types.NAT_IS_INSIDE
    }

    req := &nat44_ed.Nat44InterfaceAddDelFeature{
        IsAdd:     true,
        Flags:     flags,
        SwIfIndex: interface_types.InterfaceIndex(swIfIndex),
    }

    reply, err := nat44_ed.NewServiceClient(vppConn).Nat44InterfaceAddDelFeature(ctx, req)
    if err != nil {
        return errors.Wrap(err, "VPP API Nat44InterfaceAddDelFeature 调用失败")
    }
    if reply.Retval != 0 {
        return errors.Errorf("VPP API 返回错误码: %d", reply.Retval)
    }

    return nil
}
```

**错误场景**:

| Retval | 含义 | 处理策略 |
|--------|------|---------|
| `0` | 成功 | 继续执行 |
| `-1` | 通用错误 | 返回错误，调用 Close() 清理 |
| `-2` | 接口不存在 | 返回错误，记录 swIfIndex |
| `-68` | 接口已配置 NAT | 忽略（幂等性） |

### 2. Nat44AddDelAddressRange

**用途**: 配置或删除 NAT 公网 IP 地址池

**VPP CLI 等价命令**:
```bash
vpp# nat44 add address <ip>           # IsAdd=true
vpp# nat44 del address <ip>           # IsAdd=false
vpp# nat44 add address <start>-<end>  # FirstIPAddress != LastIPAddress
```

**Go 结构体定义** (来自 `binapi/nat44_ed/nat44_ed.ba.go:108-114`):
```go
type Nat44AddDelAddressRange struct {
    FirstIPAddress ip_types.IP4Address      `binapi:"ip4_address,name=first_ip_address"`
    LastIPAddress  ip_types.IP4Address      `binapi:"ip4_address,name=last_ip_address"`
    VrfID          uint32                   `binapi:"u32,name=vrf_id"`
    IsAdd          bool                     `binapi:"bool,name=is_add"`
    Flags          nat_types.NatConfigFlags `binapi:"nat_config_flags,name=flags"`
}

type Nat44AddDelAddressRangeReply struct {
    Retval int32 `binapi:"i32,name=retval"`
}
```

**参数契约**:

| 字段 | 类型 | 取值 | 说明 |
|------|------|------|------|
| `FirstIPAddress` | IP4Address | 公网 IPv4 地址 | 地址池起始 IP |
| `LastIPAddress` | IP4Address | 公网 IPv4 地址 | 地址池结束 IP（单个 IP 时与 FirstIPAddress 相同） |
| `VrfID` | uint32 | `0` | VRF ID（默认 0） |
| `IsAdd` | bool | `true` / `false` | `true` = 添加地址池，`false` = 删除地址池 |
| `Flags` | NatConfigFlags | `0` | 标志位（默认 0，无特殊标志） |

**返回值契约**:

| 字段 | 类型 | 成功值 | 错误值 | 说明 |
|------|------|-------|-------|------|
| `Retval` | int32 | `0` | 非零 | 0 = 成功，非零 = 错误码 |

**调用示例**:
```go
func configureNATAddressPool(ctx context.Context, vppConn api.Connection, publicIP net.IP) error {
    // 将 net.IP 转换为 ip_types.IP4Address
    var ip4Addr ip_types.IP4Address
    copy(ip4Addr[:], publicIP.To4())

    req := &nat44_ed.Nat44AddDelAddressRange{
        FirstIPAddress: ip4Addr,
        LastIPAddress:  ip4Addr, // 单个 IP
        VrfID:          0,
        IsAdd:          true,
        Flags:          0,
    }

    reply, err := nat44_ed.NewServiceClient(vppConn).Nat44AddDelAddressRange(ctx, req)
    if err != nil {
        return errors.Wrap(err, "VPP API Nat44AddDelAddressRange 调用失败")
    }
    if reply.Retval != 0 {
        return errors.Errorf("VPP API 返回错误码: %d", reply.Retval)
    }

    return nil
}
```

**错误场景**:

| Retval | 含义 | 处理策略 |
|--------|------|---------|
| `0` | 成功 | 继续执行 |
| `-1` | 通用错误 | 返回错误，调用 Close() 清理 |
| `-3` | IP 已存在 | 忽略（幂等性） |
| `-4` | IP 不存在（删除时） | 忽略（清理阶段） |
| `-12` | 内存不足 | 返回错误，记录日志 |

### 3. Nat44AddDelStaticMapping (P4 阶段)

**用途**: 配置或删除静态端口映射（Port Forwarding）

**VPP CLI 等价命令**:
```bash
vpp# nat44 add static mapping tcp local <internal-ip> <internal-port> external <public-ip> <public-port>
```

**Go 结构体定义** (来自 `binapi/nat44_ed/nat44_ed.ba.go:170-186`):
```go
type Nat44AddDelStaticMapping struct {
    IsAdd             bool                           `binapi:"bool,name=is_add"`
    Flags             nat_types.NatConfigFlags       `binapi:"nat_config_flags,name=flags"`
    LocalIPAddress    ip_types.IP4Address            `binapi:"ip4_address,name=local_ip_address"`
    ExternalIPAddress ip_types.IP4Address            `binapi:"ip4_address,name=external_ip_address"`
    Protocol          uint8                          `binapi:"u8,name=protocol"`
    LocalPort         uint16                         `binapi:"u16,name=local_port"`
    ExternalPort      uint16                         `binapi:"u16,name=external_port"`
    ExternalSwIfIndex interface_types.InterfaceIndex `binapi:"interface_index,name=external_sw_if_index"`
    VrfID             uint32                         `binapi:"u32,name=vrf_id"`
    Tag               string                         `binapi:"string[64],name=tag"`
}

type Nat44AddDelStaticMappingReply struct {
    Retval int32 `binapi:"i32,name=retval"`
}
```

**参数契约**:

| 字段 | 类型 | 取值 | 说明 |
|------|------|------|------|
| `IsAdd` | bool | `true` / `false` | `true` = 添加映射，`false` = 删除映射 |
| `Flags` | NatConfigFlags | `0` / `NAT_IS_ADDR_ONLY` (8) | 0 = 端口映射，NAT_IS_ADDR_ONLY = 仅 IP 映射 |
| `LocalIPAddress` | IP4Address | 内部服务器 IPv4 | 内部 IP |
| `ExternalIPAddress` | IP4Address | 公网 IPv4 | 公网 IP |
| `Protocol` | uint8 | `6` (TCP) / `17` (UDP) / `1` (ICMP) | IP 协议号 |
| `LocalPort` | uint16 | 1-65535 | 内部端口 |
| `ExternalPort` | uint16 | 1-65535 | 公网端口 |
| `ExternalSwIfIndex` | InterfaceIndex | `0xFFFFFFFF` | 无效值（使用 ExternalIPAddress） |
| `VrfID` | uint32 | `0` | VRF ID（默认 0） |
| `Tag` | string[64] | 任意字符串（≤64 字节） | 映射标签（用于标识和查询） |

**返回值契约**:

| 字段 | 类型 | 成功值 | 错误值 | 说明 |
|------|------|-------|-------|------|
| `Retval` | int32 | `0` | 非零 | 0 = 成功，非零 = 错误码 |

**调用示例** (P4):
```go
func addStaticMapping(ctx context.Context, vppConn api.Connection, cfg StaticMapConfig) error {
    var localIP, externalIP ip_types.IP4Address
    copy(localIP[:], net.ParseIP(cfg.InternalIP).To4())
    copy(externalIP[:], net.ParseIP(cfg.PublicIP).To4())

    protocol := uint8(6) // TCP
    if cfg.Protocol == "udp" {
        protocol = 17
    } else if cfg.Protocol == "icmp" {
        protocol = 1
    }

    req := &nat44_ed.Nat44AddDelStaticMapping{
        IsAdd:             true,
        Flags:             0,
        LocalIPAddress:    localIP,
        ExternalIPAddress: externalIP,
        Protocol:          protocol,
        LocalPort:         cfg.InternalPort,
        ExternalPort:      cfg.PublicPort,
        ExternalSwIfIndex: 0xFFFFFFFF, // 无效值
        VrfID:             0,
        Tag:               cfg.Tag,
    }

    reply, err := nat44_ed.NewServiceClient(vppConn).Nat44AddDelStaticMapping(ctx, req)
    if err != nil {
        return errors.Wrap(err, "VPP API Nat44AddDelStaticMapping 调用失败")
    }
    if reply.Retval != 0 {
        return errors.Errorf("VPP API 返回错误码: %d", reply.Retval)
    }

    return nil
}
```

**错误场景**:

| Retval | 含义 | 处理策略 |
|--------|------|---------|
| `0` | 成功 | 继续执行 |
| `-1` | 通用错误 | 返回错误，记录日志 |
| `-6` | 端口已占用 | 返回错误，跳过该映射 |
| `-7` | 映射已存在 | 忽略（幂等性） |

## 查询 API (可选，用于验证)

### Nat44InterfaceDump

**用途**: 查询所有配置 NAT 的接口

**调用示例**:
```go
func verifyNATInterfaces(ctx context.Context, vppConn api.Connection) error {
    req := &nat44_ed.Nat44InterfaceDump{}
    stream, err := nat44_ed.NewServiceClient(vppConn).Nat44InterfaceDump(ctx, req)
    if err != nil {
        return err
    }

    for {
        details, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        log.Infof("NAT 接口: SwIfIndex=%d, Flags=%d", details.SwIfIndex, details.Flags)
    }

    return nil
}
```

### Nat44UserSessionDump

**用途**: 查询 NAT 会话表

**调用示例**:
```go
func verifyNATSessions(ctx context.Context, vppConn api.Connection) error {
    req := &nat44_ed.Nat44UserSessionDump{
        IPAddress: ip_types.IP4Address{0, 0, 0, 0}, // 查询所有会话
        VrfID:     0,
    }

    stream, err := nat44_ed.NewServiceClient(vppConn).Nat44UserSessionDump(ctx, req)
    if err != nil {
        return err
    }

    for {
        details, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        log.Infof("NAT 会话: 内部=%s:%d, 外部=%s:%d",
            details.InsideIPAddress, details.InsidePort,
            details.OutsideIPAddress, details.OutsidePort)
    }

    return nil
}
```

## 类型转换契约

### net.IP → ip_types.IP4Address

```go
// Go 标准库 net.IP 转 VPP IP4Address
func ipToIP4Address(ip net.IP) ip_types.IP4Address {
    var addr ip_types.IP4Address
    ipv4 := ip.To4()
    if ipv4 == nil {
        // 错误处理：非 IPv4 地址
        return addr // 返回零值
    }
    copy(addr[:], ipv4)
    return addr
}
```

### ip_types.IP4Address → net.IP

```go
// VPP IP4Address 转 Go 标准库 net.IP
func ip4AddressToIP(addr ip_types.IP4Address) net.IP {
    return net.IPv4(addr[0], addr[1], addr[2], addr[3])
}
```

### string → ip_types.IP4Address

```go
// 字符串 IP 地址转 VPP IP4Address
func parseIPToIP4Address(ipStr string) (ip_types.IP4Address, error) {
    ip := net.ParseIP(ipStr)
    if ip == nil {
        return ip_types.IP4Address{}, errors.Errorf("无效的 IP 地址: %s", ipStr)
    }
    ipv4 := ip.To4()
    if ipv4 == nil {
        return ip_types.IP4Address{}, errors.Errorf("不是 IPv4 地址: %s", ipStr)
    }
    var addr ip_types.IP4Address
    copy(addr[:], ipv4)
    return addr, nil
}
```

## 错误码参考

| Retval | 符号名称 | 含义 |
|--------|---------|------|
| `0` | `VNET_API_ERROR_NO_ERROR` | 成功 |
| `-1` | `VNET_API_ERROR_UNIMPLEMENTED` | 未实现 |
| `-2` | `VNET_API_ERROR_NO_SUCH_ENTRY` | 条目不存在 |
| `-3` | `VNET_API_ERROR_VALUE_EXIST` | 值已存在 |
| `-4` | `VNET_API_ERROR_INVALID_VALUE` | 无效值 |
| `-6` | `VNET_API_ERROR_IN_USE` | 正在使用 |
| `-12` | `VNET_API_ERROR_NO_MEMORY` | 内存不足 |
| `-68` | `VNET_API_ERROR_FEATURE_ALREADY_ENABLED` | 功能已启用 |

## API 调用顺序契约

### 正常流程（Request）

```
1. Nat44InterfaceAddDelFeature (IsAdd=true, Flags=NAT_IS_INSIDE/OUTSIDE)
   ↓
2. Nat44AddDelAddressRange (IsAdd=true)
   ↓
3. [可选] Nat44AddDelStaticMapping (IsAdd=true, P4 阶段)
```

### 清理流程（Close）

```
1. [可选] Nat44AddDelStaticMapping (IsAdd=false, P4 阶段)
   ↓
2. Nat44AddDelAddressRange (IsAdd=false)
   ↓
3. Nat44InterfaceAddDelFeature (IsAdd=false)
```

**重要约束**:
- 必须先删除地址池，再禁用接口 NAT（否则可能导致 VPP 配置残留）
- 删除地址池时，必须使用与添加时相同的 IP 地址
- 禁用接口 NAT 时，必须使用与启用时相同的 Flags（inside/outside）

## 性能契约

| API | 预期延迟 | 并发安全 | 幂等性 |
|-----|---------|---------|-------|
| `Nat44InterfaceAddDelFeature` | <1ms | 是 | 是（重复启用返回 -68） |
| `Nat44AddDelAddressRange` | <1ms | 是 | 是（重复添加返回 -3） |
| `Nat44AddDelStaticMapping` | <1ms | 是 | 是（重复添加返回 -7） |
| `Nat44InterfaceDump` | <5ms | 是 | N/A（查询操作） |
| `Nat44UserSessionDump` | <10ms | 是 | N/A（查询操作） |

## 版本兼容性

| VPP 版本 | NAT44 ED 插件 | API 版本 | 兼容性 |
|----------|--------------|---------|-------|
| 23.10 | 支持 | v5.2.0 | ✅ 完全兼容 |
| 24.10.0 | 支持 | v5.3.0 | ✅ 向后兼容 |
| 22.x | 部分支持 | v4.x | ⚠️ 需要验证 |

**注意**: govpp binapi 生成自 VPP 23.10，与 VPP 24.10.0 兼容，但需验证 API 结构体字段是否有变更。

## 测试契约

### VPP CLI 验证命令

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

# 检查静态映射（P4）
vpp# show nat44 static mappings
# 预期输出: 公网IP:端口 → 内部IP:端口 映射
```

### 错误注入测试

- 使用无效接口索引 → 验证返回 -2
- 重复配置同一接口 → 验证返回 -68（幂等性）
- 重复添加同一 IP → 验证返回 -3（幂等性）
- 删除不存在的 IP → 验证返回 -4（清理阶段忽略）

## 参考文档

- VPP NAT44 ED API 定义: `https://github.com/FDio/vpp/blob/stable/2310/src/plugins/nat/nat44-ed/nat44_ed.api`
- VPP NAT44 ED 插件文档: `https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html`
- govpp binapi 源代码: `/root/go/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba/binapi/nat44_ed/`
