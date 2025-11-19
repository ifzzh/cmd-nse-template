# Feature Specification: VPP NAT 网络服务端点

**Feature Branch**: `003-vpp-nat`
**Created**: 2025-01-13
**Status**: Draft
**Input**: User description: "将本地基于ACL vpp的防火墙，制作一个同样基于vpp的nat，功能是能够修改NSM传输到NSE的数据包的ip和端口；要求最小化改动，严格模仿NSE端点构建的流程，只改动需要改动的部分；模仿ACL实现一个本地的NAT插件（也可以参考vpp自身的nat44插件）。要求良好定义开发流程，从简单到复杂，每一步做好版本控制和测试"

## Clarifications

### Session 2025-01-13

**Q1: 是否应该直接修改现有代码，还是创建新项目？**
- **A**: 直接修改现有 ACL 防火墙代码，不创建新项目。将 ACL 代码改为 NAT44 需要的实现。

**Q11: ACL 代码处理策略**
- **A**: 渐进式清理策略：
  - **初期**：可以直接修改 ACL 代码为 NAT，或先复制 ACL 到备份目录（如 `.archive/acl/` 或临时分支）用作参考
  - **中期**：在 NAT 实现过程中，ACL 代码作为参考模板存在
  - **最终**：NAT 实现完成并验收合格后，彻底删除所有 ACL 相关代码（包括 `internal/acl/` 目录、main.go 中的 ACL 导入和链条、配置文件中的 ACL 规则）
  - **版本管理**：删除 ACL 代码时创建单独的 commit，便于必要时回溯

**Q12: 项目重命名策略**
- **A**: 保持现有项目名称和路径不变：
  - **仓库名称**：保持 `cmd-nse-firewall-vpp` 不变
  - **模块路径**：保持 `github.com/ifzzh/cmd-nse-template` 不变
  - **目录结构**：保持现有目录名不变
  - **文档更新**：更新 README.md 和相关文档，说明项目已从 ACL 防火墙转型为 NAT 网络服务端点
  - **镜像命名**：使用 `ifzzh520/vpp-nat44-nat` 清晰表明功能（已在 spec 中定义）
  - **理由**：避免大规模重命名带来的风险和工作量，保持 Git 历史连续性，镜像名称已足够清晰

**Q13: ACL 代码渐进式清理的执行细节**
- **Q13.1 - ACL 代码删除的具体时间点**: **D - 创建独立的 P5 阶段专门删除 ACL**
  - **阶段定义**: P5 - 清理 ACL 遗留代码（v1.3.0），作为独立阶段在 P4（静态端口映射）完成后执行
  - **工作内容**:
    - 彻底删除 `internal/acl/` 目录及所有 ACL 代码文件
    - 删除 `main.go` 中的 ACL 导入和集成代码
    - 删除 `Config` 结构体中的所有 ACL 配置字段（`ACLConfigPath`、`ACLConfig`、`retrieveACLRules()`）
    - 删除配置文件示例中的 ACL 规则
    - 更新 README.md 和所有文档，移除 ACL 相关说明
    - 搜索代码库确保无 ACL 残留（`grep -ri "acl" .` 无结果，忽略大小写）
  - **验收标准**:
    - 代码编译通过，所有 NAT 功能测试通过
    - 代码库中无任何 ACL 痕迹（除 Git 历史）
    - Docker 镜像构建成功，K8s 部署验证通过
  - **版本管理**: 单独的 commit `refactor(cleanup): P5 - 彻底删除 ACL 遗留代码 (v1.3.0)`
  - **理由**: 清理工作正式化，有明确的验收标准和回退方案，避免与 P1-P4 功能开发混淆

