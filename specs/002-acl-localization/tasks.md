# Tasks: ACL 模块本地化

**Feature**: 002-acl-localization | **Date**: 2025-11-13 | **Status**: Ready for Implementation

## 概述

本任务清单基于 quickstart.md 的详细操作步骤生成，严格按照用户故事组织，确保每次仅本地化一个模块，实现增量验证和版本控制。

**实施策略**: MVP-first，每个用户故事独立可测，迭代验证。

---

## 用户故事 US1: 单个 ACL 模块本地化 (Priority: P1)

### 迭代 1: 本地化 binapi/acl_types

#### 前置准备

- [X] [T001] [US1] 验证前置条件：检查 Go 版本 (1.23.8+)、Docker 可用性、Git 状态干净，确认 go.sum 中 govpp 版本为 v0.0.0-20240328101142-8a444680fbba - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/go.sum`
- [X] [T002] [US1] 确认基线镜像存在：拉取 ifzzh520/vpp-acl-firewall:v1.0.0 - Docker 镜像仓库

#### 模块下载与复制

- [X] [T003] [US1] 下载 govpp 模块到缓存：执行 `go mod download github.com/networkservicemesh/govpp` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [X] [T004] [US1] 定位并验证缓存目录：查找 `$GOPATH/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba/binapi/acl_types/` - Go 模块缓存
- [X] [T005] [US1] 创建本地模块目录：`mkdir -p internal/binapi_acl_types` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal`
- [X] [T006] [US1] 复制源码文件：从缓存复制 `binapi/acl_types/*` 到 `internal/binapi_acl_types/` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl_types`
- [X] [T007] [US1] 取消只读权限：`chmod -R u+w internal/binapi_acl_types/` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl_types`

#### 模块配置

- [X] [T008] [US1] 创建 go.mod 文件：声明模块 `github.com/ifzzh/cmd-nse-template/internal/binapi_acl_types`，依赖 `go.fd.io/govpp v0.11.0` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl_types/go.mod`
- [X] [T009] [US1] 执行 go mod tidy：自动补全依赖并生成 go.sum - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl_types`
- [X] [T010] [US1] 添加包级别中文注释：在 `acl_types.ba.go` 开头添加中文文档注释，说明模块来源、版本哈希和注意事项 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl_types/acl_types.ba.go`
- [X] [T011] [US1] 创建 README.md：记录模块来源信息（仓库、版本、哈希、本地化日期）、文件清单、依赖关系、修改说明和升级指南 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl_types/README.md`

#### 项目集成

- [X] [T012] [US1] 备份项目 go.mod：`cp go.mod go.mod.backup` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [X] [T013] [US1] 修改项目 go.mod：在末尾添加 replace 指令 `replace github.com/networkservicemesh/govpp/binapi/acl_types => ./internal/binapi_acl_types` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/go.mod`

#### 本地验证

- [X] [T014] [US1] 验证 replace 生效：`go list -m github.com/networkservicemesh/govpp/binapi/acl_types`，期望输出包含 `=> ./internal/binapi_acl_types` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [X] [T015] [US1] 编译项目：`go build ./...`，确保编译成功无错误 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [X] [T016] [US1] 运行单元测试（如存在）：`go test ./...`，记录测试结果 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`

#### 版本控制与镜像构建

