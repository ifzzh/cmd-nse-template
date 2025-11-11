# 任务清单：项目结构重构 - 降低耦合度提升可维护性

**输入**: 设计文档来自 `/specs/001-refactor-structure/`
**前置条件**: plan.md（必需）, spec.md（必需），research.md, data-model.md, contracts/, quickstart.md

**测试**: 本项目不包含新增单元测试（保持现有Docker测试模式）

**组织方式**: 任务按用户场景分组，以支持独立实施和验证每个场景

---

## 格式说明：`[ID] [P?] [Story] 描述`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户场景（例如 US1, US2, US3）
- 包含确切的文件路径

---

## 路径约定

本项目为**单一Go应用**，采用扁平化目录结构：
- 源代码：`main.go`（根目录）+ `internal/*.go`
- 无tests/目录（使用Docker多阶段构建测试）
- 配置文件：根目录（`.golangci.yml`等）

---

## Phase 1: 准备阶段（项目初始化）

**目的**: 重构前的准备工作和备份

- [X] T001 备份main.go为main.go.backup用于对比
- [X] T002 验证Go版本（应为1.23.8）和构建工具链
- [X] T003 [P] 验证Docker环境（用于后续测试）
- [X] T004 确认当前分支为001-refactor-structure
- [X] T005 记录当前main.go行数（应为379行）作为基线

**检查点**: 准备就绪 - 可以开始模块提取

---

## Phase 2: 基础验证（阻塞性前置条件）

**目的**: 验证现有功能正常，建立验证基线

**⚠️ 关键**: 在任何重构前，必须完成此阶段以确保功能完整性

- [X] T006 运行`go build ./...`验证当前代码可构建
- [ ] T007 运行Docker test目标验证现有功能（如果存在）
- [ ] T008 [P] 记录6个启动阶段的日志输出作为对比基线
- [ ] T009 验证golangci-lint配置并运行静态检查

**检查点**: 基线建立 - 重构可以开始

---

## Phase 3: 用户场景1 - 将配置管理独立为模块 (优先级: P1) 🎯 MVP

**目标**: 提取Config结构体和配置加载逻辑到`internal/config.go`，实现配置管理的独立模块化

**独立测试**:
- 环境变量NSM_NAME=test-firewall时，配置正确解析
- ACL配置文件存在时，规则正确加载
- 日志级别设置生效
- 配置文件缺失时记录错误但不中断启动

### 实施任务 - 用户场景1

- [X] T010 [P] [US1] 创建internal/config.go文件
- [X] T011 [US1] 将Config结构体从main.go（行87-103）迁移到internal/config.go
- [X] T012 [US1] 将Config.Process方法（行106-114）迁移并重构为LoadConfig函数
- [X] T013 [US1] 将retrieveACLRules函数（行356-379）迁移到internal/config.go作为私有函数
- [X] T014 [US1] 为Config结构体和LoadConfig函数添加中文文档注释
- [X] T015 [US1] 修改main.go使用internal.LoadConfig(ctx)替代原有配置加载逻辑（行148-164）
- [X] T016 [US1] 删除main.go中的Config结构体定义、Process方法和retrieveACLRules函数
- [X] T017 [US1] 验证构建：`go build ./...`
- [X] T018 [US1] 验证配置加载功能：设置环境变量并运行，检查日志输出
- [X] T019 [US1] Git提交：`git commit -m "重构: 提取config模块到internal/config.go"`

**检查点**: 配置模块独立工作，main.go精简约60行

---

## Phase 4: 用户场景2 - 将VPP连接管理提取为独立层 (优先级: P2)

**目标**: 封装VPP连接建立、错误监控和生命周期管理到`internal/vppconn.go`

**独立测试**:
- VPP服务运行时，成功建立连接并返回有效连接对象
- VPP服务未运行时，返回明确错误信息
- VPP运行时崩溃时，错误通道接收到连接断开事件
- 应用关闭时，VPP连接正确关闭

