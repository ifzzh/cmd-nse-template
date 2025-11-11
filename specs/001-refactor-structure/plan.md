# 实施计划：项目结构重构 - 降低耦合度提升可维护性

**分支**: `001-refactor-structure` | **日期**: 2025-11-11 | **规范**: [spec.md](./spec.md)
**输入**: 特性规范来自 `/specs/001-refactor-structure/spec.md`
**用户约束**: 严格保留原有项目的功能，不允许生成复杂的目录结构

## 概述

将380行耦合度高的main.go重构为模块化架构，提取config、vpp、server、endpoint、registry五个独立模块，将main.go精简，仅负责应用生命周期管理。重构过程严格保持原有功能和逻辑不变，采用渐进式提取策略，确保每个模块独立可测试。

## 技术上下文

**语言/版本**: Go 1.23.8
**核心依赖**:
- `github.com/networkservicemesh/sdk` - NSM端点和客户端SDK
- `github.com/networkservicemesh/sdk-vpp` - VPP网络服务集成（ACL、memif、xconnect）
- `github.com/networkservicemesh/api` - NSM API定义（NetworkService、Registry）
- `go.fd.io/govpp` - VPP Go绑定
- `github.com/spiffe/go-spiffe/v2` - SPIFFE身份认证和TLS
- `google.golang.org/grpc` - gRPC服务器和客户端
- `github.com/kelseyhightower/envconfig` - 环境变量配置解析
- `gopkg.in/yaml.v2` - YAML配置文件解析

**存储**: 文件（ACL配置YAML）
**测试**: Docker多阶段构建（test/debug/release目标）
**目标平台**: Linux（VPP需要Linux内核支持）
**项目类型**: 单一Go应用（网络服务端点）
**性能目标**:
- 构建时间增加<10%
- 运行时性能无变化（重构为零成本抽象）
- 代码理解时间<5分钟/模块

**约束**:
- 必须保持380行main.go的所有功能（6个启动阶段）
- 不允许修改外部API（gRPC接口、配置格式、环境变量）
- 目录结构必须保持扁平化（最大深度≤3层）
- 不允许引入新的框架或重度依赖

**规模/范围**:
- 重构1个主文件（380行）
- 提取5个模块（config、vpp、server、endpoint、registry）
- 目标：main.go精简

## 宪章合规性检查

*门禁：Phase 0研究前必须通过。Phase 1设计后需重新检查。*

### ✅ I. 中文优先原则（强制遵守）
- **检查项**: 所有新增代码注释、日志、文档使用简体中文
- **状态**: ✅ 通过 - 计划中所有文档均为中文，代码标识符使用英文
- **实施方式**:
  - 模块文档注释用中文描述职责和用法
  - 关键逻辑添加中文行内注释
  - 错误信息和日志保持中文
  - 函数名、变量名使用英文（符合Go规范）

### ✅ II. 简洁架构原则（强制遵守）
- **检查项**:
  1. 目录结构扁平化（最大深度≤3层）
  2. 不引入复杂目录结构
  3. 每个目录职责单一明确
- **状态**: ✅ 通过 - **用户明确要求不生成复杂目录结构**
- **实施方式**:
  - **策略A（推荐）**: 所有模块代码直接放在`internal/`下，保持两层结构
    ```
    internal/
    ├── config.go      # 配置管理模块
    ├── vppconn.go     # VPP连接管理模块
    ├── server.go      # gRPC服务器管理模块
    ├── endpoint.go    # 端点构建模块
    └── registry.go    # NSM注册服务模块
    ```
  - **策略B（备选）**: 如果模块需要多个文件，每个模块一个子目录
    ```
    internal/
    ├── config/        # 配置模块
    │   ├── config.go
    │   └── acl.go
    ├── vppconn/       # VPP连接模块
    └── ...
    ```
  - **决策依据**: 优先采用策略A（单文件模块），仅在模块逻辑超过200行时考虑策略B

### ✅ III. 依赖稳定性原则（强制遵守）
- **检查项**: 不修改go.mod模块路径和版本
- **状态**: ✅ 通过 - 重构为纯代码组织变更，不涉及依赖变更
- **实施方式**:
  - 不修改任何依赖版本
  - 不新增外部依赖
  - 仅重新组织现有代码

### ✅ IV. 功能完整性原则（强制遵守）
- **检查项**:
  1. 严格保持原有6个启动阶段的逻辑
  2. 不删除任何功能
  3. 不改变外部行为
