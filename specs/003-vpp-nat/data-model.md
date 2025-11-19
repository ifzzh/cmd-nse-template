# Data Model: VPP NAT 网络服务端点

**日期**: 2025-01-15
**版本**: 1.0
**输入**: [spec.md](./spec.md) + [research.md](./research.md)

## 概述

本文档定义 VPP NAT 网络服务端点的核心数据模型,包括实体定义、字段描述、关系图、验证规则和状态转换。数据模型基于 VPP NAT44 ED 插件和 NSM SDK v1.15.0-rc.1,遵循 ACL 防火墙项目的设计模式(80% 代码结构相似度)。

---

## 实体定义

### 1. NAT NSE(网络服务端点)

**描述**: 提供 NAT 功能的 NSM 端点,集成 VPP 连接、NAT 配置、会话管理器、接口映射。

**Go 类型**: `natServer` (internal/nat/server.go)

**字段**:

| 字段名 | Go 类型 | 描述 | 必填 | 默认值 |
|--------|---------|------|------|--------|
| vppConn | *govppapi.Connection | VPP 连接实例 | ✅ | - |
| config | *NATConfig | NAT 配置 | ✅ | - |
| logger | *logrus.Entry | 日志记录器 | ✅ | - |
| interfaceStates | genericsync.Map[string, *natInterfaceState] | 连接 ID 到 NAT 接口状态的映射 | ✅ | empty map |

**natInterfaceState 结构**:
```go
type natInterfaceState struct {
    swIfIndex      interface_types.InterfaceIndex // VPP 接口索引
    role           NATInterfaceRole                // inside/outside 角色
    configured     bool                            // NAT 功能是否已配置
}
```

**关系**:
- natServer **包含** 1 个 VPP 连接
- natServer **包含** 1 个 NAT 配置
- natServer **管理** 多个 NAT 接口状态
- natServer **实现** networkservice.NetworkServiceServer 接口

**验证规则**:
- vppConn 必须已连接且健康
- config 必须通过配置验证
- logger 必须非 nil
- interfaceStates 支持并发访问(使用 genericsync.Map)

**代码示例**:
```go
type natServer struct {
    vppConn         *govppapi.Connection
    config          *NATConfig
    interfaceStates genericsync.Map[string, *natInterfaceState]
}

func NewServer(vppConn *govppapi.Connection, config *NATConfig) networkservice.NetworkServiceServer {
    return &natServer{
        vppConn: vppConn,
        config:  config,
    }
}
```

---

### 2. NAT 配置

**描述**: 定义 NAT 转换行为的参数集合,包含公网 IP 地址池、端口范围、协议类型、静态映射规则、转换方向。

**Go 类型**: `NATConfig` (internal/config/config.go)

**字段**:

| 字段名 | Go 类型 | 描述 | 必填 | 默认值 | 验证规则 |
|--------|---------|------|------|--------|----------|
| PublicIPs | []net.IP | NAT 公网 IP 地址池 | ✅ | - | 至少 1 个有效 IPv4 |
| PortRange | PortRange | 端口范围配置 | ❌ | {1024, 65535} | Min < Max, Min ≥ 1024 |
| StaticMappings | []StaticMapping | 静态端口映射规则 | ❌ | [] | 协议、IP、端口验证 |
| VrfID | uint32 | VRF ID(虚拟路由转发表 ID) | ❌ | 0 | - |

**PortRange 子结构**:

| 字段名 | Go 类型 | 描述 | 默认值 | 验证规则 |
|--------|---------|------|--------|----------|
| Min | uint16 | 端口范围起始值 | 1024 | 1024-65535 |
| Max | uint16 | 端口范围结束值 | 65535 | > Min, ≤ 65535 |

**示例(YAML)**:
```yaml
# /etc/nat/config.yaml - NAT 配置文件

# 公网 IP 地址池(必填,用于 SNAT)
public_ips:
  - 192.168.1.100
  - 192.168.1.101

# 端口范围(可选,默认 1024-65535)
port_range:
  min: 10000
  max: 60000

# VRF ID(可选,默认 0)
vrf_id: 0

# 静态端口映射(可选,P4 阶段)
static_mappings:
  - protocol: tcp
    public_ip: 192.168.1.100
    public_port: 8080
    internal_ip: 172.16.1.10
    internal_port: 80
    tag: "web-server-mapping"
```

