# VPP NAT44 NSE 实施方案

**创建时间**: 2025-01-13
**状态**: 待审核
**目的**: 整理技术调研结论和架构决策，形成完整实施方案供用户审核

---

## 1. 核心问题与决策

### 问题 1：是否应该取消 ACL 中的双向规则？

**结论：是的，NAT 不需要 ACL 的双向规则模式。**

**理由**：
1. **ACL 的双向规则实现**：
   - ACL 创建两组规则：ingress（入站）和 egress（出站）
   - 出站规则通过交换 src/dst IP 和端口实现双向过滤
   - 参考 `internal/acl/common.go:169-182`

2. **NAT 的工作机制不同**：
   - VPP NAT44 ED 通过 **inside/outside 接口角色** 自动处理双向转换
   - NAT 维护 **会话表**（session table），自动反向转换响应流量
   - 无需手动创建反向规则或交换 src/dst

3. **具体差异**：
   ```
   ACL 模式（复杂）：
   - 入站规则：permit src=NSC, dst=外部
   - 出站规则：permit src=外部, dst=NSC（手动交换）

   NAT 模式（简单）：
   - 配置接口 A 为 inside（内部）
   - 配置接口 B 为 outside（外部）
   - VPP 自动处理正向和反向转换
   ```

**实施影响**：
- 删除 `aclAdd()` 函数中的 src/dst 交换逻辑
- 不调用两次 `addACLToACLList()`（一次 ingress，一次 egress）
- 改为调用 VPP NAT44 API 一次性配置接口角色

---

### 问题 2：NAT 插件本地化策略

**结论：完全遵循 ACL 本地化模式，最小化修改。**

**本地化范围**：
1. **govpp binapi nat_types** → `internal/binapi_nat_types/`
2. **govpp binapi nat44_ed** → `internal/binapi_nat44_ed/`

**不本地化**（保持在线引用）：
- govpp 核心库（adapter、api、codec 等）
- NSM SDK（networkservicemesh/sdk、sdk-vpp）
- 其他非 NAT 相关依赖

**本地化步骤**（参考 ACL 本地化流程）：
1. **复制源代码**：
   ```bash
   # 从 Go 模块缓存复制到 internal/
   cp -r $GOPATH/pkg/mod/github.com/networkservicemesh/govpp@版本/binapi/nat_types internal/binapi_nat_types
   cp -r $GOPATH/pkg/mod/github.com/networkservicemesh/govpp@版本/binapi/nat44_ed internal/binapi_nat44_ed
   ```

2. **创建模块 go.mod**：
   ```go
   // internal/binapi_nat_types/go.mod
   module github.com/networkservicemesh/govpp/binapi/nat_types

   go 1.23

   require (
       go.fd.io/govpp v0.11.0
       github.com/networkservicemesh/govpp/binapi/interface_types v本地版本
   )
   ```

3. **配置 replace 指令**（在项目根 go.mod）：
   ```go
   replace (
       github.com/networkservicemesh/govpp/binapi/nat_types => ./internal/binapi_nat_types
       github.com/networkservicemesh/govpp/binapi/nat44_ed => ./internal/binapi_nat44_ed
   )
   ```

4. **最小化修改原则**：
   - ✅ 允许：添加中文注释
   - ✅ 允许：修复 go.mod 依赖路径
   - ❌ 禁止：修改业务逻辑
   - ❌ 禁止：修改 API 签名

5. **版本管理**：
   ```
   基线镜像：ifzzh520/vpp-nat44-nat:v1.0.0
   第一次本地化（nat_types）：v1.1.0
   第二次本地化（nat44_ed）：v1.1.1
   ```

6. **验证流程**：
   ```bash
   # 每次本地化后
   go mod tidy && go build ./...  # 编译验证
   docker build -t ifzzh520/vpp-nat44-nat:v1.1.x .  # 构建镜像
   # 部署到 K8s 测试环境，运行功能测试
   ```

**参考文件**：
- `specs/002-acl-localization/spec.md`（ACL 本地化规范）
- `internal/acl/`（ACL 本地化示例）

---

## 2. 架构决策

