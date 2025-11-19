# Implementation Plan: VPP NAT 网络服务端点

**Branch**: `003-vpp-nat` | **Date**: 2025-01-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-vpp-nat/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

将现有的 ACL 防火墙 NSE 项目转型为基于 VPP NAT44 ED 插件的网络地址转换服务端点。项目采用渐进式清理策略，分 5 个阶段（P1-P5）实现：P1 创建基础 SNAT 功能，P2 实现配置管理，P3 本地化 NAT 模块，P4 添加静态端口映射，P5 彻底删除 ACL 遗留代码。技术路线选择 VPP 官方 NAT44 ED 实现（而非自研），通过 govpp binapi 调用 VPP API，每个子模块独立验证和版本控制，确保每步可回退。

## Technical Context

**Language/Version**: Go 1.23.8
**Primary Dependencies**:
- VPP v24.10.0（提供 NAT44 ED 插件）
- govpp v0.0.0-20240328101142-8a444680fbba（VPP 23.10-rc0~170-g6f1548434）
- Network Service Mesh SDK v1.15.0-rc.1（NSM 框架）
- SPIRE v1.8.0（身份认证）
- 本地化 govpp binapi 模块：nat_types、nat44_ed

**Storage**: N/A（NAT 会话由 VPP 内存管理，无需外部存储）
**Testing**: Go 原生测试框架 + Docker 容器测试 + K8s 部署验证
**Target Platform**: Linux server（Kubernetes 环境中的容器化部署）
**Project Type**: 单体项目（Network Service Endpoint，集成到 NSM 服务网格）
**Performance Goals**:
- NAT 转换延迟 < 1ms
- 单实例支持 ≥1000 并发 NAT 会话
- 会话建立/清理响应时间 < 100ms
- NSC 连接成功率 ≥99%

**Constraints**:
- 仅支持 IPv4 NAT44（不支持 IPv6 或 NAT64）
- NAT 会话不持久化（重启后清空）
- 配置验证失败时拒绝启动（< 5 秒内检测）
- 端口池耗尽时拒绝新连接（100% 拒绝率，无冲突）

**Scale/Scope**:
- 代码量：~500-1000 行（参考 ACL 实现，NAT 比 ACL 更简单）
- 阶段数：5 个（P1-P5）
- 子模块数：9 个（P1.1/P1.2/P1.3, P2, P3.1/P3.2, P4, P5）
- 版本号：v1.0.0 → v1.3.0（9 个增量版本）
- 交付周期：预估 2-3 周

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. 中文优先原则 ✅ 通过
- **状态**: 完全遵守
- **验证**:
  - 所有代码注释、日志、错误信息、文档将使用简体中文
  - Git 提交信息使用简体中文（格式：`类型(范围): 子模块ID - 描述 (版本号)`）
  - 代码标识符保持英文（如 `natServer`、`configureNATInterface`）
- **行动**: 无需调整

### II. 简洁架构原则 ✅ 通过
- **状态**: 完全遵守
- **验证**:
  - 项目保持扁平化结构（最大深度 ≤3 层）
  - 新增 `internal/nat/` 目录（模仿 `internal/acl/`）
  - P5 阶段删除 `internal/acl/` 目录，保持结构简洁
  - 不新增 CI/CD 配置或 vendor 目录
- **行动**: 无需调整

### III. 依赖稳定性原则 ✅ 通过（需谨慎）
- **状态**: 遵守，但需注意风险
- **验证**:
  - govpp 版本固定为 v0.0.0-20240328101142-8a444680fbba（与 ACL 一致）
  - VPP 版本固定为 v24.10.0
  - NSM SDK 版本固定为 v1.15.0-rc.1
  - 本地化 nat_types 和 nat44_ed 模块到 `internal/`，避免外部依赖变更风险
- **风险**:
  - govpp binapi 中 NAT44 API 标记为 Deprecated（`Nat44AddDelStaticMapping`），但仍可用
  - 未来 VPP 版本可能移除或重构 NAT44 API
- **行动**: Phase 0 研究中验证 API 稳定性，确认替代方案

### IV. 功能完整性原则 ⚠️ 有变更（已澄清）
- **状态**: P5 阶段删除 ACL 功能，需记录
- **验证**:
  - ACL 功能将在 P5（v1.3.0）完全删除
  - P1.3 之前创建 Git 标签 `v1.0.0-acl-final` 保留 ACL 最后稳定版本
  - 核心功能保护（VPP 连接、gRPC 端点、NSM 注册、SPIRE 认证）不受影响
- **理由**:
  - 项目战略转型为 NAT 服务端点
  - ACL 和 NAT 功能互斥，不需要共存
  - Git 历史和标签提供充分的回退能力
- **行动**: 已在 spec.md Q13.1 中明确，无需额外调整