### 实施任务 - 用户场景2

- [ ] T020 [P] [US2] 创建internal/vppconn.go文件
- [ ] T021 [US2] 定义VPPManager结构体（封装conn、errCh、cancel）
- [ ] T022 [US2] 实现StartVPP函数封装vpphelper.StartAndDialContext逻辑（main.go行229-230）
- [ ] T023 [US2] 实现VPPManager.GetConnection方法返回VPP连接对象
- [ ] T024 [US2] 实现VPPManager.GetErrorChannel方法返回错误通道
- [ ] T025 [US2] 封装exitOnErr逻辑到VPPManager内部监控goroutine
- [ ] T026 [US2] 为VPPManager和相关函数添加中文文档注释
- [ ] T027 [US2] 修改main.go使用internal.StartVPP(ctx)替代vpphelper直接调用
- [ ] T028 [US2] 将main.go中所有使用vppConn的地方改为vppManager.GetConnection()
- [ ] T029 [US2] 验证构建和VPP连接功能
- [ ] T030 [US2] Git提交：`git commit -m "重构: 提取vppconn模块到internal/vppconn.go"`

**检查点**: VPP连接管理独立工作，错误监控正常

---

## Phase 5: 用户场景3 - 将gRPC服务器配置和启动逻辑模块化 (优先级: P3)

**目标**: 封装gRPC服务器创建、TLS配置、启动和停止逻辑到`internal/server.go`

**独立测试**:
- SPIFFE证书就绪时，TLS配置正确应用
- 服务器启动后，客户端可连接并调用服务方法
- 服务器监听Unix socket，socket文件在临时目录创建
- 应用关闭时，服务器优雅关闭并清理临时文件

### 实施任务 - 用户场景3

- [ ] T031 [P] [US3] 创建internal/server.go文件
- [ ] T032 [US3] 定义GRPCServer结构体（封装server、listenOn、tmpDir）
- [ ] T033 [US3] 实现NewGRPCServer函数创建临时目录和gRPC服务器（main.go行268-285）
- [ ] T034 [US3] 实现GRPCServer.GetServer方法返回grpc.Server实例
- [ ] T035 [US3] 实现GRPCServer.GetListenURL方法返回监听地址URL
- [ ] T036 [US3] 实现GRPCServer.ListenAndServe方法启动服务器监听（封装grpcutils.ListenAndServe）
- [ ] T037 [US3] 实现GRPCServer.Cleanup方法清理临时目录
- [ ] T038 [US3] 为GRPCServer和相关方法添加中文文档注释
- [ ] T039 [US3] 修改main.go使用internal.NewGRPCServer创建服务器
- [ ] T040 [US3] 修改main.go使用grpcServer.GetServer()、GetListenURL()和ListenAndServe()
- [ ] T041 [US3] 添加defer grpcServer.Cleanup()确保资源清理
- [ ] T042 [US3] 验证构建和gRPC服务器启动功能
- [ ] T043 [US3] Git提交：`git commit -m "重构: 提取server模块到internal/server.go"`

**检查点**: gRPC服务器管理独立工作，TLS配置和生命周期管理正常

---

## Phase 6: 用户场景4 - 将NSM端点创建逻辑封装为工厂模式 (优先级: P4)

**目标**: 封装NSM防火墙端点的chain构建逻辑到`internal/endpoint.go`

**独立测试**:
- 提供完整配置和VPP连接时，返回包含所有必需组件的端点实例
- 配置中指定ACL规则时，ACL组件正确配置并包含在chain中
- 端点创建后注册到gRPC服务器，NetworkService服务可被远程调用
- 缺少必需依赖时，返回明确错误信息

### 实施任务 - 用户场景4