**验证规则**:
1. PublicIPs 不能为空
2. 每个 PublicIP 必须是有效的 IPv4 地址(`ip.To4() != nil`)
3. PortRange.Min ≥ 1024(警告:< 1024 可能与系统端口冲突)
4. PortRange.Max ≤ 65535
5. PortRange.Min < PortRange.Max
6. StaticMappings 中的 protocol 必须是 tcp/udp/icmp
7. StaticMappings 中的端口号必须 > 0
8. StaticMappings 中的 PublicIP 必须在 PublicIPs 中
9. StaticMappings 中的 InternalIP 必须是有效 IPv4

**配置加载代码示例**:
```go
func retrieveNATConfig(ctx context.Context, c *Config) error {
    raw, err := os.ReadFile(c.NATConfigPath)
    if err != nil {
        return errors.Wrap(err, "读取 NAT 配置文件失败")
    }

    var natCfg NATConfig
    if err := yaml.Unmarshal(raw, &natCfg); err != nil {
        return errors.Wrap(err, "解析 NAT 配置文件失败")
    }

    // 验证必填字段
    if len(natCfg.PublicIPs) == 0 {
        return errors.New("NAT 配置错误: public_ips 不能为空")
    }

    // 验证 IPv4 格式
    for i, ip := range natCfg.PublicIPs {
        if ip.To4() == nil {
            return errors.Errorf("NAT 配置错误: public_ips[%d] 不是有效的 IPv4 地址: %s", i, ip)
        }
    }

    // 验证端口范围
    if natCfg.PortRange.Min >= natCfg.PortRange.Max {
        return errors.Errorf("NAT 配置错误: port_range.min (%d) 必须小于 port_range.max (%d)",
            natCfg.PortRange.Min, natCfg.PortRange.Max)
    }

    c.NATConfig = natCfg
    return nil
}
```

---

### 3. 静态端口映射

**描述**: 固定的公网 IP:端口到内部 IP:端口的映射规则(P4 阶段)。

**Go 类型**: `StaticMapping` (internal/config/config.go)

**字段**:

| 字段名 | Go 类型 | 描述 | 必填 | 验证规则 |
|--------|---------|------|------|----------|
| Protocol | string | 协议类型 | ✅ | "tcp" \| "udp" \| "icmp" |
| PublicIP | net.IP | 公网 IP 地址 | ✅ | 必须在 PublicIPs 中 |
| PublicPort | uint16 | 公网端口 | ✅ | 1-65535 |
| InternalIP | net.IP | 内部服务器 IP | ✅ | 有效 IPv4 |
| InternalPort | uint16 | 内部端口 | ✅ | 1-65535 |
| Tag | string | 描述标签 | ❌ | 最长 64 字符 |

**示例**:
```go
StaticMapping{
    Protocol:     "tcp",
    PublicIP:     net.ParseIP("192.168.1.100"),
    PublicPort:   8080,
    InternalIP:   net.ParseIP("10.0.0.5"),
    InternalPort: 80,
    Tag:          "web-server-mapping",
}
```

**VPP API 调用**:
```go
func configureStaticMapping(ctx context.Context, vppConn *govppapi.Connection, mapping StaticMapping) error {
    client := nat44_ed.NewServiceClient(vppConn)

    // 协议转换
    var protocol uint8
    switch mapping.Protocol {
    case "tcp":
        protocol = 6
    case "udp":
        protocol = 17
    case "icmp":
        protocol = 1
    }

    _, err := client.Nat44AddDelStaticMapping(ctx, &nat44_ed.Nat44AddDelStaticMapping{
        IsAdd:             true,
        LocalIPAddress:    mapping.InternalIP.To4(),
        ExternalIPAddress: mapping.PublicIP.To4(),
        Protocol:          protocol,
        LocalPort:         mapping.InternalPort,
        ExternalPort:      mapping.PublicPort,
        Tag:               mapping.Tag,
    })

    return err
}
```