- **状态**: ✅ 通过 - 重构目标明确为"代码组织变更"，功能零变化
- **实施方式**:
  - 每个模块提取时，逐行迁移原有代码
  - 保持函数调用顺序和错误处理逻辑
  - 通过对比main.go重构前后的行为验证等价性
  - 核心功能保护清单（全部保留）:
    - ✅ VPP连接和错误处理（vpphelper.StartAndDialContext）
    - ✅ gRPC服务器创建和TLS配置
    - ✅ ACL规则加载和应用（retrieveACLRules）
    - ✅ NSM端点注册（nseRegistryClient.Register）
    - ✅ SPIFFE SVID获取和认证

### ✅ V. 一致性优先原则（强制遵守）
- **检查项**:
  1. 继承现有命名模式
  2. 复用现有工具函数
  3. 遵循现有错误处理方式
- **状态**: ✅ 通过 - 重构保持原有设计模式
- **实施方式**:
  - 命名沿用驼峰命名（如`retrieveACLRules`风格）
  - 错误处理保持`log.Fatal`和`log.Error`模式
  - 配置管理继续使用`envconfig`
  - 日志格式保持`logrus`和嵌套格式化
  - 上下文传递统一使用`context.Context`

### ✅ VI. 本地优先原则（建议遵守）
- **检查项**: 直接在工作目录修改，不创建副本
- **状态**: ✅ 通过 - 就地重构，小步提交
- **实施方式**:
  - 逐模块提取，每次提交一个模块
  - 每次提交后验证构建和测试
  - 不创建临时目录或项目副本

### ✅ VII. 仓库独立性原则（项目战略）
- **检查项**: 独立演化，不依赖上游
- **状态**: ✅ 通过 - 独立重构，不同步上游
- **实施方式**:
  - 重构基于当前main分支
  - 不考虑上游未来变更
  - 保持NSM协议兼容性

### 📊 合规性评分

| 原则 | 强制级别 | 状态 | 备注 |
|-----|---------|------|------|
| I. 中文优先 | 强制 | ✅ 通过 | 所有文档和注释使用中文 |
| II. 简洁架构 | 强制 | ✅ 通过 | 用户明确要求，采用扁平结构 |
| III. 依赖稳定 | 强制 | ✅ 通过 | 不涉及依赖变更 |
| IV. 功能完整 | 强制 | ✅ 通过 | 功能零变化 |
| V. 一致性优先 | 强制 | ✅ 通过 | 继承现有设计模式 |
| VI. 本地优先 | 建议 | ✅ 通过 | 就地重构 |
| VII. 仓库独立 | 战略 | ✅ 通过 | 独立演化 |

**结论**: ✅ **全部通过** - 可进入Phase 0研究阶段

## 项目结构

### 文档（本特性）

```text
specs/001-refactor-structure/
├── spec.md              # 特性规范（已完成）
├── plan.md              # 本文件（实施计划）
├── research.md          # Phase 0输出（技术研究）
├── data-model.md        # Phase 1输出（数据模型）
├── quickstart.md        # Phase 1输出（快速开始）
├── contracts/           # Phase 1输出（接口契约）
├── checklists/          # 质量检查清单
│   └── requirements.md  # 需求检查清单（已完成）
└── tasks.md             # Phase 2输出（由/speckit.tasks生成，不由/speckit.plan创建）
```

### 源代码（仓库根目录）

**当前结构（重构前）**:
```text
cmd-nse-firewall-vpp/
├── main.go              # 380行，包含所有逻辑（需重构）
├── internal/
│   └── imports/         # 导入声明（保持不变）
├── go.mod               # Go模块定义
├── go.sum               # 依赖校验和
├── Dockerfile           # 容器化构建
└── ...                  # 其他配置文件
```

**目标结构（重构后）**:

**策略A（推荐）- 扁平化单文件模块**:
```text
cmd-nse-firewall-vpp/
├── main.go              # ~100行，仅应用生命周期管理
├── internal/
│   ├── imports/         # 导入声明（已有，保持）
│   ├── config.go        # 配置管理模块（~60行）
│   ├── vppconn.go       # VPP连接管理模块（~50行）
│   ├── server.go        # gRPC服务器管理模块（~60行）
│   ├── endpoint.go      # 端点构建模块（~80行）
│   └── registry.go      # NSM注册服务模块（~50行）
├── go.mod               # 不变
├── go.sum               # 不变
└── Dockerfile           # 不变
```

**策略B（备选）- 模块子目录**（仅在单文件模块超过200行时考虑）:
```text
internal/
├── imports/
├── config/
│   ├── config.go        # 配置结构和环境变量解析
│   └── acl.go           # ACL规则加载
├── vppconn/
│   └── manager.go       # VPP连接管理器
├── server/
│   └── grpc.go          # gRPC服务器包装器
├── endpoint/
│   └── builder.go       # 端点构建器
└── registry/
    └── service.go       # 注册服务
```