- [X] [T017] [US1] 创建 VERSION 文件：写入 "v1.0.1" - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/VERSION`
- [ ] [T018] [US1] Git 提交本地化代码：添加 `internal/binapi_acl_types/`、`go.mod`、`go.sum`、`VERSION`，提交信息包含模块名称、来源、版本哈希和目标镜像版本 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T019] [US1] 构建 Docker 镜像：`docker build -t ifzzh520/vpp-acl-firewall:v1.0.1 -t ifzzh520/vpp-acl-firewall:latest .` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T020] [US1] 验证镜像构建成功：`docker images | grep vpp-acl-firewall`，确认 v1.0.1 标签存在 - Docker

#### 推送与标记

- [ ] [T021] [US1] 推送镜像到仓库：`docker push ifzzh520/vpp-acl-firewall:v1.0.1` 和 `docker push ifzzh520/vpp-acl-firewall:latest` - Docker 镜像仓库
- [ ] [T022] [US1] 打 Git tag：`git tag v1.0.1 -m "ACL 模块本地化迭代 1: binapi/acl_types"` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T023] [US1] 推送代码和标签：`git push origin v1.0.1 && git push origin HEAD` - Git 远程仓库

---

## 用户故事 US2: 增量本地化与测试验证 (Priority: P2)

### 用户测试 - 迭代 1

- [ ] [T024] [US2] 等待用户测试迭代 1：用户部署 v1.0.1 镜像并执行功能验证（NSM 注册、VPP 连接、ACL 规则、网络流量、日志检查） - 测试环境
- [ ] [T025] [US2] 确认迭代 1 测试通过：收集用户测试反馈，确认无功能回归和日志正常 - 测试报告

### 迭代 2: 本地化 binapi/acl（依赖迭代 1 成功）

#### 模块下载与复制

- [ ] [T026] [P] [US2] 复用 govpp 缓存：验证缓存目录 `$GOPATH/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba/binapi/acl/` 存在 - Go 模块缓存
- [ ] [T027] [US2] 创建本地模块目录：`mkdir -p internal/binapi_acl` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal`
- [ ] [T028] [US2] 复制源码文件：从缓存复制 `binapi/acl/*` 到 `internal/binapi_acl/` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl`
- [ ] [T029] [US2] 取消只读权限：`chmod -R u+w internal/binapi_acl/` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl`

#### 模块配置

- [ ] [T030] [US2] 创建 go.mod 文件：声明模块 `github.com/ifzzh/cmd-nse-template/internal/binapi_acl`，依赖 `go.fd.io/govpp v0.11.0` 和 `github.com/networkservicemesh/govpp/binapi/acl_types v0.0.0-20240328101142-8a444680fbba` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl/go.mod`
- [ ] [T031] [US2] 执行 go mod tidy：自动补全依赖并生成 go.sum - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl`
- [ ] [T032] [US2] 添加包级别中文注释：在 `acl.ba.go` 开头添加中文文档注释，说明模块来源、版本哈希和注意事项 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl/acl.ba.go`
- [ ] [T033] [US2] 创建 README.md：记录模块来源信息、文件清单（acl.ba.go, acl_rpc.ba.go）、依赖关系、修改说明和升级指南 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_acl/README.md`

#### 项目集成

- [ ] [T034] [US2] 更新项目 go.mod：在 replace 部分新增 `replace github.com/networkservicemesh/govpp/binapi/acl => ./internal/binapi_acl` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/go.mod`

#### 本地验证

- [ ] [T035] [US2] 验证 replace 生效：`go list -m github.com/networkservicemesh/govpp/binapi/acl`，期望输出包含 `=> ./internal/binapi_acl` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T036] [US2] 编译项目：`go build ./...`，确保编译成功无错误 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T037] [US2] 运行单元测试（如存在）：`go test ./...`，记录测试结果 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`

#### 版本控制与镜像构建

- [ ] [T038] [US2] 更新 VERSION 文件：写入 "v1.0.2" - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/VERSION`
- [ ] [T039] [US2] Git 提交本地化代码：添加 `internal/binapi_acl/`、`go.mod`、`go.sum`、`VERSION`，提交信息包含模块名称、依赖关系和目标镜像版本 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T040] [US2] 构建 Docker 镜像：`docker build -t ifzzh520/vpp-acl-firewall:v1.0.2 -t ifzzh520/vpp-acl-firewall:latest .` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T041] [US2] 验证镜像构建成功：`docker images | grep vpp-acl-firewall`，确认 v1.0.2 标签存在 - Docker

#### 推送与标记

- [ ] [T042] [US2] 推送镜像到仓库：`docker push ifzzh520/vpp-acl-firewall:v1.0.2` 和 `docker push ifzzh520/vpp-acl-firewall:latest` - Docker 镜像仓库
- [ ] [T043] [US2] 打 Git tag：`git tag v1.0.2 -m "ACL 模块本地化迭代 2: binapi/acl"` - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp`
- [ ] [T044] [US2] 推送代码和标签：`git push origin v1.0.2 && git push origin HEAD` - Git 远程仓库