- **Q13.2 - P1.3 中 ACL 和 NAT 代码的集成策略**: **B - 直接替换（删除 ACL，添加 NAT）**
  - **实施方式**: P1.3 的 commit 中直接删除 `main.go:224` 的 `acl.NewServer(vppConn, config.ACLConfig)`，替换为 `nat.NewServer(vppConn, config.NATConfig)`
  - **回退保障**: P1.3 之前创建 Git 标签 `v1.0.0-acl-final`，标记 ACL 功能的最后稳定版本，便于必要时回溯
  - **理由**: 符合"直接转型为 NAT 项目"的核心意图，避免长期维护两套代码的成本，Git 版本控制提供充分的回退能力

- **Q13.3 - Config 结构体中 ACLConfig 字段的处理时机**: **B - P2（配置管理阶段）删除并替换**
  - **P1.3 策略**: 保留 `Config` 结构体中的 ACL 字段（避免大范围改动），但 `nat.NewServer()` 使用硬编码的公网 IP 参数（如 `[]net.IP{net.ParseIP("192.168.1.100")}`），不依赖配置文件
  - **P2 策略**:
    - 删除 `ACLConfigPath`、`ACLConfig` 字段和 `retrieveACLRules()` 函数
    - 添加 `NATConfigPath string` 和 `NATConfig NATConfig` 字段
    - 实现 NAT 配置文件加载逻辑（从 YAML 加载公网 IP、端口范围等）
    - 版本号 v1.0.4
  - **理由**: P1.3 专注于核心功能（NAT API 调用验证），P2 专注于配置管理，职责单一，降低每个阶段的风险

**Q2: VPP 数据面的规则/插件注入是双向的还是在某个接口发生？**
- **A**: 基于代码分析（`main.go:210-248`）：
  - NSE 使用双接口架构：Interface A（server 端）连接 NSC，Interface B（client 端）连接下游 NSE
  - VPP xconnect 在两个接口之间创建 L2 桥接
  - NAT 配置通过接口角色实现（inside/outside），VPP 自动处理双向转换
  - 与 ACL 不同：ACL 需要手动创建双向规则（src/dst 交换），NAT 仅需配置接口角色一次

**Q3: 是否应该取消 ACL 中的双向规则模式？**
- **A**: 是的。NAT 不需要 ACL 的 src/dst 交换逻辑：
  - ACL 实现：创建两组规则（ingress + egress），出站规则交换 src/dst
  - NAT 实现：配置接口为 inside/outside，VPP 通过会话表自动处理反向转换
  - 结论：NAT 实现比 ACL 更简单

**Q4: NAT 插件本地化策略？**
- **A**: 完全遵循 ACL 本地化模式（`specs/002-acl-localization/spec.md`）：
  - 本地化模块：`nat_types` 和 `nat44_ed`（从 govpp binapi 复制到 `internal/`）
  - 版本：govpp v0.0.0-20240328101142-8a444680fbba（VPP 23.10-rc0~170-g6f1548434）
  - 最小化修改：仅添加中文注释，不修改业务逻辑
  - 版本管理：基线 v1.0.0 → v1.1.0（nat_types）→ v1.1.1（nat44_ed）

**Q5: 应该使用 VPP NAT44 官方实现还是自研？**
- **A**: 使用 VPP NAT44 ED 官方实现：
  - 难度对比：官方 2/5 vs 自研 5/5
  - 代码量：官方 ~50 行 vs 自研 ~500-1000 行
  - 开发周期：官方 1-2 周 vs 自研 1-2 个月
  - 功能：VPP 自动管理会话表、超时、端口分配，自研需手动实现
  - 风险：官方低风险，自研高风险

**Q6: NAT 接口角色配置？**
- **A**: 基于 SFC 架构（NSC → NAT NSE → 下游 NSE → 外网）：
  - Interface A（server 端）：配置为 NAT **inside** 接口（内部网络侧）
  - Interface B（client 端）：配置为 NAT **outside** 接口（外部网络侧）
  - 数据流：NSC → Interface A (inside) → SNAT 转换 → xconnect → Interface B (outside) → 外网