### 2.1 总体架构：基于 VPP NAT44 ED 官方实现

**选择理由**：
| 对比维度 | VPP NAT44 官方 | 自研实现 |
|---------|---------------|----------|
| **实现难度** | 2/5 | 5/5 |
| **代码量** | ~50 行 | ~500-1000 行 |
| **会话管理** | VPP 自动管理 | 需手动实现会话表、超时、端口分配 |
| **性能** | VPP 数据面处理，高性能 | 控制面处理，性能较低 |
| **维护成本** | 低，依赖 VPP 稳定性 | 高，需自行维护复杂逻辑 |
| **功能完整性** | 支持 SNAT/DNAT/端口映射/协议全覆盖 | 需逐个实现，易遗漏边界情况 |
| **开发周期** | 1-2 周 | 1-2 个月 |
| **风险评估** | 低 | 高 |

**决策：使用 VPP NAT44 ED 官方实现。**

---

### 2.2 Service Function Chaining 架构

**NSE 在 SFC 中的位置**：
```
NSC（内网客户端）→ NAT NSE → 下游 NSE（可选）→ 外网
```

**双接口架构**（基于现有 ACL 实现）：
```
┌─────────────────────────────────────────┐
│            NAT NSE Pod                  │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │       Server 端链式处理          │   │
│  │  (Interface A - NAT inside)     │   │
│  │                                 │   │
│  │  1. recvfd/sendfd              │   │
│  │  2. up.NewServer (创建接口)     │   │
│  │  3. xconnect (L2 桥接)         │   │
│  │  4. nat.NewServer (配置 NAT)   │◀──── NSC 连接（内网流量）
│  │  5. mechanisms (memif/kernel)  │   │
│  └────────────┬────────────────────┘   │
│               │ xconnect 桥接           │
│               ▼                         │
│  ┌────────────┴────────────────────┐   │
│  │       Client 端链式处理          │   │
│  │  (Interface B - NAT outside)    │   │
│  │                                 │   │
│  │  1. metadata.NewClient         │   │
│  │  2. up.NewClient (创建接口)     │   │
│  │  3. xconnect (L2 桥接)         │   │
│  │  4. memif/kernel               │   │
│  │  5. sendfd/recvfd              │───▶ 下游 NSE 或外网
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

**接口角色配置**：
- **Interface A**（server 端）：配置为 NAT **inside** 接口（内部网络）
- **Interface B**（client 端）：配置为 NAT **outside** 接口（外部网络）

**数据流路径**：
1. **出站流量**（NSC → 外网）：
   ```
   NSC → Interface A (inside) → SNAT 转换（修改源 IP/端口）→ xconnect → Interface B (outside) → 外网
   ```

2. **入站流量**（外网 → NSC）：
   ```
   外网 → Interface B (outside) → 反向 SNAT 转换（恢复目的 IP/端口）→ xconnect → Interface A (inside) → NSC
   ```

**关键发现**（基于代码分析）：
- `main.go:210-248` 显示两个独立链式处理：server 端和 client 端
- `xconnect.NewServer()` 和 `xconnect.NewClient()` 在 VPP 中创建 L2 桥接
- ACL 在 server 端应用，NAT 同样应在 server 端应用（但配置接口角色）

---

### 2.3 NAT 功能映射

**P1：基础 SNAT（最小可交付）**：
- 出站流量：NSC → 外网（修改源 IP/端口）
- 入站流量：外网响应 → NSC（VPP 自动反向转换）
- 实现方式：配置 Interface A = inside, Interface B = outside

**P4：静态端口映射（DNAT，可选高级功能）**：
- 出站流量：外部主动连接 → 内部服务（修改目的 IP/端口）
- 入站流量：内部响应 → 外部（VPP 自动反向转换）
- 实现方式：VPP API `nat44_add_del_static_mapping`

**两者关系**：
- SNAT 反向路径 ≠ DNAT 正向路径
- SNAT 反向：基于会话表的动态反向转换（响应流量）
- DNAT 正向：基于静态映射的主动连接转发（新建流量）
- 可共存，互不冲突

---

## 3. 实施阶段

### 阶段 1：P1 - 基础 SNAT 实现

**目标**：实现最小可用的 NAT 功能，验证架构正确性。

**步骤**：
1. **创建 internal/nat/ 模块**（参考 internal/acl/）：
   ```
   internal/nat/
   ├── server.go      # 实现 networkservice.NetworkServiceServer
   ├── common.go      # NAT 配置和接口管理
   └── config.go      # 配置结构体（可选，P2 再实现）
   ```

2. **实现 natServer 结构体**（参考 `internal/acl/server.go`）：
   ```go
   type natServer struct {
       vppConn    api.Connection
       publicIP   string  // NAT 公网 IP（硬编码或从环境变量读取）
       portRange  [2]uint16  // 端口范围（如 1024-65535）
       natIndices genericsync.Map[string, uint32]  // 连接 ID -> NAT 配置索引
   }

   func NewServer(vppConn api.Connection, publicIP string, portRange [2]uint16) networkservice.NetworkServiceServer {
       return &natServer{
           vppConn:   vppConn,
           publicIP:  publicIP,
           portRange: portRange,
       }
   }
   ```

3. **实现 Request() 方法**：
   ```go
   func (n *natServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
       // 1. 调用链中的下一个服务器（创建接口）
       conn, err := next.Server(ctx).Request(ctx, request)
       if err != nil {
           return nil, err
       }

       // 2. 检查是否已配置 NAT
       _, loaded := n.natIndices.Load(conn.GetId())
       if !loaded {
           // 3. 获取接口索引
           swIfIndex, ok := ifindex.Load(ctx, metadata.IsClient(n))
           if !ok {
               return nil, errors.New("未找到软件接口索引")
           }

           // 4. 配置 NAT inside/outside 接口
           isInside := !metadata.IsClient(n)  // server 端 = inside
           if err := configureNATInterface(ctx, n.vppConn, swIfIndex, isInside); err != nil {
               // 清理连接
               closeCtx, cancelClose := postpone.ContextWithValues(ctx)()
               defer cancelClose()
               n.Close(closeCtx, conn)
               return nil, err
           }

           // 5. 配置 NAT 地址池（仅在 outside 接口首次配置）
           if !isInside {
               if err := configureNATAddressPool(ctx, n.vppConn, n.publicIP, n.portRange); err != nil {
                   // 清理
                   return nil, err
               }
           }

           // 6. 记录配置索引
           n.natIndices.Store(conn.GetId(), swIfIndex)
       }

       return conn, nil
   }
   ```

4. **实现 Close() 方法**：
   ```go
   func (n *natServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
       // 1. 加载并删除 NAT 配置索引
       swIfIndex, loaded := n.natIndices.LoadAndDelete(conn.GetId())

       if loaded {
           // 2. 禁用接口的 NAT 功能
           if err := disableNATInterface(ctx, n.vppConn, swIfIndex); err != nil {
               log.FromContext(ctx).WithError(err).Debug("禁用 NAT 接口失败")
           }
       }

       // 3. 调用链中的下一个服务器
       return next.Server(ctx).Close(ctx, conn)
   }
   ```

5. **实现 NAT 配置辅助函数**（`internal/nat/common.go`）：
   ```go
   // 配置接口为 NAT inside 或 outside
   func configureNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex uint32, isInside bool) error {
       client := nat44_ed.NewServiceClient(vppConn)

       req := &nat44_ed.Nat44InterfaceAddDelFeature{
           SwIfIndex: interface_types.InterfaceIndex(swIfIndex),
           IsAdd:     true,
           Flags:     nat_types.NAT_IS_INSIDE,  // 如果是 outside，则为 0
       }
       if !isInside {
           req.Flags = 0
       }

       _, err := client.Nat44InterfaceAddDelFeature(ctx, req)
       return err
   }

   // 配置 NAT 地址池
   func configureNATAddressPool(ctx context.Context, vppConn api.Connection, publicIP string, portRange [2]uint16) error {
       client := nat44_ed.NewServiceClient(vppConn)

       // 解析 IP 地址
       ip := net.ParseIP(publicIP).To4()
       if ip == nil {
           return errors.New("无效的 IPv4 地址")
       }

       req := &nat44_ed.Nat44AddDelAddressRange{
           FirstIPAddress: ip,
           LastIPAddress:  ip,  // 单个 IP
           VrfID:          0,
           IsAdd:          true,
           Flags:          0,
       }

       _, err := client.Nat44AddDelAddressRange(ctx, req)
       return err
   }

   // 禁用接口 NAT 功能
   func disableNATInterface(ctx context.Context, vppConn api.Connection, swIfIndex uint32) error {
       client := nat44_ed.NewServiceClient(vppConn)

       req := &nat44_ed.Nat44InterfaceAddDelFeature{
           SwIfIndex: interface_types.InterfaceIndex(swIfIndex),
           IsAdd:     false,
           Flags:     nat_types.NAT_IS_INSIDE,
       }

       _, err := client.Nat44InterfaceAddDelFeature(ctx, req)
       return err
   }
   ```

6. **集成到 main.go**：
   ```go
   // 导入
   import "github.com/networkservicemesh/cmd-nse-firewall-vpp/internal/nat"

   // 在 endpoint.NewServer() 的 server 端链中添加（替换或并列 ACL）
   firewallEndpoint.Endpoint = endpoint.NewServer(ctx,
       endpoint.WithAdditionalFunctionality(
           recvfd.NewServer(),
           sendfd.NewServer(),
           up.NewServer(ctx, vppConn),
           clienturl.NewServer(&config.ConnectTo),
           xconnect.NewServer(vppConn),
           nat.NewServer(vppConn, config.NATPublicIP, [2]uint16{1024, 65535}),  // ← 添加 NAT
           mechanisms.NewServer(...),
           connect.NewServer(...),
       ))
   ```

7. **验证**：
   ```bash
   # 编译
   go build -o cmd-nse-firewall-vpp .

   # 构建镜像
   docker build -t ifzzh520/vpp-nat44-nat:v1.0.0 .

   # 部署到 K8s
   kubectl apply -f deployments/nse-nat.yaml

   # 测试
   # 从 NSC 发起 ping 或 curl，验证源 IP 被转换为 NAT 公网 IP
   ```

**关键差异（vs ACL）**：
- ❌ 不调用 `ACLAddReplace`（创建规则列表）
- ❌ 不调用 `ACLInterfaceSetACLList`（应用规则到接口）
- ❌ 不交换 src/dst（无需双向规则）
- ✅ 调用 `Nat44InterfaceAddDelFeature`（配置接口角色）
- ✅ 调用 `Nat44AddDelAddressRange`（配置地址池）
- ✅ VPP 自动管理会话表

---

### 阶段 2：P2 - 配置管理

**目标**：支持通过配置文件或环境变量定义 NAT 参数。

**步骤**：
1. 定义配置结构体：
   ```go
   type NATConfig struct {
       PublicIP      string   `yaml:"publicIP"`
       PortRangeMin  uint16   `yaml:"portRangeMin"`
       PortRangeMax  uint16   `yaml:"portRangeMax"`
       EnableTCP     bool     `yaml:"enableTCP"`
       EnableUDP     bool     `yaml:"enableUDP"`
       EnableICMP    bool     `yaml:"enableICMP"`
   }
   ```

2. 加载配置（参考 ACL 配置加载逻辑）

3. 配置验证（启动时检查）

4. 测试配置变更场景

---

### 阶段 3：P3 - 模块本地化

**目标**：将 govpp binapi NAT 模块拉取到本地。

**步骤**：
1. 本地化 `nat_types`（提交 → 构建 v1.1.0 → 测试）
2. 本地化 `nat44_ed`（提交 → 构建 v1.1.1 → 测试）
3. 验证所有功能无回归

---

### 阶段 4：P4 - 静态端口映射

**目标**：支持 DNAT 静态映射（可选高级功能）。

**步骤**：
1. 配置结构体新增静态映射字段：
   ```go
   type StaticMapping struct {
       PublicIP   string
       PublicPort uint16
       LocalIP    string
       LocalPort  uint16
       Protocol   string  // "tcp" | "udp"
   }
   ```

2. 调用 VPP API：
   ```go
   nat44_ed.Nat44AddDelStaticMapping{
       LocalIPAddress:    localIP,
       ExternalIPAddress: publicIP,
       LocalPort:         localPort,
       ExternalPort:      publicPort,
       Protocol:          protocolID,  // 6=TCP, 17=UDP
       IsAdd:             true,
   }
   ```

3. 测试外部主动连接场景

---

## 4. 关键技术点

### 4.1 与 ACL 的差异对比

| 对比项 | ACL 实现 | NAT 实现 |
|--------|---------|---------|
| **规则创建** | 两次调用 `ACLAddReplace`（ingress + egress） | 一次调用 `Nat44InterfaceAddDelFeature` |
| **双向处理** | 手动交换 src/dst 创建反向规则 | VPP 自动维护会话表反向转换 |
| **规则应用** | `ACLInterfaceSetACLList` 绑定规则列表到接口 | 配置接口角色（inside/outside） |
| **状态管理** | 无状态（每个数据包独立匹配） | 有状态（会话表跟踪连接） |
| **清理操作** | `ACLDel` 删除规则，遍历索引列表 | `Nat44InterfaceAddDelFeature` 禁用接口功能 |
| **配置复杂度** | 需构造规则列表（src/dst/port/protocol） | 仅需配置 IP 地址池和端口范围 |

**结论：NAT 实现比 ACL 更简单。**

---

### 4.2 VPP NAT44 ED API 使用

**关键 API**：
1. `Nat44InterfaceAddDelFeature`：配置接口为 inside/outside
2. `Nat44AddDelAddressRange`：配置 NAT 地址池
3. `Nat44AddDelStaticMapping`：配置静态端口映射（P4）
4. `Nat44UserDump`：查询会话信息（调试用）

**与 govpp 集成**：
```go
import (
    "github.com/networkservicemesh/govpp/binapi/nat44_ed"
    "github.com/networkservicemesh/govpp/binapi/nat_types"
    "github.com/networkservicemesh/govpp/binapi/interface_types"
)