### V. 一致性优先原则 ✅ 通过
- **状态**: 完全遵守
- **验证**:
  - NAT 实现完全参考 ACL 实现（`internal/acl/server.go`、`common.go`）
  - 复用 ACL 的配置管理模式（`envconfig`、`Config` 结构体）
  - 复用 ACL 的日志记录方式（`logrus`、中文日志）
  - 复用 ACL 的错误处理方式（`pkg/errors` 包装）
  - 函数命名遵循既有模式（动词+名词，如 `configureNATInterface`）
- **行动**: Phase 0 研究中分析至少 3 个 ACL 相似实现

### VI. 本地优先原则 ✅ 通过
- **状态**: 完全遵守
- **验证**:
  - 直接在工作目录修改现有文件
  - 增量修改和小步提交（每个子模块一个 commit）
  - 不创建项目副本或临时目录
- **行动**: 无需调整

### VII. 仓库独立性原则 ✅ 通过
- **状态**: 完全遵守
- **验证**:
  - 保持模块路径 `github.com/ifzzh/cmd-nse-template` 不变（Q12）
  - 外部依赖保持 `github.com/networkservicemesh/...` 路径
  - 本地代码使用 `github.com/ifzzh/...` 路径
  - 镜像名称 `ifzzh520/vpp-nat44-nat` 清晰表明功能
- **行动**: 无需调整

### 总体评估
- **通过项数**: 6/7
- **需注意项数**: 1/7（依赖稳定性 - NAT44 API Deprecated）
- **有变更项数**: 1/7（功能完整性 - ACL 删除，已澄清）
- **结论**: ✅ 允许进入 Phase 0 研究，需在研究阶段验证 NAT44 API 稳定性和替代方案

## Project Structure

### Documentation (this feature)

```text
specs/003-vpp-nat/
├── spec.md              # 功能规范（包含 Q1-Q13 澄清）
├── plan.md              # 本文件（实施计划）
├── research.md          # Phase 0 输出（技术研究）
├── data-model.md        # Phase 1 输出（数据模型）
├── quickstart.md        # Phase 1 输出（快速开始指南）
├── contracts/           # Phase 1 输出（VPP API 合约）
│   ├── vpp-nat44-api.md         # API 总览
│   ├── interface-role-api.md    # 接口角色配置 API（P1.2）
│   ├── address-pool-api.md      # 地址池配置 API（P1.3）
│   └── static-mapping-api.md    # 静态端口映射 API（P4）
├── implementation-approach.md # 实施方法（可选）
└── tasks.md             # Phase 2 输出（/speckit.tasks 命令生成）
```

### Source Code (repository root)

**项目类型**: 单体项目（Network Service Endpoint）

```text
./
├── main.go                      # 主程序入口（NSM 端点注册和服务启动）
├── internal/
│   ├── config/                  # 配置管理
│   │   └── config.go            # 配置结构体（将在 P2 更新为 NATConfig）
│   ├── acl/                     # ACL 防火墙实现（P5 阶段删除）
│   │   ├── server.go            # ACL server 实现（P1 参考模板）
│   │   └── common.go            # ACL 公共函数（P1 参考模板）
│   ├── nat/                     # NAT 实现（P1.1 创建）
│   │   ├── server.go            # NAT server 实现
│   │   └── common.go            # NAT 公共函数
│   ├── binapi_nat_types/        # 本地化 nat_types 模块（P3.1 创建）
│   │   ├── nat_types.ba.go
│   │   ├── nat_types_rpc.ba.go
│   │   └── go.mod
│   └── binapi_nat44_ed/         # 本地化 nat44_ed 模块（P3.2 创建）
│       ├── nat44_ed.ba.go
│       ├── nat44_ed_rpc.ba.go
│       └── go.mod
├── Dockerfile                   # 容器化配置
├── go.mod                       # Go 模块依赖（P3 更新 replace 指令）
├── go.sum                       # 依赖校验和
├── README.md                    # 项目文档（P5 更新）
└── .specify/                    # SpecKit 配置
    ├── memory/
    │   └── constitution.md      # 项目宪章
    └── templates/               # 模板文件
```

**Structure Decision**:
- 采用扁平化单体项目结构（遵循 Constitution II. 简洁架构原则）
- NAT 实现模仿 ACL 实现（`internal/acl/` → `internal/nat/`）
- 本地化模块独立目录（`internal/binapi_nat_types/`、`internal/binapi_nat44_ed/`）
- P5 阶段删除 `internal/acl/` 目录，保持结构简洁

---

## Implementation Phases

### Phase 0: 研究与环境准备（已完成）

**目标**: 解决 Technical Context 中的未知项，验证技术可行性