**Q7: SNAT 和 DNAT 的关系？**
- **A**: 两者可共存，功能不同：
  - SNAT（P1）：出站流量源地址转换（NSC → 外网），VPP 自动处理反向响应
  - DNAT（P4）：入站流量目的地址转换（外网 → 内部服务），基于静态端口映射
  - SNAT 反向路径 ≠ DNAT 正向路径（前者基于会话，后者基于静态映射）

**Q8: P1 阶段（基础 SNAT）的最小可验证单元是什么？**
- **A**: 将 P1 拆分为 3 个独立的可验证子模块，每个子模块可独立提交、构建、测试、回退：
  - **P1.1 - NAT 框架创建**（v1.0.1）：创建 `internal/nat/` 目录和文件结构（`server.go`、`common.go`），实现空的 `natServer` 结构体和接口方法（仅返回 `next.Server(ctx).Request/Close()`），确保项目编译通过，无功能变更
  - **P1.2 - 接口角色配置**（v1.0.2）：实现 `configureNATInterface()` 和 `disableNATInterface()` 函数，调用 VPP API `Nat44InterfaceAddDelFeature` 配置 inside/outside 角色，集成到 `Request()` 和 `Close()` 方法，验证 VPP 接口配置成功（通过 VPP CLI `show nat44 interfaces` 检查）
  - **P1.3 - 地址池配置与集成**（v1.0.3）：实现 `configureNATAddressPool()` 函数，调用 VPP API `Nat44AddDelAddressRange` 配置公网 IP 地址池，集成到 `main.go` 的 server 端链中（替换或并列 ACL），端到端测试 NAT 地址转换功能（NSC → 外部服务器 ping 测试）

**Q9: P3（模块本地化）的执行顺序和回退策略？**
- **A**: 按依赖顺序逐个本地化，每个模块独立验证后再进行下一个：
  - **P3.1 - 本地化 nat_types**（v1.1.0）：从 govpp 缓存复制 `binapi/nat_types/` 到 `internal/binapi_nat_types/`，创建 go.mod，配置 replace 指令，添加中文注释，编译验证 → Docker 镜像构建 → K8s 部署测试 → 确认所有 NAT 功能无回归
  - **P3.2 - 本地化 nat44_ed**（v1.1.1）：从 govpp 缓存复制 `binapi/nat44_ed/` 到 `internal/binapi_nat44_ed/`，创建 go.mod，配置 replace 指令（nat44_ed 依赖本地化的 nat_types），添加中文注释，编译验证 → Docker 镜像构建 → K8s 部署测试 → 确认所有 NAT 功能无回归
  - **回退策略**：任一步骤失败，立即回退到上一稳定版本（v1.0.3 → v1.1.0 → v1.1.1），删除失败的本地化模块，恢复 go.mod 的 replace 指令

**Q10: Git 提交粒度和消息格式？**
- **A**: 每个子模块完成后立即提交（编译通过 + 基本测试），提交粒度与子模块划分完全对应，便于精确回退。提交消息格式：
  ```
  <type>(<scope>): <子模块ID> - <描述> (<版本号>)

  <详细说明>
  - 关键变更点 1
  - 关键变更点 2
  - 验证方式
  ```
  **示例**：
  ```
  feat(nat): P1.1 - 创建 NAT 框架 (v1.0.1)

  - 创建 internal/nat/server.go 和 common.go
  - 实现空的 natServer 结构体
  - Request/Close 方法仅调用 next.Server()
  - 编译验证通过，无功能变更
  ```
  **type**: `feat`（新功能）、`fix`（修复）、`refactor`（重构）、`docs`（文档）
  **scope**: `nat`（NAT 模块）、`config`（配置）、`localize`（本地化）
  **回退命令**: `git revert <commit-hash>` 或 `git reset --hard <previous-commit>`

**关键 API 接口**（来自 govpp binapi nat44_ed）：