client := nat44_ed.NewServiceClient(vppConn)
```

---

### 4.3 错误处理与边界情况

**端口耗尽**：
- VPP 自动拒绝新连接（返回错误）
- NAT NSE 记录错误日志："NAT 端口池耗尽，拒绝连接"

**配置错误**：
- 启动时验证配置（IP 格式、端口范围、静态映射冲突）
- 配置错误时拒绝启动，记录中文错误日志

**会话清理**：
- VPP 自动管理超时（TCP 7200s, UDP 300s, ICMP 60s）
- NSC 断开连接时，禁用接口 NAT 功能，VPP 清理会话

---

## 5. 验证计划

### 5.1 功能测试

**P1 测试用例**：
1. NSC ping 外部服务器（验证 ICMP SNAT）
2. NSC curl 外部 HTTP 服务（验证 TCP SNAT）
3. NSC 发送 UDP 数据包（验证 UDP SNAT）
4. 多个 NSC 并发连接（验证端口分配无冲突）
5. VPP CLI 查询会话表（验证会话创建和清理）

**P2 测试用例**：
1. 修改配置文件中的公网 IP（验证配置加载）
2. 修改端口范围（验证端口分配在范围内）
3. 禁用 ICMP（验证协议过滤）
4. 配置错误时启动失败（验证配置验证）

**P3 测试用例**：
1. 本地化后编译成功（验证依赖正确）
2. 运行功能测试无回归（验证功能一致性）
3. 版本号正确递增（验证版本管理）

**P4 测试用例**：
1. 配置静态映射（公网IP:8080 → 内部IP:80）
2. 外部客户端连接公网IP:8080（验证 DNAT）
3. 内部服务响应（验证反向转换）
4. 静态映射与动态 SNAT 共存（验证互不干扰）

---

### 5.2 性能测试

**目标**：
- 单个 NAT NSE 支持 ≥1000 并发会话
- NAT 延迟增加 <1ms
- 连接成功率 ≥99%

**工具**：
- iperf3（TCP 性能测试）
- hping3（UDP/ICMP 测试）
- ab / wrk（HTTP 压力测试）

---

## 6. 风险与缓解

### 风险 1：VPP NAT44 插件配置不当导致数据包丢失

**缓解**：
- 仔细研读 VPP NAT44 ED 文档
- 在测试环境充分验证后再推广
- 添加详细的中文日志，便于排查

### 风险 2：本地化模块导致依赖冲突

**缓解**：
- 严格遵循 ACL 本地化模式
- 使用 `go mod tidy` 验证依赖
- 每次本地化后运行完整测试

### 风险 3：静态映射与动态 SNAT 冲突

**缓解**：
- 配置验证：检测端口冲突
- VPP 自身已处理静态/动态共存
- 添加测试用例验证共存场景

---

## 7. 参考资料

### 代码参考
- `internal/acl/server.go`：NSE 服务器链式处理模式
- `internal/acl/common.go`：VPP API 调用模式
- `main.go:210-248`：双接口 SFC 架构

### 规范参考
- `specs/002-acl-localization/spec.md`：本地化流程
- `specs/003-vpp-nat/spec.md`：NAT 功能需求
- `specs/003-vpp-nat/checklists/requirements.md`：质量标准

### VPP 官方文档（参考 VPP 23.10，与项目使用的 govpp binapi 版本对应）
- **NAT44 ED 插件文档**：https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- **NAT44 ED CLI 参考**：https://s3-docs.fd.io/vpp/23.02/cli-reference/clis/clicmd_src_plugins_nat_nat44-ed.html
- **VPP NAT Wiki**：https://wiki.fd.io/view/VPP/NAT
- **NAT44 ED API 定义**（govpp binapi）：
  - `$GOPATH/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba/binapi/nat44_ed/nat44_ed.ba.go`
  - `$GOPATH/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba/binapi/nat_types/nat_types.ba.go`

### VPP CLI 命令示例（供理解，代码实现使用 Binary API）
```bash
# 启用 NAT44 插件
vpp# nat44 enable