- [ ] T044 [P] [US4] 创建internal/endpoint.go文件
- [ ] T045 [US4] 定义FirewallEndpoint结构体（嵌入endpoint.Endpoint）
- [ ] T046 [US4] 实现NewFirewallEndpoint函数封装60+行的chain构建逻辑（main.go行232-264）
- [ ] T047 [US4] 在NewFirewallEndpoint中配置endpoint.WithName、WithAuthorizeServer等选项
- [ ] T048 [US4] 配置AdditionalFunctionality：recvfd、sendfd、up、clienturl、xconnect、acl
- [ ] T049 [US4] 配置mechanisms.NewServer和memif.MECHANISM
- [ ] T050 [US4] 配置connect.NewServer和内部client.NewClient
- [ ] T051 [US4] 实现FirewallEndpoint.Register方法注册到gRPC服务器
- [ ] T052 [US4] 为FirewallEndpoint和NewFirewallEndpoint添加中文文档注释
- [ ] T053 [US4] 修改main.go使用internal.NewFirewallEndpoint创建端点
- [ ] T054 [US4] 修改main.go使用firewallEndpoint.Register(grpcServer.GetServer())
- [ ] T055 [US4] 验证构建和端点创建功能
- [ ] T056 [US4] Git提交：`git commit -m "重构: 提取endpoint模块到internal/endpoint.go"`

**检查点**: 端点构建逻辑独立工作，chain组装正确

---

## Phase 7: 用户场景5 - 将NSM注册逻辑提取为服务模块 (优先级: P5)

**目标**: 封装NSM Registry连接、端点注册和策略配置逻辑到`internal/registry.go`

**独立测试**:
- NSM Manager可访问时，端点成功注册并返回注册信息
- 注册策略文件存在时，策略正确加载并应用
- 网络断开时，返回连接错误并允许重试
- 应用关闭时，端点从Registry中移除

### 实施任务 - 用户场景5

- [X] T057 [P] [US5] 创建internal/registry.go文件
- [X] T058 [US5] 实现NewRegistryClient函数创建NSM注册客户端（main.go行295-304）
- [X] T059 [US5] 在NewRegistryClient中配置ClientURL、DialOptions和AdditionalFunctionality
- [X] T060 [US5] 配置授权策略WithAuthorizeNSERegistryClient
- [X] T061 [US5] 实现RegisterEndpoint函数向NSM Manager注册端点（main.go行305-319）
- [X] T062 [US5] 在RegisterEndpoint中构造NetworkServiceEndpoint对象
- [X] T063 [US5] 为NewRegistryClient和RegisterEndpoint添加中文文档注释
- [X] T064 [US5] 修改main.go使用internal.NewRegistryClient创建注册客户端
- [X] T065 [US5] 修改main.go使用internal.RegisterEndpoint注册端点
- [X] T066 [US5] 验证构建和端点注册功能
- [X] T067 [US5] Git提交：`git commit -m "重构: 提取registry模块到internal/registry.go"`

**检查点**: NSM注册逻辑独立工作，端点注册和策略应用正常

---

## Phase 8: 主程序重构（最终精简）

**目标**: 将main.go重构为简洁的应用协调层（≤100行），仅负责模块初始化和生命周期管理

**独立测试**: 完整应用运行，6个启动阶段日志输出与重构前一致

### 实施任务 - 主程序重构

- [ ] T068 [P] [US-MAIN] 重组main.go的import语句，添加internal包导入
- [ ] T069 [US-MAIN] 简化main函数，使用各模块的导出函数替代内联逻辑
- [ ] T070 [US-MAIN] 确保main.go仅保留以下职责：context设置、日志初始化、模块调用、生命周期管理
- [ ] T071 [US-MAIN] 保留exitOnErr和notifyContext辅助函数（不需提取）
- [ ] T072 [US-MAIN] 验证main.go行数≤100行
- [ ] T073 [US-MAIN] 运行`go build ./...`验证构建
- [ ] T074 [US-MAIN] 运行Docker test目标验证完整功能
- [ ] T075 [US-MAIN] 对比重构前后的日志输出，确认6个启动阶段完全一致
- [ ] T076 [US-MAIN] 记录最终行数统计（main.go和各模块文件）
- [ ] T077 [US-MAIN] Git提交：`git commit -m "重构: 重构main.go为简洁的应用协调层"`

