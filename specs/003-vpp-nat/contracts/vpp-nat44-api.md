# VPP NAT44 ED Binary API 调用规范

**项目**: VPP NAT 网络服务端点
**版本**: 1.0
**日期**: 2025-01-15
**VPP 版本**: VPP v23.10-rc0~170 / v24.10.0

## 概述

本文档定义 VPP NAT44 Endpoint-Dependent (ED) 插件的 Binary API 调用规范，用于在 NSM 网络服务端点中实现 NAT 功能。所有 API 通过 govpp 框架调用，使用 binapi 自动生成的 Go 绑定。

## API 清单

| API 名称 | 用途 | 阶段 | 详细文档 |
|---------|------|------|---------|
| Nat44InterfaceAddDelFeature | 配置接口角色（inside/outside） | P1.2 | [interface-role-api.md](./interface-role-api.md) |
| Nat44AddDelAddressRange | 配置地址池 | P1.3 | [address-pool-api.md](./address-pool-api.md) |
| Nat44AddDelStaticMapping | 配置静态端口映射 | P4 | [static-mapping-api.md](./static-mapping-api.md) |

## 依赖模块

### binapi 模块

| 模块名 | 导入路径 | 用途 | 本地化阶段 |
|--------|----------|------|-----------|
| nat_types | go.fd.io/govpp/binapi/nat_types | NAT 基础类型定义 | P3.1 (v1.1.0) |
| nat44_ed | go.fd.io/govpp/binapi/nat44_ed | NAT44 ED API 绑定 | P3.2 (v1.1.1) |
| interface_types | go.fd.io/govpp/binapi/interface_types | 接口类型定义 | 已内置 |
| ip_types | go.fd.io/govpp/binapi/ip_types | IP 地址类型定义 | 已内置 |

### 本地化后的导入路径（P3 阶段后）

```go
import (
    "go.fd.io/govpp/api"
    nat44_ed "github.com/ifzzh/cmd-nse-template/internal/binapi_nat44_ed"
    nat_types "github.com/ifzzh/cmd-nse-template/internal/binapi_nat_types"
    interface_types "go.fd.io/govpp/binapi/interface_types"
    ip_types "go.fd.io/govpp/binapi/ip_types"
)
```

## 通用调用模式

### 标准 API 调用流程

```go
// 1. 创建 API 客户端
client := nat44_ed.NewServiceClient(vppConn)

// 2. 创建请求结构
req := &nat44_ed.Nat44InterfaceAddDelFeature{
    IsAdd:     true,
    Flags:     nat_types.NAT_IS_INSIDE,
    SwIfIndex: swIfIndex,
}

// 3. 调用 VPP API
reply := &nat44_ed.Nat44InterfaceAddDelFeatureReply{}
if err := vppConn.Invoke(context.Background(), req, reply); err != nil {
    return fmt.Errorf("VPP API 调用失败: %w", err)
}

// 4. 检查返回值
if reply.Retval != 0 {
    return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
}

log.Info("NAT 配置成功")
return nil
```

### 错误处理模式

```go
func callVPPAPI(vppConn *govppapi.Connection, swIfIndex interface_types.InterfaceIndex) error {
    // 1. 上下文日志
    logger := log.WithField("swIfIndex", swIfIndex)

    // 2. API 调用
    client := nat44_ed.NewServiceClient(vppConn)
    req := &nat44_ed.Nat44InterfaceAddDelFeature{
        IsAdd:     true,
        Flags:     nat_types.NAT_IS_INSIDE,
        SwIfIndex: swIfIndex,
    }

    reply := &nat44_ed.Nat44InterfaceAddDelFeatureReply{}
    if err := vppConn.Invoke(context.Background(), req, reply); err != nil {
        logger.WithError(err).Error("VPP API 调用失败")
        return errors.Wrap(err, "VPP API Nat44InterfaceAddDelFeature 调用失败")
    }

    // 3. 返回值检查
    if reply.Retval != 0 {
        logger.WithField("retval", reply.Retval).Error("VPP API 返回错误")
        return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
    }

    logger.Info("NAT inside 接口配置成功")
    return nil
}
```

## 常见错误码

| 错误码 | 含义 | 可能原因 | 解决方法 |
|--------|------|----------|----------|
| 0 | 成功 | - | - |
| -1 | 通用错误 | API 参数无效、VPP 内部错误 | 检查日志，验证参数 |
| -2 | 资源不存在 | 接口索引无效、地址池不存在 | 确认接口已创建，检查配置 |
| -3 | 资源已存在 | 重复配置 NAT 接口或地址池 | 检查是否已配置，避免重复调用 |
| -68 | 端口池耗尽 | NAT 端口分配失败 | 扩大端口范围或增加公网 IP |
| -113 | 无可用地址 | NAT 地址池为空 | 配置地址池后再启用接口 |