**结构决策**:
- ✅ **采用策略A（扁平化单文件模块）** - 符合"简洁架构原则"和用户约束
- **理由**:
  1. 用户明确要求"不允许生成复杂的目录结构"
  2. 每个模块预计50-80行，无需拆分为多文件
  3. 保持internal/目录两层结构（最大深度=3层：root → internal → *.go）
  4. 便于理解和导航，文件数从1个增加到6个（可接受）
- **例外情况**: 如果在实施过程中发现某模块逻辑复杂度高（>200行），再考虑拆分为子目录

**模块职责划分**:
| 模块 | 文件 | 行数 | 职责 | 主要函数 |
|-----|------|------|------|---------|
| **config** | `internal/config.go` | ~60 | 配置管理：环境变量解析、ACL规则加载 | `LoadConfig()`, `RetrieveACLRules()` |
| **vppconn** | `internal/vppconn.go` | ~50 | VPP连接管理：连接建立、错误监控、生命周期 | `NewVPPManager()`, `Start()`, `GetConn()` |
| **server** | `internal/server.go` | ~60 | gRPC服务器：创建、TLS配置、启动、停止 | `NewGRPCServer()`, `Start()`, `Stop()` |
| **endpoint** | `internal/endpoint.go` | ~80 | 端点构建：组装NSM防火墙端点的chain | `NewFirewallEndpoint()` |
| **registry** | `internal/registry.go` | ~50 | NSM注册：注册客户端创建、端点注册、注销 | `NewRegistryClient()`, `Register()` |
| **main** | `main.go` | ~100 | 应用协调：初始化各模块、生命周期管理 | `main()` |

## 复杂度跟踪

> **仅在宪章合规性检查有需要说明的违规时填写**

**本重构无违规项** - 所有宪章原则均通过检查，无需复杂度说明。

---

## Phase 0: 研究阶段 ✅ 已完成

### 输出文件
- ✅ [research.md](./research.md) - 技术决策和最佳实践研究