# 配置接口为 inside（内部）
vpp# set interface nat44 in GigabitEthernet0/8/0

# 配置接口为 outside（外部）
vpp# set interface nat44 out GigabitEthernet0/a/0

# 添加 NAT 地址池
vpp# nat44 add address 192.0.2.1

# 查看 NAT 接口配置
vpp# show nat44 interfaces

# 查看 NAT 会话
vpp# show nat44 sessions
```

### govpp Binary API vs VPP CLI 对应关系
| CLI 命令 | Binary API | govpp 函数 |
|---------|-----------|-----------|
| `set interface nat44 in <if>` | `nat44_interface_add_del_feature` (is_add=true, flags=NAT_IS_INSIDE) | `Nat44InterfaceAddDelFeature()` |
| `set interface nat44 out <if>` | `nat44_interface_add_del_feature` (is_add=true, flags=NAT_IS_OUTSIDE) | `Nat44InterfaceAddDelFeature()` |
| `nat44 add address <ip>` | `nat44_add_del_address_range` (is_add=true) | `Nat44AddDelAddressRange()` |
| `nat44 add static mapping ...` | `nat44_add_del_static_mapping` (is_add=true) | `Nat44AddDelStaticMapping()` |

---

## 8. 总结

**核心要点**：
1. ✅ 使用 VPP NAT44 ED 官方实现（2/5 难度，1-2 周开发）
2. ✅ 不需要 ACL 的双向规则（VPP 自动处理反向转换）
3. ✅ NAT 插件完全本地化（nat_types + nat44_ed）
4. ✅ 遵循 ACL 项目结构（最小化改动）
5. ✅ 增量交付（P1→P2→P3→P4）

**下一步行动**：
1. 用户审核本实施方案
2. 更新 `spec.md` 的 Clarifications 章节
3. 执行 `/speckit.plan` 生成详细计划

---

**待用户确认的问题**：
1. 是否同意使用 VPP NAT44 ED 官方实现？
2. 是否同意删除 ACL 的双向规则模式（改为 inside/outside 接口配置）？
3. 是否同意本地化策略（nat_types + nat44_ed，最小化修改）？
4. 是否同意分阶段实施（P1→P2→P3→P4）？
5. 是否有其他技术疑问或调整建议？