---

### 4. NAT 会话

**描述**: 记录单个网络流的地址转换映射,包含内部 IP/端口、公网 IP/端口、协议类型、会话状态、超时时间。

**注意**: NAT 会话由 VPP NAT44 ED 插件管理,Go 代码不直接操作会话表。

**VPP 内部表示**: NAT44 Session Table

**关键信息**:
- 内部地址(inside address):NSC 的 IP:端口
- 外部地址(outside address):公网 IP:端口
- 协议类型:TCP/UDP/ICMP
- 会话状态:ESTABLISHED/TRANSITORY
- 超时时间:TCP 7200s, UDP 300s, ICMP 60s

**查询方式**: VPP CLI `show nat44 sessions`

**示例输出**:
```
NAT44 ED sessions:
  i2o 10.0.0.10:12345 -> 192.168.1.100:54321 [protocol TCP]
      external: 8.8.8.8:80
      state: ESTABLISHED
      timeout: 7200s
```

**会话生命周期**:
1. **创建**:NSC 发送首个数据包到 NAT NSE(inside 接口)
2. **转换**:VPP NAT44 ED 从地址池分配公网 IP:端口
3. **记录**:插入会话表,记录(内部 IP:端口 ↔ 公网 IP:端口)映射
4. **反向转换**:外部服务器响应时,查询会话表,反向转换目标地址
5. **超时**:会话空闲超过超时时间,自动删除
6. **手动删除**:NSC 断开连接时,NAT NSE 通过接口禁用触发会话清理

---

### 5. VPP 接口

**描述**: NSM 在 VPP 中创建的虚拟网络接口(memif/kernel),配置为 NAT inside(内部网络)或 outside(外部网络)。

**Go 类型**: `natInterfaceState` (internal/nat/common.go)

**字段**:

| 字段名 | Go 类型 | 描述 | 必填 |
|--------|---------|------|------|
| swIfIndex | interface_types.InterfaceIndex | VPP 接口索引 | ✅ |
| role | NATInterfaceRole | inside/outside 角色 | ✅ |
| configured | bool | NAT 功能是否已配置 | ✅ |

**NATInterfaceRole 枚举**:
```go
type NATInterfaceRole string

const (
    NATRoleInside  NATInterfaceRole = "inside"  // 内部网络侧(NSC 端)
    NATRoleOutside NATInterfaceRole = "outside" // 外部网络侧(下游 NSE/外网端)
)
```

**VPP 配置**:
- Inside 接口:启用 `nat44-ed-in2out` feature(出站转换)
- Outside 接口:启用 `nat44-ed-out2in` feature(入站反向转换)

**配置代码示例**:
```go
func configureNATInterface(ctx context.Context, vppConn *govppapi.Connection, swIfIndex interface_types.InterfaceIndex, role NATInterfaceRole) error {
    client := nat44_ed.NewServiceClient(vppConn)

    var flags nat_types.NatConfigFlags
    if role == NATRoleInside {
        flags = nat_types.NAT_IS_INSIDE
    } else {
        flags = nat_types.NAT_IS_OUTSIDE
    }

    _, err := client.Nat44InterfaceAddDelFeature(ctx, &nat44_ed.Nat44InterfaceAddDelFeature{
        IsAdd:     true,
        Flags:     flags,
        SwIfIndex: swIfIndex,
    })

    return err
}
```

**VPP CLI 验证**:
```bash
vppctl show nat44 interfaces
# 预期输出:
# NAT44 interfaces:
#   memif0/0 in   (inside 接口)
#   memif0/1 out  (outside 接口)
```

---

### 6. 本地化 NAT 模块

**描述**: 从 govpp binapi 复制到 internal/ 的 NAT 类型定义和 API 绑定,包含 nat_types、nat44_ed、版本信息、依赖关系。

**模块列表**:

| 模块名 | 源路径 | 本地化路径 | VPP 版本 | 依赖 |
|--------|--------|-----------|----------|------|
| nat_types | go.fd.io/govpp/binapi/nat_types | internal/binapi_nat_types | VPP 23.10-rc0~170 | - |
| nat44_ed | go.fd.io/govpp/binapi/nat44_ed | internal/binapi_nat44_ed | VPP 23.10-rc0~170 | nat_types |