## VPP CLI 验证命令

### 查看 NAT 接口配置

```bash
vppctl show nat44 interfaces
```

**预期输出示例**:
```
NAT44 interfaces:
  memif0/0 in   (inside 接口)
  memif0/1 out  (outside 接口)
```

### 查看 NAT 地址池

```bash
vppctl show nat44 addresses
```

**预期输出示例**:
```
NAT44 pool addresses:
  192.168.1.100
    tenant VRF independent
    0 busy udp ports
    0 busy tcp ports
    0 busy icmp ports
```

### 查看 NAT 会话

```bash
vppctl show nat44 sessions
```

**预期输出示例**:
```
NAT44 ED sessions:
  i2o 172.16.1.1:12345 -> 192.168.1.100:54321 [protocol TCP]
      external: 8.8.8.8:80
      state: ESTABLISHED
      timeout: 7200s
```

### 查看静态端口映射

```bash
vppctl show nat44 static mappings
```

**预期输出示例**:
```
NAT44 static mappings:
  tcp local 172.16.1.10:80 external 192.168.1.100:8080 vrf 0
```

## API 调用顺序

### P1.2 - 接口角色配置

```
1. NSM 创建 VPP 接口（memif/kernel）
2. 获取接口索引（ifindex.Load）
3. 调用 Nat44InterfaceAddDelFeature 配置 inside 接口
4. 调用 Nat44InterfaceAddDelFeature 配置 outside 接口
5. 验证：vppctl show nat44 interfaces
```

### P1.3 - 地址池配置与集成

```
1. 完成 P1.2 接口角色配置
2. 调用 Nat44AddDelAddressRange 配置地址池
3. 验证：vppctl show nat44 addresses
4. 端到端测试：NSC → NAT NSE → 外部服务器
5. 验证：vppctl show nat44 sessions
```

### P4 - 静态端口映射

```
1. 完成 P1.3 地址池配置
2. 调用 Nat44AddDelStaticMapping 创建静态映射
3. 验证：vppctl show nat44 static mappings
4. 端到端测试：外部客户端 → 公网IP:端口 → 内部服务器
```

## 日志记录规范

### 日志字段

| 字段名 | 类型 | 描述 | 示例 |
|--------|------|------|------|
| nat_server | string | NAT 服务器操作类型 | "configure", "cleanup" |
| swIfIndex | uint32 | VPP 接口索引 | 1, 2 |
| role | string | NAT 接口角色 | "inside", "outside" |
| publicIP | string | 公网 IP 地址 | "192.168.1.100" |
| retval | int32 | VPP API 返回值 | 0, -1 |

### 日志示例

```go
log.WithFields(logrus.Fields{
    "nat_server": "configure",
    "swIfIndex":  swIfIndex,
    "role":       "inside",
}).Info("NAT inside 接口配置成功")

log.WithFields(logrus.Fields{
    "nat_server": "configure",
    "publicIP":   publicIP.String(),
}).Info("NAT 地址池配置成功")

log.WithFields(logrus.Fields{
    "nat_server": "configure",
    "mapping":    fmt.Sprintf("%s:%d -> %s:%d", publicIP, publicPort, internalIP, internalPort),
}).Info("NAT 静态端口映射配置成功")
```

## 测试策略

### 单元测试（可选）

由于项目不强制要求单元测试，建议使用 VPP CLI 验证和端到端测试代替。

### VPP CLI 验证

每个 API 调用后使用相应的 VPP CLI 命令验证配置是否生效。

### 端到端测试

- **P1.2**: 验证接口角色配置成功（`show nat44 interfaces`）
- **P1.3**: 验证 NAT 地址转换功能（NSC ping 外部服务器，检查会话表）
- **P4**: 验证静态端口映射功能（外部客户端访问公网 IP:端口）

## 参考资料

### VPP 官方文档

- **NAT44 ED 插件文档**: https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- **NAT44 ED CLI 参考**: https://s3-docs.fd.io/vpp/23.02/cli-reference/clis/clicmd_src_plugins_nat_nat44-ed.html
- **VPP NAT Wiki**: https://wiki.fd.io/view/VPP/NAT

### govpp 文档

- **govpp API 文档**: https://pkg.go.dev/go.fd.io/govpp
- **binapi 生成工具**: https://github.com/FDio/govpp/tree/master/cmd/binapi-generator

### 项目内部参考

- **Data Model**: [data-model.md](../data-model.md)
- **Research**: [research.md](../research.md)
- **Feature Spec**: [spec.md](../spec.md)

## 更新记录

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| 1.0 | 2025-01-15 | 初始版本，定义 3 个核心 API | Claude Code |

---

**下一步**: 查阅各 API 的详细文档以了解具体参数和使用示例。