1. **配置接口角色**：`Nat44InterfaceAddDelFeature`
   ```go
   type Nat44InterfaceAddDelFeature struct {
       IsAdd     bool                           // true = 启用, false = 禁用
       Flags     nat_types.NatConfigFlags       // NAT_IS_INSIDE(32) 或 NAT_IS_OUTSIDE(16)
       SwIfIndex interface_types.InterfaceIndex // VPP 接口索引
   }
   ```

2. **配置地址池**：`Nat44AddDelAddressRange`
   ```go
   type Nat44AddDelAddressRange struct {
       FirstIPAddress ip_types.IP4Address      // NAT 公网 IP 起始地址
       LastIPAddress  ip_types.IP4Address      // NAT 公网 IP 结束地址（单 IP 时相同）
       VrfID          uint32                   // VRF ID（默认 0）
       IsAdd          bool                     // true = 添加, false = 删除
       Flags          nat_types.NatConfigFlags // 标志位（默认 0）
   }
   ```

3. **静态端口映射**（P4）：`Nat44AddDelStaticMapping`（已标记 Deprecated，但可用）
   ```go
   type Nat44AddDelStaticMapping struct {
       IsAdd             bool                           // true = 添加, false = 删除
       Flags             nat_types.NatConfigFlags       // NAT_IS_ADDR_ONLY 等
       LocalIPAddress    ip_types.IP4Address            // 内部服务器 IP
       ExternalIPAddress ip_types.IP4Address            // 公网 IP
       Protocol          uint8                          // 协议（6=TCP, 17=UDP）
       LocalPort         uint16                         // 内部端口
       ExternalPort      uint16                         // 公网端口
       ExternalSwIfIndex interface_types.InterfaceIndex // 外部接口索引（可选）
       VrfID             uint32                         // VRF ID
       Tag               string                         // 标签（最长 64 字符）
   }
   ```

**实施策略调整**：
- 框架复制：完全参考 `internal/acl/` 的结构（`server.go`、`common.go`）
- 微调部分：API 调用改为 NAT44 接口，删除 src/dst 交换逻辑
- 大改部分：无（NAT 比 ACL 更简单，不需要大改）

**VPP 官方文档参考**：
- **NAT44 ED 插件文档**（VPP 23.02）：https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- **NAT44 ED CLI 参考**：https://s3-docs.fd.io/vpp/23.02/cli-reference/clis/clicmd_src_plugins_nat_nat44-ed.html
- **VPP NAT Wiki**：https://wiki.fd.io/view/VPP/NAT
- **关键 CLI 命令示例**（代码实现使用 Binary API，CLI 仅供理解）：
  ```bash
  # 启用 NAT44 插件
  vpp# nat44 enable

  # 配置接口为 inside（内部）/ outside（外部）
  vpp# set interface nat44 in <interface> out <interface>

  # 添加 NAT 地址池
  vpp# nat44 add address <public-ip>

  # 查看配置和会话
  vpp# show nat44 interfaces
  vpp# show nat44 sessions
  ```

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 基础 NAT44 地址转换 (Priority: P1)

网络服务提供者需要在 Network Service Mesh 环境中提供基本的 NAT44 地址转换功能，将通过 NSM 连接到 NSE 的客户端数据包的源 IP 地址和端口进行转换，使多个内部客户端能够共享公网 IP 地址访问外部网络服务。

**Why this priority**: 这是 NAT 功能的核心价值，能够独立验证 VPP NAT44 插件与 NSM 的集成，为后续高级功能奠定基础。地址转换是 NAT 的最小可交付功能。

**Incremental Implementation** (详见 Clarifications Q8)：
- **P1.1** (v1.0.1): NAT 框架创建（空实现，编译通过）
- **P1.2** (v1.0.2): 接口角色配置（VPP API 调用，VPP CLI 验证）
- **P1.3** (v1.0.3): 地址池配置与集成（端到端 NAT 转换测试）