**交付物**:
- ✅ research.md（技术研究文档）
- ✅ VPP NAT44 ED API 稳定性验证
- ✅ ACL 实现模式分析（3 个相似实现）
- ✅ 测试策略和回退机制设计

**关键成果**:
- VPP NAT44 ED API 可用且稳定
- NAT 实现比 ACL 简单 56%（代码简化率）
- 80% 的 ACL 代码结构可直接复用

---

### Phase 1: 设计与合约（已完成）

**目标**: 定义数据模型、API 合约、快速开始指南

**交付物**:
- ✅ data-model.md（7 个核心实体）
- ✅ contracts/（4 个 VPP API 合约文档）
- ✅ quickstart.md（< 10 分钟部署指南）
- ✅ CLAUDE.md（已更新 agent context）

**关键成果**:
- 数据模型定义完整（NAT NSE、NAT 配置、NAT 会话、VPP 接口等）
- VPP Binary API 调用规范明确（Nat44InterfaceAddDelFeature、Nat44AddDelAddressRange、Nat44AddDelStaticMapping）
- 快速开始指南符合 SC-011 成功标准（< 10 分钟）

---

### Phase 2: 任务分解（待执行 /speckit.tasks）

**目标**: 将实施计划转化为可操作的任务清单

**交付物**:
- tasks.md（P1-P5 阶段的详细任务）
- 每个任务包含：描述、验收标准、依赖关系、优先级

---

### Phase 3: P1 - 基础 SNAT 实现（v1.0.1-v1.0.3）

**目标**: 创建 NAT 框架，配置接口角色和地址池，实现端到端 NAT 转换

**子阶段**:

#### P1.1 - NAT 框架创建（v1.0.1）
- 创建 `internal/nat/` 目录
- 创建 `server.go`、`common.go`（空实现）
- 实现 `natServer` 结构体和接口方法（仅调用 `next.Server()`）
- **验收标准**: 编译通过，无功能变更
- **Git commit**: `feat(nat): P1.1 - 创建 NAT 框架 (v1.0.1)`

#### P1.2 - 接口角色配置（v1.0.2）
- 实现 `configureNATInterface()` 函数（调用 `Nat44InterfaceAddDelFeature`）
- 实现 `disableNATInterface()` 函数（清理接口）
- 集成到 `Request()` 和 `Close()` 方法
- **验收标准**: VPP CLI `show nat44 interfaces` 显示正确的 inside/outside 接口
- **Git commit**: `feat(nat): P1.2 - 接口角色配置 (v1.0.2)`

#### P1.3 - 地址池配置与集成（v1.0.3）
- 实现 `configureNATAddressPool()` 函数（调用 `Nat44AddDelAddressRange`，硬编码公网 IP）
- 删除 `main.go` 中的 `acl.NewServer()`，替换为 `nat.NewServer()`（基于 Q13.2）
- 创建 Git 标签 `v1.0.0-acl-final`（在 P1.3 之前）
- **验收标准**: NSC → NAT NSE → 外部服务器 ping 测试成功，VPP CLI `show nat44 sessions` 显示 NAT 会话
- **Git commit**: `feat(nat): P1.3 - 地址池配置与集成 (v1.0.3)`

---

### Phase 4: P2 - NAT 配置管理（v1.0.4）

**目标**: 实现从配置文件加载 NAT 配置，删除 ACL 配置字段

**任务**:
- 删除 `Config` 结构体中的 `ACLConfigPath`、`ACLConfig`、`retrieveACLRules()` 函数
- 添加 `NATConfigPath string` 和 `NATConfig NATConfig` 字段
- 实现 YAML 配置文件加载逻辑
- 修改 `nat.NewServer()` 接受 `Config.NATConfig` 参数
- **验收标准**: 配置文件加载成功，配置验证失败时拒绝启动（< 5 秒）
- **Git commit**: `feat(config): P2 - NAT 配置管理 (v1.0.4)`

---

### Phase 5: P3 - 模块本地化（v1.1.0-v1.1.1）

**目标**: 本地化 govpp binapi NAT 模块，避免外部依赖变更风险

**子阶段**:

#### P3.1 - 本地化 nat_types（v1.1.0）
- 从 govpp 缓存复制 `binapi/nat_types/` 到 `internal/binapi_nat_types/`
- 创建 `go.mod`，配置 `replace` 指令
- 添加中文注释
- **验收标准**: 编译验证 → Docker 镜像构建 → K8s 部署测试 → NAT 功能无回归
- **Git commit**: `feat(localize): P3.1 - 本地化 nat_types (v1.1.0)`

#### P3.2 - 本地化 nat44_ed（v1.1.1）
- 从 govpp 缓存复制 `binapi/nat44_ed/` 到 `internal/binapi_nat44_ed/`
- 创建 `go.mod`，配置 `replace` 指令（nat44_ed 依赖本地化的 nat_types）
- 添加中文注释
- **验收标准**: 编译验证 → Docker 镜像构建 → K8s 部署测试 → NAT 功能无回归
- **Git commit**: `feat(localize): P3.2 - 本地化 nat44_ed (v1.1.1)`