**go.mod 配置**:
```go
module github.com/ifzzh/cmd-nse-template

require (
    go.fd.io/govpp v0.0.0-20240328101142-8a444680fbba
)

replace (
    go.fd.io/govpp/binapi/nat_types => ./internal/binapi_nat_types
    go.fd.io/govpp/binapi/nat44_ed => ./internal/binapi_nat44_ed
)
```

**本地化流程**(P3 阶段):
1. **P3.1**(v1.1.0):本地化 nat_types
   - 复制 `binapi/nat_types/` 到 `internal/binapi_nat_types/`
   - 创建 `internal/binapi_nat_types/go.mod`
   - 添加 go.mod replace 指令
   - 添加中文注释
   - 验证:编译 → Docker 镜像 → K8s 部署测试

2. **P3.2**(v1.1.1):本地化 nat44_ed
   - 复制 `binapi/nat44_ed/` 到 `internal/binapi_nat44_ed/`
   - 创建 `internal/binapi_nat44_ed/go.mod`(依赖本地化的 nat_types)
   - 添加 go.mod replace 指令
   - 添加中文注释
   - 验证:编译 → Docker 镜像 → K8s 部署测试

**验证规则**:
- 本地化代码与在线版本完全一致(仅添加中文注释)
- 项目编译成功,所有依赖正确解析
- Docker 镜像构建成功
- K8s 部署测试通过,无功能回归
- 任一步骤失败立即回退

---

### 7. 容器镜像

**描述**: 包含 NAT NSE 代码和依赖的可部署软件包。

**镜像命名**: `ifzzh520/vpp-nat44-nat`
- `vpp`:使用 VPP 技术
- `nat44`:VPP 插件名称
- `nat`:网络服务名称

**版本标签**:

| 阶段 | 版本号 | 功能 | Git 标签 |
|------|--------|------|----------|
| Baseline | v1.0.0 | ACL 防火墙(转型前) | v1.0.0-acl-final |
| P1.1 | v1.0.1 | NAT 框架创建 | v1.0.1 |
| P1.2 | v1.0.2 | 接口角色配置 | v1.0.2 |
| P1.3 | v1.0.3 | 地址池配置与集成 | v1.0.3 |
| P2 | v1.0.4 | NAT 配置管理 | v1.0.4 |
| P3.1 | v1.1.0 | 本地化 nat_types | v1.1.0 |
| P3.2 | v1.1.1 | 本地化 nat44_ed | v1.1.1 |
| P4 | v1.2.0 | 静态端口映射 | v1.2.0 |
| P5 | v1.3.0 | 删除 ACL 遗留代码 | v1.3.0 |

**镜像内容**:
- NAT NSE 二进制文件(/bin/cmd-nse-template)
- VPP v24.10.0 运行时
- 本地化 NAT 模块(P3 阶段后)
- NSM SDK v1.15.0-rc.1
- SPIRE agent 集成

**镜像构建**:
```bash
# Dockerfile target
docker build --target runtime -t ifzzh520/vpp-nat44-nat:v1.0.1 .
```

**K8s 部署引用**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nse-nat-vpp
spec:
  template:
    spec:
      containers:
      - name: nat
        image: ifzzh520/vpp-nat44-nat:v1.0.1