### 用户测试 - 迭代 2

- [ ] [T045] [US2] 等待用户测试迭代 2：用户部署 v1.0.2 镜像并执行完整功能验证 - 测试环境
- [ ] [T046] [US2] 确认迭代 2 测试通过：收集用户测试反馈，确认所有 ACL 模块功能正常 - 测试报告

---

## 用户故事 US3: 版本一致性维护 (Priority: P3)

### 版本追溯与文档维护

- [ ] [T047] [US3] 验证版本一致性：对比 `internal/binapi_acl_types/` 和 `internal/binapi_acl/` 代码与缓存源码的 diff，确认仅有预期修改（注释、go.mod、README.md） - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal`
- [ ] [T048] [US3] 记录版本映射：在项目文档中记录镜像版本与本地化模块的对应关系（v1.0.1 → binapi/acl_types, v1.0.2 → binapi/acl） - 项目文档或 README
- [ ] [T049] [US3] 生成最终目录结构总览：验证 internal/ 目录包含 `acl/`、`binapi_acl_types/`、`binapi_acl/` 三个并列模块 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal`
- [ ] [T050] [US3] 创建版本追溯文档：记录每个本地化模块的原始仓库地址、commit hash、go.sum hash 和本地化日期，便于未来升级或问题排查 - `/home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/docs` 或 specs 目录

---

## 依赖关系说明

- **T024 依赖 T023**：迭代 1 必须推送完成后，用户才能开始测试
- **T026-T044 依赖 T025**：迭代 2 必须在迭代 1 测试通过后才能开始
- **T047-T050 可在 T046 后并行执行**：版本一致性验证可以在所有模块本地化完成后统一进行

**并行机会**：
- **T003 与 T001-T002 可并行**：前置检查和模块下载可同时进行
- **T026 可与 T024-T025 并行**：迭代 1 测试期间可预先验证缓存

---

## 实施策略

### MVP-first 原则

1. **迭代 1 是最小可交付单元**：完成 T001-T023，构建 v1.0.1 镜像，用户验证单个模块本地化流程
2. **迭代 2 基于迭代 1 成功**：仅在 T025 通过后启动 T026-T044
3. **版本维护是收尾工作**：T047-T050 确保长期可维护性，但不阻塞核心功能

### 回滚机制

如果任何测试任务（T024, T045）失败：
1. 停止后续任务执行
2. 回滚到上一个稳定版本镜像（v1.0.0 或 v1.0.1）
3. 执行代码回滚：`git reset --hard HEAD~1`
4. 删除失败的 git tag 和镜像
5. 分析失败原因，修复后重新执行失败的迭代

### 验收标准

所有任务完成后，必须满足：
- ✅ 项目编译通过：`go build ./...` 无错误
- ✅ 所有本地化模块包含 go.mod、README.md 和中文注释
- ✅ 项目 go.mod 包含正确的 replace 指令
- ✅ Docker 镜像成功构建并推送：v1.0.1, v1.0.2
- ✅ Git 提交和标签已推送：v1.0.1, v1.0.2
- ✅ 用户功能测试全部通过（迭代 1 和迭代 2）
- ✅ 版本追溯文档完整，可快速定位每个模块的上游版本

---

**任务总数**: 50 | **预计完成时间**: 迭代 1 (2 小时) + 用户测试 (1 小时) + 迭代 2 (2 小时) + 用户测试 (1 小时) + 版本维护 (0.5 小时) = 约 6.5 小时

**生成日期**: 2025-11-13 | **文档版本**: 1.0