---

### Phase 6: P4 - 静态端口映射（v1.2.0）

**目标**: 实现静态端口映射功能（端口转发）

**任务**:
- 实现 `configureStaticMapping()` 函数（调用 `Nat44AddDelStaticMapping`）
- 实现 `cleanupStaticMapping()` 函数（清理映射）
- 从配置文件加载静态映射规则
- **验收标准**: 外部客户端 → 公网 IP:端口 → 内部服务器 访问成功
- **Git commit**: `feat(nat): P4 - 静态端口映射 (v1.2.0)`

---

### Phase 7: P5 - 清理 ACL 遗留代码（v1.3.0）

**目标**: 彻底删除所有 ACL 相关代码，完成项目转型

**任务**（基于 Q13.1）:
- 删除 `internal/acl/` 目录
- 删除 `main.go` 中的 ACL 导入
- 删除配置文件示例中的 ACL 规则
- 更新 README.md 和所有文档，移除 ACL 相关说明
- 搜索验证：`grep -ri "acl" .` 无结果（忽略大小写）
- **验收标准**:
  - 代码编译通过
  - 所有 NAT 功能测试通过（100% 通过率）
  - Docker 镜像构建成功
  - K8s 部署验证通过
  - 代码库无 ACL 痕迹（除 Git 历史）
- **Git commit**: `refactor(cleanup): P5 - 彻底删除 ACL 遗留代码 (v1.3.0)`

---

## Version Management

### Version Timeline

```
v1.0.0 (baseline, ACL 防火墙)
    ↓
[Git 标签 v1.0.0-acl-final]  ← P1.3 之前创建
    ↓
v1.0.1 (P1.1 - NAT 框架创建)
    ↓
v1.0.2 (P1.2 - 接口角色配置)
    ↓
v1.0.3 (P1.3 - 地址池配置与集成，直接替换 ACL)
    ↓
v1.0.4 (P2 - NAT 配置管理，删除 ACL 配置字段)
    ↓
v1.1.0 (P3.1 - 本地化 nat_types)
    ↓
v1.1.1 (P3.2 - 本地化 nat44_ed)
    ↓
v1.2.0 (P4 - 静态端口映射)
    ↓
v1.3.0 (P5 - 彻底删除 ACL 遗留代码)
```

### Rollback Strategy

**三层防护**:
1. **Git 回退**: `git revert <commit-hash>` 或 `git reset --hard <previous-commit>`
2. **Docker 镜像回退**: `kubectl set image deployment/vpp-nat vpp-nat=ifzzh520/vpp-nat44-nat:v1.0.2`
3. **K8s Rollout 回退**: `kubectl rollout undo deployment/vpp-nat`

**Git 标签管理**:
- `v1.0.0-acl-final`: ACL 功能的最后稳定版本（P1.3 之前创建）
- `v1.0.1`-`v1.3.0`: NAT 功能渐进式版本

---

## Risk Management

### 关键风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| VPP NAT44 ED API Deprecated | 高 | 低 | 已验证 API 在 VPP v23.10/v24.10 中仍可用，VPP 社区提前 2-3 年通知废弃 |
| P1.3 替换 ACL 失败 | 高 | 中 | 创建 Git 标签 `v1.0.0-acl-final`，快速回退 |
| 配置验证不充分 | 中 | 中 | 实施 11 条配置验证规则（research.md）|
| 模块本地化失败 | 中 | 低 | 每个模块独立验证，失败立即回退 |
| ACL 代码删除过早 | 低 | 低 | P5 阶段执行，NAT 功能已完全稳定 |

---

## Timeline Estimate

| 阶段 | 工作量 | 依赖 | 预计时间 |
|------|--------|------|----------|
| P0（研究） | - | - | 已完成 |
| P1（设计） | - | P0 | 已完成 |
| P2（任务分解） | 2h | P1 | 待执行 |
| P3（P1.1-P1.3）| 8h | P2 | 1-2 天 |
| P4（P2） | 4h | P3 | 0.5 天 |
| P5（P3.1-P3.2）| 6h | P4 | 1 天 |
| P6（P4） | 4h | P5 | 0.5 天 |
| P7（P5） | 2h | P6 | 0.5 天 |
| **总计** | **26h** | - | **3-5 天** |

**注**:
- 预估基于单人开发
- 包含编码、测试、文档更新、Git 提交
- 不包括等待 CI/CD 或外部依赖的时间

---

## Complexity Tracking

> 无需填充，所有 Constitution Check 均已通过或有明确理由