**Independent Test**: 可以通过部署 NAT NSE，从 NSC 发起网络连接到外部服务器，验证源 IP 地址和端口被正确转换，并且能够接收响应数据包。测试包括: (1) NSC → NAT NSE → 外部服务器的连接建立 (2) ICMP ping 测试 (3) TCP 连接测试 (4) 验证 VPP NAT44 会话表中存在转换记录。每个子模块可独立回退。

**Acceptance Scenarios**:

1. **Given** NSC 已连接到 NAT NSE, **When** NSC 向外部服务器发送 ICMP ping 请求, **Then** 数据包的源 IP 地址被转换为 NAT 公网 IP 地址，外部服务器能够收到请求并响应
2. **Given** NAT 转换已建立, **When** 外部服务器回复响应数据包, **Then** NAT NSE 能够将目标 IP 和端口反向转换为 NSC 的内部地址，NSC 收到正确的响应
3. **Given** 多个 NSC 同时连接, **When** 它们同时发起外部连接, **Then** 每个 NSC 的数据包使用不同的转换端口号，不发生端口冲突
4. **Given** NAT NSE 运行中, **When** 查询 VPP NAT44 会话表, **Then** 能够看到所有活跃的地址转换映射记录
5. **Given** NSC 断开连接, **When** NAT NSE 关闭该连接的网络接口, **Then** 相关的 NAT 会话被清理，端口资源被释放

---

### User Story 2 - NAT 规则配置管理 (Priority: P2)

运维人员需要能够通过配置文件或环境变量定义 NAT 转换规则，包括指定公网 IP 地址池、端口范围、转换方向（入站/出站）、协议类型（TCP/UDP/ICMP）等参数，而无需修改代码或重新构建镜像。

**Why this priority**: 配置管理提高了 NAT NSE 的可复用性和灵活性，但在基础转换功能验证通过后再实现更为合理，避免过早优化。

**Independent Test**: 通过修改 ConfigMap 或环境变量中的 NAT 配置，重启 NAT NSE Pod，验证新配置被正确加载并应用到 VPP NAT44 插件中。测试包括: (1) 修改公网 IP 地址 (2) 修改端口范围 (3) 启用/禁用特定协议的转换 (4) 验证配置加载日志和 VPP 配置状态。

**Acceptance Scenarios**:

1. **Given** NAT 配置文件包含公网 IP 地址池, **When** NAT NSE 启动时加载配置, **Then** VPP NAT44 插件使用指定的 IP 地址池进行转换
2. **Given** 配置指定了端口范围 1024-65535, **When** NAT 分配转换端口, **Then** 所有分配的端口号在该范围内
3. **Given** 配置禁用了 ICMP NAT 转换, **When** NSC 发送 ICMP ping, **Then** 数据包不经过 NAT 转换，直接转发或丢弃
4. **Given** 运维人员修改了配置文件, **When** 重新部署 NAT NSE, **Then** 新配置生效，旧的 NAT 会话被保留或平滑迁移
5. **Given** 配置格式错误, **When** NAT NSE 尝试加载配置, **Then** 启动失败并记录清晰的中文错误日志，指明配置问题

---

### User Story 3 - NAT 模块本地化与增量交付 (Priority: P3)

开发团队需要将 VPP govpp binapi 中的 NAT 相关模块（nat_types、nat44_ed）逐个本地化到项目 internal/ 目录，每次本地化一个模块后进行版本控制提交、Docker 镜像构建和功能测试，确保每个增量步骤都可验证和回滚。

**Why this priority**: 模块本地化是长期维护和减少外部依赖的需求，但在基础功能和配置管理实现并验证后进行更为稳妥，避免过早引入模块管理复杂度。

**Incremental Implementation** (详见 Clarifications Q9)：
- **P3.1** (v1.1.0): 本地化 nat_types（基础类型定义，无逻辑风险）
- **P3.2** (v1.1.1): 本地化 nat44_ed（依赖本地化的 nat_types）
- **回退策略**：任一步骤失败立即回退，删除失败模块，恢复 go.mod