```

---

## 实体关系图

```
┌──────────────────────────────────────────────────────────────┐
│                        NAT NSE                                │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ natServer                                               │ │
│  │  - vppConn: *govppapi.Connection                       │ │
│  │  - config: *NATConfig                                  │ │
│  │  - logger: *logrus.Entry                               │ │
│  │  - interfaceStates: Map[string, *natInterfaceState]   │ │
│  └─────────────────────────────────────────────────────────┘ │
│           │                │                │                 │
│           ▼                ▼                ▼                 │
│  ┌────────────┐  ┌─────────────────┐  ┌─────────────────┐   │
│  │ VPP Conn   │  │ NATConfig       │  │ natInterfaceState│   │
│  │ (govpp)    │  │ - PublicIPs     │  │ - swIfIndex      │   │
│  └────────────┘  │ - PortRange     │  │ - role           │   │
│                  │ - StaticMappings│  │ - configured     │   │
│                  └─────────────────┘  └─────────────────┘   │
│                          │                      │             │
│                          ▼                      ▼             │
│                  ┌─────────────────┐  ┌─────────────────┐   │
│                  │ StaticMapping   │  │ VPP NAT44 ED    │   │
│                  │ - Protocol      │  │ - Session Table │   │
│                  │ - PublicIP      │  │ - Address Pool  │   │
│                  │ - PublicPort    │  │ - Interface     │   │
│                  │ - InternalIP    │  │   Features      │   │
│                  │ - InternalPort  │  │ - in2out/out2in │   │
│                  │ - Tag           │  └─────────────────┘   │
│                  └─────────────────┘                         │
└──────────────────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────┐
│ 容器镜像                              │
│ ifzzh520/vpp-nat44-nat:v1.0.x-v1.3.0│
│ - NAT NSE 代码                        │
│ - 本地化 binapi 模块(P3+)            │
│ - VPP v24.10.0                       │
│ - NSM SDK v1.15.0-rc.1               │
│ - SPIRE agent                        │
└──────────────────────────────────────┘
```

**关系说明**:
1. NAT NSE 包含 1 个 VPP 连接、1 个 NAT 配置、N 个接口状态
2. NAT 配置包含 M 个公网 IP、1 个端口范围、K 个静态映射
3. 每个 NSC 连接创建 2 个接口状态(inside + outside)
4. VPP NAT44 ED 管理会话表和地址池(不暴露给 Go 层)
5. 容器镜像包含所有运行时依赖

---

## 状态转换

### NAT NSE 生命周期

```
┌──────────┐  Request()  ┌───────────────┐  NSC Disconnect  ┌───────────┐
│ Created  │─────────────>│ NAT Enabled   │─────────────────>│ Cleaned   │
└──────────┘              └───────────────┘                  └───────────┘
                                │ △
                                │ │ VPP API 调用失败
                                ▼ │
                          ┌───────────────┐
                          │ Error State   │
                          └───────────────┘
                                │
                                │ postponeCtxFunc() 清理
                                ▼
                          ┌───────────────┐
                          │ Rollback      │
                          │ (Close 调用)  │
                          └───────────────┘
```

**状态描述**:
1. **Created**:natServer 实例创建,VPP 连接已建立
2. **NAT Enabled**:NAT 接口配置成功,地址池配置成功,VPP 开始处理 NAT 转换
3. **Cleaned**:NSC 断开连接,NAT 接口禁用,会话清理
4. **Error State**:VPP API 调用失败(接口配置失败、地址池配置失败),记录错误日志,返回错误给 NSM
5. **Rollback**:postponeCtxFunc() 触发清理,删除已配置的 NAT 资源

**状态转换触发条件**:
- Created → NAT Enabled:`configureNATInterface()` + `configureNATAddressPool()` 成功
- NAT Enabled → Cleaned:NSM 调用 `Close()`,`disableNATInterface()` 成功
- NAT Enabled → Error State:VPP API 调用返回错误
- Error State → Rollback:`postponeCtxFunc()` 执行延迟清理

### NAT 接口配置流程

```
┌────────────────┐  NSM 创建接口  ┌────────────────┐
│ Interface      │───────────────>│ Interface      │
│ Not Exist      │                │ Created by NSM │
└────────────────┘                └────────────────┘
                                         │
                                         │ configureNATInterface()
                                         ▼
                                  ┌───────────────────┐
                                  │ NAT Feature       │
                                  │ Enabled (in/out)  │
                                  └───────────────────┘
                                         │
                                         │ NSC 数据流
                                         ▼
                                  ┌───────────────────┐
                                  │ NAT Translating   │
                                  │ (会话活跃)        │
                                  └───────────────────┘
                                         │
                                         │ NSC Disconnect
                                         ▼
                                  ┌───────────────────┐
                                  │ NAT Feature       │
                                  │ Disabled          │
                                  └───────────────────┘
                                         │
                                         │ NSM 删除接口
                                         ▼
                                  ┌────────────────┐
                                  │ Interface      │
                                  │ Deleted        │
                                  └────────────────┘