**检查点**: main.go精简至100行以内，功能完全保持不变

---

## Phase 9: 最终验证和文档（跨场景关注点）

**目的**: 全面验证重构成果，确保质量和完整性

- [ ] T078 [P] 运行golangci-lint验证代码质量
- [ ] T079 [P] 验证无循环依赖：`go mod graph | grep "cmd-nse-firewall-vpp/internal"`
- [ ] T080 验证构建时间增加<10%（对比重构前）
- [ ] T081 验证注释覆盖率≥80%（所有导出函数有中文文档注释）
- [ ] T082 [P] 对比main.go.backup和main.go，确认功能等价性
- [ ] T083 [P] 验证所有成功标准（SC-001到SC-008）
- [ ] T084 更新CLAUDE.md Agent上下文（如果需要）
- [ ] T085 [P] 更新operations-log.md记录重构完成
- [ ] T086 运行quickstart.md中的完整验证清单
- [ ] T087 删除main.go.backup备份文件（可选，建议保留一段时间）

**检查点**: 重构完成，所有质量标准通过

---

## 依赖关系与执行顺序

### 阶段依赖

- **准备阶段（Phase 1）**: 无依赖 - 可立即开始
- **基础验证（Phase 2）**: 依赖准备阶段完成 - **阻塞所有用户场景**
- **用户场景（Phase 3-7）**: 全部依赖基础验证完成
  - 用户场景按依赖关系**顺序执行**（不可并行）：US1 → US2 → US3 → US4 → US5
  - 理由：后续场景依赖前面场景提取的模块
- **主程序重构（Phase 8）**: 依赖所有用户场景（US1-US5）完成
- **最终验证（Phase 9）**: 依赖主程序重构完成

### 用户场景依赖关系

```
US1（config）: 无依赖 - 可在基础验证后立即开始
  ↓
US2（vppconn）: 无依赖（但建议在US1后，验证模块提取流程成功）
  ↓
US3（server）: 依赖US1（需要config模块）
  ↓
US4（endpoint）: 依赖US1（config）和US2（vppconn）
  ↓
US5（registry）: 依赖US1（config）和US3（server的GetListenURL）
  ↓
主程序重构: 依赖所有模块（US1-US5）
```

**⚠️ 关键约束**:
- 用户场景必须**顺序执行**，不可并行（每个场景提取一个模块，后续场景依赖前面的提取结果）
- 每个用户场景完成后**立即提交**，确保增量验证

### 每个用户场景内部

- 创建文件（标记[P]）可并行
- 实现逻辑按顺序执行（定义结构体 → 实现函数 → 修改main.go）
- 验证和提交在最后顺序执行

### 并行机会

**准备阶段内部**:
- T002（验证Go版本）和T003（验证Docker环境）可并行

**基础验证阶段内部**:
- T008（记录日志）和T009（golangci-lint）可并行

**每个用户场景内部**:
- 创建文件任务（标记[P]）可并行

**最终验证阶段内部**:
- T078（golangci-lint）、T079（循环依赖检查）、T081（注释覆盖率）、T082（对比备份）、T083（成功标准）、T085（operations-log）可并行

---

## 并行示例：用户场景1

```bash
# 并行创建文件（同时执行）:
Task: "创建internal/config.go文件"

# 顺序实施逻辑（按依赖顺序）:
Task: "将Config结构体迁移到internal/config.go"
Task: "将Config.Process方法重构为LoadConfig函数"
Task: "将retrieveACLRules函数迁移"
Task: "添加中文文档注释"
Task: "修改main.go使用internal.LoadConfig"
Task: "删除main.go中的原有定义"

# 顺序验证:
Task: "验证构建"
Task: "验证配置加载功能"
Task: "Git提交"
```

---

## 实施策略

### MVP优先（仅用户场景1）