**Independent Test**: 每次本地化一个 NAT 模块后，通过编译项目、运行单元测试、构建 Docker 镜像并在 Kubernetes 中部署验证，确认所有 NAT 功能仍然正常工作。测试包括: (1) Go 编译成功 (2) 单元测试通过 (3) Docker 镜像构建成功 (4) Kubernetes 部署功能测试通过 (5) 版本号正确递增。每个子模块可独立回退。

**Acceptance Scenarios**:

1. **Given** 选定 nat_types 模块进行本地化, **When** 从 govpp 缓存复制代码到 internal/binapi_nat_types/, **Then** 代码与在线版本完全一致，仅添加中文注释
2. **Given** 本地化模块代码已复制, **When** 创建模块 go.mod 并配置依赖, **Then** 项目编译成功，所有依赖正确解析
3. **Given** 本地化完成并提交, **When** 构建 Docker 镜像, **Then** 镜像版本号从基线（如 v1.1.0）自动递增（如 v1.1.1）
4. **Given** 新镜像已构建, **When** 在 Kubernetes 中部署测试, **Then** 所有 NAT 功能测试通过，无功能回归
5. **Given** 测试失败, **When** 执行回滚操作, **Then** 能够快速恢复到上一个稳定镜像版本，并定位到失败的模块

---

### User Story 4 - 静态端口映射 (Priority: P4)

服务运维人员需要配置静态端口映射（Static NAT / Port Forwarding），将特定的公网 IP 和端口固定映射到内部服务器的 IP 和端口，支持外部客户端主动访问内部服务（如对外提供的 Web 服务或数据库服务）。

**Why this priority**: 静态映射是高级功能，满足特定的服务暴露需求，但不是 NAT 的核心价值，可在基础动态 NAT 稳定后扩展。

**Independent Test**: 配置静态映射规则（例如 公网IP:8080 → 内部IP:80），从外部客户端尝试访问公网 IP 的 8080 端口，验证请求被正确转发到内部服务器的 80 端口，并能收到响应。

**Acceptance Scenarios**:

1. **Given** 配置文件包含静态映射规则, **When** NAT NSE 加载配置, **Then** VPP NAT44 插件创建对应的静态映射条目
2. **Given** 静态映射已建立, **When** 外部客户端连接到公网 IP:8080, **Then** 请求被转发到内部服务器的 80 端口
3. **Given** 内部服务器响应请求, **When** 响应数据包经过 NAT NSE, **Then** 源 IP 和端口被转换为公网 IP:8080，外部客户端收到响应
4. **Given** 静态映射与动态 NAT 共存, **When** 同时存在入站和出站流量, **Then** 两种转换规则互不干扰，均能正常工作

---

### Edge Cases