### 关键决策
1. **模块提取策略**: 渐进式提取，每次一个模块
2. **目录结构**: 扁平化单文件模块（internal/*.go）
3. **接口设计**: 构造函数 + 简单方法，无接口抽象
4. **错误处理**: 保持log.Fatal和error返回模式
5. **日志记录**: 保持logrus + 上下文logger
6. **测试策略**: 保持Docker测试模式

### NEEDS CLARIFICATION解决情况
- ✅ 所有技术细节已明确
- ✅ 无遗留的NEEDS CLARIFICATION标记

---

## Phase 1: 设计阶段 ✅ 已完成

### 输出文件
- ✅ [data-model.md](./data-model.md) - 数据模型定义
- ✅ [contracts/internal-modules.md](./contracts/internal-modules.md) - 内部模块接口契约
- ✅ [quickstart.md](./quickstart.md) - 快速开始指南

### 数据模型总结
- **Config**: 配置对象（不可变，加载后只读）
- **VPPManager**: VPP连接管理器（生命周期管理）
- **GRPCServer**: gRPC服务器包装器（TLS和socket管理）
- **FirewallEndpoint**: 防火墙端点（chain构建）
- **RegistryClient**: NSM注册客户端（端点注册）

### 接口契约总结
- **config模块**: LoadConfig(ctx) → (*Config, error)
- **vppconn模块**: StartVPP(ctx) → (*VPPManager, error)
- **server模块**: NewGRPCServer(ctx, config, tlsConfig) → (*GRPCServer, error)
- **endpoint模块**: NewFirewallEndpoint(ctx, config, vppConn, source, clientOptions) → *FirewallEndpoint
- **registry模块**: NewRegistryClient(ctx, config, clientOptions) → registryapi.NetworkServiceEndpointRegistryClient

### Agent上下文更新
- ✅ 已运行`.specify/scripts/bash/update-agent-context.sh claude`
- ✅ CLAUDE.md已更新，包含本特性的技术栈信息

---

## 宪章合规性复查（Phase 1设计后）

*门禁：Phase 1设计完成后必须重新检查宪章合规性*

### ✅ I. 中文优先原则（强制遵守）
- **检查项**: 所有设计文档、接口注释使用简体中文
- **状态**: ✅ 通过
- **验证**:
  - research.md全中文 ✅
  - data-model.md全中文 ✅
  - contracts/internal-modules.md全中文 ✅
  - quickstart.md全中文 ✅

### ✅ II. 简洁架构原则（强制遵守）
- **检查项**: 设计中的目录结构符合扁平化要求
- **状态**: ✅ 通过
- **验证**:
  - 目标结构：internal/*.go（5个文件）
  - 最大深度：3层（root → internal → *.go）
  - 无复杂子目录嵌套 ✅

### ✅ III. 依赖稳定性原则（强制遵守）
- **检查项**: 设计中不引入新依赖
- **状态**: ✅ 通过
- **验证**:
  - 所有依赖来自现有go.mod ✅
  - 无新增外部依赖 ✅

### ✅ IV. 功能完整性原则（强制遵守）
- **检查项**: 设计保持功能完全等价
- **状态**: ✅ 通过
- **验证**:
  - 所有接口覆盖main.go的6个启动阶段 ✅
  - Config结构体字段与main.go完全一致 ✅
  - VPP连接、gRPC服务器、端点、注册逻辑全部保留 ✅

### ✅ V. 一致性优先原则（强制遵守）
- **检查项**: 设计继承现有代码风格
- **状态**: ✅ 通过
- **验证**:
  - 接口设计遵循现有模式（构造函数返回对象）✅
  - 错误处理保持log.Fatal和error返回 ✅
  - 日志记录保持logrus模式 ✅

### ✅ VI. 本地优先原则（建议遵守）
- **检查项**: 设计支持就地重构
- **状态**: ✅ 通过
- **验证**:
  - quickstart.md明确采用渐进式提取 ✅
  - 每个阶段独立提交 ✅

### ✅ VII. 仓库独立性原则（项目战略）
- **检查项**: 设计不依赖上游
- **状态**: ✅ 通过
- **验证**:
  - 设计基于当前main分支 ✅
  - 不考虑上游未来变更 ✅

### 📊 合规性评分（Phase 1设计后）

| 原则 | 强制级别 | 状态 | 备注 |
|-----|---------|------|------|
| I. 中文优先 | 强制 | ✅ 通过 | 所有设计文档使用中文 |
| II. 简洁架构 | 强制 | ✅ 通过 | 扁平化结构，无复杂目录 |
| III. 依赖稳定 | 强制 | ✅ 通过 | 不引入新依赖 |
| IV. 功能完整 | 强制 | ✅ 通过 | 功能完全等价 |
| V. 一致性优先 | 强制 | ✅ 通过 | 继承现有设计模式 |
| VI. 本地优先 | 建议 | ✅ 通过 | 支持就地重构 |
| VII. 仓库独立 | 战略 | ✅ 通过 | 独立设计 |

**结论**: ✅ **全部通过** - 设计符合所有宪章原则，可进入Phase 2（任务生成阶段）

---

## Phase 2: 任务生成（由/speckit.tasks命令执行）

**注意**: Phase 2任务生成由`/speckit.tasks`命令执行，不由`/speckit.plan`命令创建。

本实施计划（plan.md）在此结束。下一步请运行：
```bash
/speckit.tasks
```

---

## 计划总结

### 生成的文档
1. ✅ [plan.md](./plan.md) - 本文件（实施计划）
2. ✅ [research.md](./research.md) - 技术研究
3. ✅ [data-model.md](./data-model.md) - 数据模型
4. ✅ [contracts/internal-modules.md](./contracts/internal-modules.md) - 接口契约
5. ✅ [quickstart.md](./quickstart.md) - 快速开始指南

### 关键决策
- **目录结构**: 扁平化单文件模块（internal/*.go）
- **模块数量**: 5个（config、vppconn、server、endpoint、registry）
- **main.go目标**: 100行以内（从380行减少73%）
- **提取策略**: 渐进式，每次一个模块，立即验证

### 风险缓解
- 逐模块提取降低风险
- 每次提交后验证构建和功能
- 保持代码逻辑完全一致
- 对比重构前后的日志输出

### 符合宪章
- ✅ 中文优先原则：所有文档和设计使用中文
- ✅ 简洁架构原则：扁平化结构，无复杂目录
- ✅ 依赖稳定性原则：不修改依赖，不新增依赖
- ✅ 功能完整性原则：功能零变化
- ✅ 一致性优先原则：继承现有设计模式
- ✅ 本地优先原则：就地重构，小步提交
- ✅ 仓库独立性原则：独立演化

### 预期成果
- main.go从380行精简至100行以内（73%减少）
- 5个独立模块，职责明确
- 功能完全保持不变
- 代码可读性大幅提升

### 下一步
运行 `/speckit.tasks` 生成详细的开发任务清单。

---

**实施计划完成时间**: 2025-11-11
**计划版本**: 1.0