```

**配置步骤**:
1. NSM 调用 `Request()`,创建 VPP 接口(memif/kernel)
2. natServer 获取接口索引(`ifindex.Load(ctx, isClient)`)
3. 调用 `configureNATInterface()` 配置 inside/outside 角色
4. VPP NAT44 ED 启用接口 feature(in2out 或 out2in)
5. 接口进入 NAT Translating 状态,处理数据流
6. NSM 调用 `Close()`,natServer 调用 `disableNATInterface()`
7. VPP NAT44 ED 禁用接口 feature,清理会话
8. NSM 删除 VPP 接口

---

## 数据流

### SNAT 出站流程(P1)

```
┌─────┐  src:10.0.0.10:12345  ┌──────────────┐  src:192.168.1.100:54321  ┌─────────┐
│ NSC │───────────────────────>│ NAT NSE      │───────────────────────────>│ 外部    │
└─────┘                         │ (VPP NAT44)  │                            │ 服务器  │
                                │              │<───────────────────────────│ 8.8.8.8 │
                                │              │  dst:192.168.1.100:54321   └─────────┘
                                └──────────────┘
                                      │ △
                                      │ │ VPP NAT44 ED 自动处理
                                      ▼ │ - 会话创建
                                ┌──────────────┐ - 地址转换
                                │ Session Table│ - 反向转换
                                │ 10.0.0.10:   │
                                │ 12345 <->    │
                                │ 192.168.1.   │
                                │ 100:54321    │
                                └──────────────┘
```

**步骤**:
1. NSC 发送数据包(src: 10.0.0.10:12345, dst: 8.8.8.8:80)
2. VPP 接收数据包在 inside 接口(memif0/0)
3. NAT44 ED 插件检查会话表,未找到 → 创建新会话
4. 从地址池分配公网 IP:端口(192.168.1.100:54321)
5. 创建 NAT 会话记录(10.0.0.10:12345 ↔ 192.168.1.100:54321)
6. 修改数据包源地址为 192.168.1.100:54321
7. 通过 xconnect 转发数据包到 outside 接口(memif0/1)
8. 外部服务器接收数据包,回复响应(dst: 192.168.1.100:54321)
9. VPP 接收响应在 outside 接口
10. NAT44 ED 插件查询会话表,找到映射 192.168.1.100:54321 → 10.0.0.10:12345
11. 修改数据包目标地址为 10.0.0.10:12345
12. 通过 xconnect 转发响应到 inside 接口
13. NSC 收到响应

### 静态端口映射流程(P4)

```
┌─────────┐  dst:192.168.1.100:8080  ┌──────────────┐  dst:10.0.0.5:80  ┌─────────────┐
│ 外部    │─────────────────────────>│ NAT NSE      │─────────────────>│ 内部服务器  │
│ 客户端  │                           │ (VPP NAT44)  │                   │ (via NSC)   │
└─────────┘                           │              │<─────────────────│ 10.0.0.5:80 │
                                      │              │  src:10.0.0.5:80  └─────────────┘
                                      └──────────────┘
                                            │ △
                                            │ │ 静态映射规则
                                            ▼ │ 192.168.1.100:8080
                                      ┌──────────────┐ -> 10.0.0.5:80
                                      │ Static       │
                                      │ Mappings     │
                                      └──────────────┘