- 当 NAT 端口池耗尽（所有端口已分配），新的 NSC 尝试建立连接时，如何处理？是拒绝连接、排队等待，还是记录错误日志？
- 当 NAT 会话数量达到 VPP 内存限制或性能瓶颈时，系统如何表现？是否会导致延迟增加或丢包？
- 当 NAT NSE 重启或崩溃，已有的 NAT 会话是否能够恢复？客户端是否需要重新建立连接？
- 当外部服务器返回 ICMP 错误消息（如 Destination Unreachable）时，NAT NSE 如何反向转换 ICMP 错误包中嵌入的原始 IP 头部？
- 当配置中指定的公网 IP 地址在 VPP 接口上不存在时，NAT 模块是否能够检测并报告配置错误？
- 当多个 NSC 使用相同的内部 IP 地址（如都是 172.16.1.x），NAT 如何区分它们并正确转换？
- 当 NAT 规则配置变更（如修改 IP 地址池），是否需要清理旧的会话，还是仅影响新建连接？
- 当 NSC 和 NAT NSE 之间的接口启用了 Jumbo Frame 或特殊 MTU 设置，NAT 转换是否会影响数据包分片和重组？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须基于 VPP NAT44 Endpoint-Dependent（ED）插件实现 Network Address Translation 功能
- **FR-002**: 系统必须能够作为 NSM NetworkServiceEndpoint（NSE）运行，接收来自 NSC（NetworkServiceClient）的连接请求
- **FR-003**: 系统必须对通过 NSM 传输到 NSE 的数据包执行源 NAT（SNAT）转换，修改源 IP 地址和源端口号
- **FR-004**: 系统必须维护 NAT 会话状态表，记录内部地址与公网地址的映射关系，支持双向数据包转换
- **FR-005**: 系统必须支持 TCP、UDP、ICMP 三种协议的 NAT 转换
- **FR-006**: 系统必须通过配置文件或环境变量指定 NAT 公网 IP 地址池和端口范围
- **FR-007**: 系统必须在 NSC 连接时创建 VPP 接口并配置 NAT inside/outside 接口属性
- **FR-008**: 系统必须在 NSC 断开连接时清理 NAT 会话和接口配置，释放资源
- **FR-009**: 系统必须将 VPP govpp binapi 中的 NAT 相关模块（nat_types、nat44_ed）本地化到项目 internal/ 目录
- **FR-010**: 本地化的 NAT 模块必须通过 go.mod replace 指令重定向导入路径，无需修改业务代码
- **FR-011**: 系统必须在每次模块本地化后执行 Git 提交、Docker 镜像构建、镜像版本号递增和功能测试
- **FR-012**: 系统必须生成清晰的中文日志，记录 NAT 配置加载、会话创建、会话删除、错误情况等关键事件
- **FR-013**: 系统必须支持可选的静态端口映射配置，将固定的公网 IP:端口映射到内部 IP:端口
- **FR-014**: 系统必须遵循 ACL 防火墙 NSE 的项目结构和开发流程作为参考模板，在 P5 阶段（v1.3.0）彻底删除所有 ACL 相关代码（详见 Q13.1）
- **FR-015**: 系统必须在 NAT 端口池耗尽时拒绝新连接并记录错误日志
- **FR-016**: 系统必须支持配置验证，在启动时检测配置错误并拒绝启动
- **FR-017**: 系统必须为每个 NSC 连接分配唯一的 NAT 会话，避免端口冲突
- **FR-018**: 系统必须能够在 Kubernetes 环境中通过 Deployment 部署，支持 SPIRE 身份认证和 NSM 服务注册

### Key Entities

- **NAT NSE（网络服务端点）**: 提供 NAT 功能的 NSM 端点，包含 VPP 连接、NAT 配置、会话管理器、接口映射
- **NAT 配置**: 定义 NAT 转换行为的参数集合，包含公网 IP 地址池、端口范围、协议类型、静态映射规则、转换方向
- **NAT 会话**: 记录单个网络流的地址转换映射，包含内部 IP/端口、公网 IP/端口、协议类型、会话状态、超时时间
- **VPP 接口**: NSM 在 VPP 中创建的虚拟网络接口（memif/kernel），配置为 NAT inside（内部网络）或 outside（外部网络）
- **本地化 NAT 模块**: 从 govpp binapi 复制到 internal/ 的 NAT 类型定义和 API 绑定，包含 nat_types、nat44_ed、版本信息、依赖关系
- **容器镜像**: 包含 NAT NSE 代码和依赖的可部署软件包，镜像名称为 `ifzzh520/vpp-nat44-nat`（nat44 表示 VPP 插件名称，nat 表示网络服务名称），包含语义化版本标签（如 v1.0.0）

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: NSC 通过 NAT NSE 建立外部连接的成功率达到 99% 以上
- **SC-002**: NAT 地址转换的延迟增加低于 1 毫秒（相比无 NAT 的直连）
- **SC-003**: 单个 NAT NSE 实例支持至少 1000 个并发 NAT 会话而不影响性能
- **SC-004**: NAT 会话建立和清理的响应时间在 100 毫秒以内
- **SC-005**: 所有 NAT 操作的日志覆盖率达到 95% 以上，且日志为简体中文
- **SC-006**: 每个 NAT 模块的本地化、提交、构建、测试周期在 2 小时内完成
- **SC-007**: 本地化后的 NAT 模块功能测试通过率达到 100%，无功能回归
- **SC-008**: NAT 配置加载失败时，系统能够在 5 秒内检测并拒绝启动
- **SC-009**: NAT 端口池耗尽时，新连接的拒绝率达到 100%，无端口冲突或会话混乱
- **SC-010**: 代码结构与 ACL 防火墙的相似度达到 80% 以上（体现最小化改动原则）
- **SC-011**: 从零开始部署 NAT NSE 到功能验证通过的总时间在 10 分钟以内
- **SC-012**: 测试失败时能够在 10 分钟内回滚到上一稳定版本