1. 完成Phase 1：准备阶段
2. 完成Phase 2：基础验证（**关键** - 阻塞所有场景）
3. 完成Phase 3：用户场景1（配置模块）
4. **停止并验证**：独立测试用户场景1
5. 如果通过，继续或演示MVP

### 渐进式交付

1. 完成准备 + 基础验证 → 基础就绪
2. 添加用户场景1 → 独立测试 → 提交（配置模块独立工作！）
3. 添加用户场景2 → 独立测试 → 提交（VPP连接管理独立工作！）
4. 添加用户场景3 → 独立测试 → 提交（gRPC服务器独立工作！）
5. 添加用户场景4 → 独立测试 → 提交（端点构建独立工作！）
6. 添加用户场景5 → 独立测试 → 提交（NSM注册独立工作！）
7. 重构main.go → 完整测试 → 提交（应用协调层完成！）
8. 每个场景增加价值，不破坏已完成的场景

### 单人开发策略

本重构为**单人顺序执行**模式：

1. 开发者完成准备 + 基础验证
2. 依次完成用户场景1-5（顺序执行，不可跳过）
3. 重构main.go
4. 最终验证

**预计总时间**: 8-12小时（每个用户场景1-2小时）

---

## 注意事项

- **[P]任务** = 不同文件，无依赖，可并行
- **[Story]标签** = 将任务映射到特定用户场景，便于追溯
- **用户场景必须顺序执行** - 后续场景依赖前面提取的模块
- 每个用户场景完成后立即提交（小步提交）
- 在每个检查点验证场景独立工作
- 避免：模糊任务、同文件冲突、跨场景依赖破坏独立性

---

## 任务统计

### 总任务数
- **总计**: 87个任务

### 按阶段统计
- **Phase 1（准备）**: 5个任务
- **Phase 2（基础验证）**: 4个任务
- **Phase 3（US1 - 配置模块）**: 10个任务
- **Phase 4（US2 - VPP连接）**: 11个任务
- **Phase 5（US3 - gRPC服务器）**: 13个任务
- **Phase 6（US4 - 端点构建）**: 13个任务
- **Phase 7（US5 - NSM注册）**: 11个任务
- **Phase 8（主程序重构）**: 10个任务
- **Phase 9（最终验证）**: 10个任务

### 按用户场景统计
- **US1（配置管理）**: 10个任务
- **US2（VPP连接管理）**: 11个任务
- **US3（gRPC服务器）**: 13个任务
- **US4（端点构建）**: 13个任务
- **US5（NSM注册）**: 11个任务
- **US-MAIN（主程序重构）**: 10个任务

### 并行机会
- **可并行任务**: 10个任务标记为[P]
- **主要并行阶段**: 准备阶段、最终验证阶段内部任务
- **用户场景执行**: 顺序执行（不可并行）

---

## 建议的MVP范围

**MVP = 用户场景1（配置模块独立）**

完成Phase 1-3后，您将拥有：
- ✅ 独立的配置管理模块（internal/config.go）
- ✅ main.go精简约60行
- ✅ 配置加载功能完整且可独立测试
- ✅ 验证了模块提取流程的可行性

**MVP验收标准**:
1. internal/config.go文件存在且约60行
2. main.go精简约60行（从380行到320行）
3. `go build ./...`构建成功
4. 配置加载功能正常（通过日志验证）
5. Git提交记录清晰

**后续迭代**: 在MVP成功后，依次完成US2-US5和主程序重构

---

## 格式验证

✅ **所有任务遵循清单格式**:
- ✅ 每个任务以`- [ ]`开头（markdown复选框）
- ✅ 任务ID顺序编号（T001到T087）
- ✅ [P]标记用于可并行任务
- ✅ [Story]标签映射到用户场景（US1-US5、US-MAIN）
- ✅ 描述包含确切文件路径或明确操作
- ✅ 准备和验证阶段无[Story]标签
- ✅ 用户场景阶段必须有[Story]标签

---

**生成时间**: 2025-11-11
**任务清单版本**: 1.0