```

**步骤**:
1. 外部客户端发送请求(dst: 192.168.1.100:8080)
2. VPP 接收数据包在 outside 接口
3. NAT44 ED 插件查询静态映射表,找到规则:192.168.1.100:8080 → 10.0.0.5:80
4. 修改数据包目标地址为 10.0.0.5:80
5. 通过 xconnect 转发数据包到 inside 接口
6. NSC(内部服务器)接收请求,生成响应(src: 10.0.0.5:80)
7. VPP 接收响应在 inside 接口
8. NAT44 ED 插件根据静态映射反向转换,修改源地址为 192.168.1.100:8080
9. 通过 xconnect 转发响应到 outside 接口
10. 外部客户端收到响应

---

## 配置验证

### 验证流程

```
┌─────────────┐  加载配置  ┌─────────────┐  验证  ┌─────────────┐  启动
│ YAML 文件   │─────────>│ NATConfig   │────────>│ 验证通过    │─────────> NAT NSE
│ config.yaml │           └─────────────┘         └─────────────┘
└─────────────┘                  │
                                 │ 验证失败
                                 ▼
                          ┌─────────────┐  记录错误日志
                          │ 拒绝启动    │──────────────────> 程序退出
                          └─────────────┘  (< 5 秒)
```

### 验证规则

| 验证项 | 规则 | 错误级别 | 错误信息 |
|--------|------|----------|----------|
| PublicIPs 非空 | len(PublicIPs) > 0 | 致命 | "NAT 配置错误:公网 IP 地址池不能为空" |
| PublicIPs IPv4 格式 | 每个 IP 必须是有效 IPv4 | 致命 | "NAT 配置错误:public_ips[%d] 不是有效的 IPv4 地址: %s" |
| 端口范围下限 | PortRange.Min ≥ 1024 | 警告 | "NAT 配置警告:port_range.min < 1024 可能与系统端口冲突" |
| 端口范围上限 | PortRange.Max ≤ 65535 | 致命 | "NAT 配置错误:port_range.max 超出范围" |
| 端口范围顺序 | PortRange.Min < PortRange.Max | 致命 | "NAT 配置错误:port_range.min 必须小于 port_range.max" |
| 静态映射协议 | Protocol in {tcp,udp,icmp} | 致命 | "NAT 配置错误:static_mappings[%d].protocol 必须是 tcp/udp/icmp" |
| 静态映射端口 | Port > 0 | 致命 | "NAT 配置错误:static_mappings[%d] 端口不能为 0" |
| 静态映射 PublicIP | ExternalIP in PublicIPs | 致命 | "NAT 配置错误:static_mappings[%d].public_ip 不在地址池中" |
| 静态映射 InternalIP | InternalIP 是有效 IPv4 | 致命 | "NAT 配置错误:static_mappings[%d].internal_ip 格式无效" |
| 配置文件缺失 | 文件存在 | 致命 | "读取 NAT 配置文件失败: %v" |
| YAML 解析错误 | 合法 YAML | 致命 | "解析 NAT 配置文件失败: %v" |

---

## 总结

本数据模型定义了 VPP NAT 网络服务端点的 7 个核心实体:
1. **NAT NSE(网络服务端点)**:集成 VPP 连接、配置、接口状态管理
2. **NAT 配置**:公网 IP 地址池、端口范围、静态映射规则
3. **静态端口映射**:固定的公网 IP:端口到内部 IP:端口映射(P4)
4. **NAT 会话**:VPP NAT44 ED 管理的地址转换映射(不暴露给 Go 层)
5. **VPP 接口**:配置为 inside/outside 角色的虚拟网络接口
6. **本地化 NAT 模块**:nat_types 和 nat44_ed 模块本地化(P3)
7. **容器镜像**:包含 NAT NSE 代码和依赖的可部署软件包

模型遵循以下设计原则:
- **简洁性**:实体数量最小化,字段仅保留必需项
- **一致性**:命名和结构与 ACL 防火墙保持 80% 相似(代码复用率高)
- **可验证性**:每个实体都有明确的验证规则和错误信息
- **可追溯性**:每个实体都有清晰的状态转换和数据流
- **安全性**:配置验证拒绝无效输入,VPP API 调用失败时自动回滚

**实现优势**:
- NAT 实现比 ACL 更简单(无需双向规则交换)
- VPP NAT44 ED 自动管理会话表、端口分配、超时
- 配置管理、测试流程、K8s 部署方式完全继承 ACL 项目
- 净代码量增加仅 4.2%(+15 行 vs ACL 353 行)

**下一步**:基于此数据模型生成 API contracts(VPP Binary API 调用)和 quickstart.md。