## Assumptions

- 假设项目已有成熟的 VPP 集成和 NSM 端点开发经验（基于 ACL 防火墙项目）
- 假设开发团队对 VPP NAT44 插件的工作原理和配置方法有基本了解
- 假设测试环境能够模拟 NSC 到 NAT NSE 的连接和外部网络访问场景
- 假设 VPP NAT44 ED 插件已在 VPP 镜像中编译并可用（基于 v24.10.0）
- 假设 NAT 公网 IP 地址由网络管理员分配，且在 VPP 接口上可配置
- 假设 NAT 端口范围默认为 1024-65535，可通过配置调整
- 假设 NAT 会话超时时间遵循 VPP NAT44 插件的默认配置（TCP 7200s, UDP 300s, ICMP 60s）
- 假设镜像构建和版本管理流程可复用 ACL 防火墙项目的现有流程
- 假设 NAT NSE 不需要处理 IPv6 或 NAT64 场景（仅 IPv4 NAT44）
- 假设 NSC 和 NAT NSE 之间的接口类型为 memif 或 kernel，与 ACL 防火墙一致
- 假设每个 NSC 连接独占一个 VPP 接口，不存在多租户共享接口的场景
- 假设开发流程遵循"简单到复杂"的增量交付原则，每个里程碑可独立验证

## Dependencies

- VPP v24.10.0 或更高版本（包含 NAT44 ED 插件）
- govpp binapi 中的 nat_types 和 nat44_ed 模块
- Network Service Mesh SDK v1.15.0-rc.1 或兼容版本
- SPIRE v1.8.0（用于身份认证）
- Docker 或 Podman（用于镜像构建）
- Kubernetes 集群（用于部署测试）
- Git 版本控制系统
- Go 1.23.8 或更高版本
- 现有的 ACL 防火墙项目代码和文档（作为参考模板，最终将被删除）

## Out of Scope

- IPv6 NAT 或 NAT64 功能（仅支持 IPv4 NAT44）
- NAT 会话持久化或跨重启恢复（会话重启后清空）
- NAT 日志的详细数据包级别记录（仅记录关键事件）
- NAT 性能优化或高级调优（使用 VPP 默认配置）
- 与第三方网络设备或云 NAT 服务的集成
- 用户界面或 CLI 工具（仅支持配置文件管理）
- NAT 穿透或 UPnP/NAT-PMP 协议支持
- 多层 NAT 或 Carrier-Grade NAT（CGN）场景
- 对非 NSM 流量的 NAT 处理（仅处理 NSM 连接）
- 实时监控或 Prometheus 指标导出（可作为未来增强）
- 自动化在线模块版本更新同步机制（手动追踪）
- ACL 防火墙功能的保留或维护（项目最终将完全转型为 NAT 项目）
