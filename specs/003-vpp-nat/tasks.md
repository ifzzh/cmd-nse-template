# Implementation Tasks: VPP NAT 网络服务端点

**Feature Branch**: `003-vpp-nat` | **Date**: 2025-01-15
**Source**: Generated from [plan.md](plan.md) and [spec.md](spec.md)

## Quick Reference

- **Total Tasks**: 110
- **User Stories**: 5（P1-P5）
- **Parallel Opportunities**: 30+ 个任务可并行执行
- **Estimated Timeline**: 26 小时（3-5 天，单人开发）

---

## Task Format Legend

```
- [ ] [TaskID] [P?] [Story?] Description with file path
      ↑       ↑      ↑        ↑
      │       │      │        └─ 具体操作 + 文件路径
      │       │      └─ 用户故事标签（Setup/Foundational 阶段无此标签）
      │       └─ [P] = 可并行执行
      └─ 任务 ID（T001, T002...）
```

**Example**:
- `- [ ] T001 Create project structure per implementation plan`
- `- [ ] T012 [P] [US1] Create natServer struct in internal/nat/server.go`

---

## Phase 1: Setup（项目初始化）

**目标**: 准备基线环境，验证依赖可用性

### Tasks

- [x] T001 创建 Git 标签 v1.0.0-acl-final（标记 ACL 最后稳定版本）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp
- [x] T002 验证 VPP v24.10.0 镜像可用性（ghcr.io/networkservicemesh/govpp/vpp:v24.10.0-4-ga9d527a67）
- [x] T003 验证 govpp v0.11.0 版本（遵循现有代码）
- [x] T004 验证 NSM SDK v0.5.1-0.20250625085623-466f486d183e 版本（遵循现有代码）
- [x] T005 创建功能分支 003-vpp-nat（已在分支上）
- [x] T006 更新 .claude/operations-log.md 文件，记录决策和操作

**Acceptance Criteria**:
- ✅ Git 标签 v1.0.0-acl-final 已创建
- ✅ 所有依赖版本验证通过
- ✅ 功能分支已切换

---

## Phase 2: Foundational（阻塞性前置任务）

**目标**: 无（本项目无阻塞性前置任务，直接进入用户故事实现）

---

## Phase 3: User Story 1 - 基础 NAT44 地址转换（P1）

**Priority**: P1
**Goal**: 创建 NAT 框架，配置接口角色和地址池，实现端到端 NAT 转换
**Version**: v1.0.1 → v1.0.3
**Independent Test**: NSC → NAT NSE → 外部服务器 ping 测试成功，VPP CLI 显示 NAT 会话

### Acceptance Scenarios

1. **Given** NSC 已连接到 NAT NSE, **When** NSC 向外部服务器发送 ICMP ping 请求, **Then** 数据包的源 IP 地址被转换为 NAT 公网 IP 地址，外部服务器能够收到请求并响应
2. **Given** NAT 转换已建立, **When** 外部服务器回复响应数据包, **Then** NAT NSE 能够将目标 IP 和端口反向转换为 NSC 的内部地址，NSC 收到正确的响应
3. **Given** 多个 NSC 同时连接, **When** 它们同时发起外部连接, **Then** 每个 NSC 的数据包使用不同的转换端口号，不发生端口冲突
4. **Given** NAT NSE 运行中, **When** 查询 VPP NAT44 会话表, **Then** 能够看到所有活跃的地址转换映射记录

### Sub-Phase: P1.1 - NAT 框架创建（v1.0.1）

**Goal**: 创建 NAT 模块基础结构，确保编译通过，无功能变更

#### Tasks

- [x] T007 [P] [US1] 创建 internal/nat/ 目录在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/nat/
- [x] T008 [US1] 创建 internal/nat/server.go，定义 natServer 结构体（参考 internal/acl/server.go）包含字段 vppConn
- [x] T009 [US1] 实现 natServer 的 Request() 方法（空实现，仅调用 next.Server(ctx).Request()）在 internal/nat/server.go
- [x] T010 [US1] 实现 natServer 的 Close() 方法（空实现，仅调用 next.Server(ctx).Close()）在 internal/nat/server.go
- [x] T011 [P] [US1] 创建 internal/nat/common.go（空文件，预留公共函数）
- [x] T012 [US1] 更新 go.mod，确保 NAT 模块导入路径正确（module github.com/ifzzh/cmd-nse-template）
- [x] T013 [US1] 编译验证（go build ./...）确认无错误
- [x] T014 [US1] Git 提交：feat(nat): P1.1 - 创建 NAT 框架 (v1.0.1)

**Verification**:
```bash
go build ./...  # 编译通过
git log -1 --oneline  # 确认 commit 存在
```

---

### Sub-Phase: P1.2 - 接口角色配置（v1.0.2）

**Goal**: 实现 VPP 接口角色配置（inside/outside），验证 VPP CLI 显示正确

#### Tasks

- [x] T015 [US1] 在 internal/nat/common.go 中实现 configureNATInterface() 函数（调用 Nat44InterfaceAddDelFeature，参考 contracts/interface-role-api.md）
- [x] T016 [US1] 在 internal/nat/common.go 中实现 disableNATInterface() 函数（清理接口配置，IsAdd=false）
- [x] T017 [US1] 在 internal/nat/server.go 的 Request() 方法中集成 configureNATInterface()（判断 inside/outside 角色，使用 metadata.IsClient）
- [x] T018 [US1] 在 internal/nat/server.go 的 Close() 方法中集成 disableNATInterface()（资源清理）
- [x] T019 [US1] 添加中文日志在 internal/nat/common.go（"配置 NAT 接口: inside/outside"）
- [x] T020 [US1] 编译验证（go build ./...）
- [x] T021 [US1] VPP CLI 验证（vppctl show nat44 interfaces，确认 inside/outside 接口配置）
- [x] T022 [US1] Git 提交：feat(nat): P1.2 - 接口角色配置 (v1.0.2)

**Verification**:
```bash
# 本地 VPP 测试（可选）
docker run --rm --privileged -it ligato/vpp-base:v24.10.0 vppctl show nat44 interfaces

# 编译验证
go build ./...

# Git 提交验证
git log -1 --oneline
```

---

### Sub-Phase: P1.3 - 地址池配置与集成（v1.0.3）

**Goal**: 配置 NAT 地址池，替换 ACL 为 NAT，实现端到端 NAT 转换

#### Tasks

- [x] T023 [US1] 在 internal/nat/common.go 中实现 configureNATAddressPool() 函数（调用 Nat44AddDelAddressRange，硬编码公网 IP: 192.168.1.100，参考 contracts/address-pool-api.md）
- [x] T024 [US1] 在 internal/nat/common.go 中实现 cleanupNATAddressPool() 函数（清理地址池，IsAdd=false）
- [x] T025 [US1] 在 internal/nat/server.go 的 Request() 方法中集成 configureNATAddressPool()（在接口配置前调用，首次请求时配置）
- [x] T026 [US1] 在 internal/nat/server.go 的 Close() 方法中集成 cleanupNATAddressPool()（资源清理，暂不实现最后连接检测）
- [x] T027 [US1] 在 main.go 中删除 acl.NewServer(vppConn, config.ACLConfig) 行
- [x] T028 [US1] 在 main.go 中添加 nat.NewServer(vppConn, []net.IP{net.ParseIP("192.168.1.100")})（硬编码公网 IP）
- [x] T029 [US1] 添加中文日志在 internal/nat/common.go（"配置 NAT 地址池成功: 192.168.1.100"）
- [x] T030 [US1] 编译验证（go build ./...）
- [x] T031 [US1] 构建 Docker 镜像（docker build -t ifzzh520/vpp-nat44-nat:v1.0.3 .）（需用户执行）
- [x] T032 [US1] K8s 部署测试（kubectl apply -f deployments/，参考 quickstart.md）（需用户在远程环境执行）
- [x] T033 [US1] 端到端验证（NSC → NAT NSE → 外部服务器 ping 测试）（需用户在远程环境执行）
- [x] T034 [US1] VPP CLI 验证（vppctl show nat44 addresses, show nat44 sessions）（需用户在远程环境执行）
- [x] T035 [US1] Git 提交：feat(nat): P1.3 - 地址池配置与集成 (v1.0.3)

**Verification**:
```bash
# 编译验证
go build ./...

# Docker 镜像验证
docker images | grep vpp-nat44-nat:v1.0.3

# K8s 部署验证
kubectl get pods -n nsm-system | grep vpp-nat

# 端到端验证（从 NSC Pod 内执行）
kubectl exec -it <nsc-pod> -- ping 8.8.8.8

# VPP CLI 验证（从 NAT NSE Pod 内执行）
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 interfaces
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 addresses
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 sessions

# Git 提交验证
git log -1 --oneline
```

---

## Phase 4: User Story 2 - NAT 配置管理（P2）

**Priority**: P2
**Goal**: 实现从配置文件加载 NAT 配置，删除 ACL 配置字段
**Version**: v1.0.4
**Independent Test**: 配置文件加载成功，配置验证失败时拒绝启动（< 5 秒）

### Acceptance Scenarios

1. **Given** 配置文件包含有效的 NAT 配置, **When** NAT NSE 启动, **Then** 配置加载成功，NAT 地址池和端口范围正确配置
2. **Given** 配置文件包含无效的公网 IP, **When** NAT NSE 启动, **Then** 配置验证失败，程序在 5 秒内退出并记录错误日志
3. **Given** 配置文件中端口范围不合法（起始 > 结束）, **When** NAT NSE 启动, **Then** 配置验证失败，程序拒绝启动
4. **Given** 配置文件为空或缺失必填字段, **When** NAT NSE 启动, **Then** 程序记录错误日志并退出

### Tasks

- [ ] T036 [P] [US2] 在 internal/config/config.go 中删除 ACLConfigPath string 字段
- [ ] T037 [P] [US2] 在 internal/config/config.go 中删除 ACLConfig []acl_types.ACLRule 字段
- [ ] T038 [P] [US2] 在 internal/config/config.go 中删除 retrieveACLRules() 函数
- [ ] T039 [US2] 在 internal/config/config.go 中添加 NATConfigPath string 字段（默认值：/etc/nat/config.yaml）
- [ ] T040 [US2] 在 internal/config/config.go 中添加 NATConfig 结构体（PublicIPs, PortRangeStart, PortRangeEnd, StaticMappings, VrfID）参考 data-model.md
- [ ] T041 [US2] 实现 loadNATConfig() 函数（从 YAML 文件加载配置，参考 research.md 配置管理设计）在 internal/config/config.go
- [ ] T042 [US2] 实现 validateNATConfig() 函数（11 条验证规则，参考 data-model.md 配置验证）在 internal/config/config.go
- [ ] T043 [US2] 在 main.go 中集成配置加载逻辑（loadNATConfig() → validateNATConfig()）
- [ ] T044 [US2] 修改 nat.NewServer() 接受 Config.NATConfig 参数（而非硬编码 IP）在 internal/nat/server.go
- [ ] T045 [US2] 创建示例配置文件 examples/nat-config.yaml（包含公网 IP、端口范围示例）
- [ ] T046 [US2] 添加中文日志在 internal/config/config.go（"加载 NAT 配置成功: X 个公网 IP，端口范围 Y-Z"）
- [ ] T047 [US2] 添加中文错误日志在 internal/config/config.go（"NAT 配置错误: [具体错误信息]"）
- [ ] T048 [US2] 编译验证（go build ./...）
- [ ] T049 [US2] 配置验证测试（使用无效配置启动，验证 < 5 秒退出）
- [ ] T050 [US2] 构建 Docker 镜像（docker build -t ifzzh520/vpp-nat44-nat:v1.0.4 .）
- [ ] T051 [US2] K8s 部署测试（使用 ConfigMap 加载配置）
- [ ] T052 [US2] Git 提交：feat(config): P2 - NAT 配置管理 (v1.0.4)

**Verification**:
```bash
# 配置验证测试（无效公网 IP）
echo "nat:
  publicIPs:
    - invalid-ip
" > /tmp/invalid-config.yaml
timeout 10s go run main.go --nat-config=/tmp/invalid-config.yaml || echo "正确退出"

# 配置验证测试（端口范围错误）
echo "nat:
  publicIPs:
    - 192.168.1.100
  portRangeStart: 20000
  portRangeEnd: 10000
" > /tmp/invalid-port.yaml
timeout 10s go run main.go --nat-config=/tmp/invalid-port.yaml || echo "正确退出"

# 有效配置测试
go build ./...
docker build -t ifzzh520/vpp-nat44-nat:v1.0.4 .
kubectl apply -f deployments/
kubectl logs <nat-nse-pod> | grep "加载 NAT 配置成功"

# Git 提交验证
git log -1 --oneline
```

---

## Phase 5: User Story 3 - 模块本地化（P3）

**Priority**: P3
**Goal**: 本地化 govpp binapi NAT 模块，避免外部依赖变更风险
**Version**: v1.1.0 → v1.1.1
**Independent Test**: 编译验证 → Docker 镜像构建 → K8s 部署测试 → NAT 功能无回归

### Acceptance Scenarios

1. **Given** nat_types 模块本地化完成, **When** 编译项目, **Then** go.mod 的 replace 指令生效，导入路径指向 internal/binapi_nat_types/
2. **Given** nat44_ed 模块本地化完成, **When** 编译项目, **Then** nat44_ed 依赖本地化的 nat_types，导入路径正确
3. **Given** 本地化模块部署到 K8s, **When** 运行 NAT 功能测试, **Then** 所有测试通过，无功能回归
4. **Given** 本地化模块包含中文注释, **When** 阅读代码, **Then** 注释清晰解释 API 用途和参数

### Sub-Phase: P3.1 - 本地化 nat_types（v1.1.0）

**Goal**: 本地化 nat_types 模块，添加中文注释

#### Tasks

- [ ] T053 [US3] 查找 govpp 缓存中的 binapi/nat_types/ 目录（通常在 $GOPATH/pkg/mod/go.fd.io/govpp@...）
- [ ] T054 [P] [US3] 复制 binapi/nat_types/ 到 internal/binapi_nat_types/ 在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_nat_types/
- [ ] T055 [US3] 在 internal/binapi_nat_types/ 中创建 go.mod（module github.com/ifzzh/cmd-nse-template/internal/binapi_nat_types）
- [ ] T056 [P] [US3] 在 internal/binapi_nat_types/nat_types.ba.go 中添加中文注释（解释 NatConfigFlags、IP4Address 等类型）
- [ ] T057 [US3] 在项目根 go.mod 中添加 replace 指令（go.fd.io/govpp/binapi/nat_types => ./internal/binapi_nat_types）
- [ ] T058 [US3] 编译验证（go build ./...，确认 replace 生效）
- [ ] T059 [US3] 构建 Docker 镜像（docker build -t ifzzh520/vpp-nat44-nat:v1.1.0 .）
- [ ] T060 [US3] K8s 部署测试（kubectl apply -f deployments/）
- [ ] T061 [US3] NAT 功能回归测试（NSC → NAT NSE → 外部服务器 ping 测试，确认无功能变化）
- [ ] T062 [US3] VPP CLI 验证（vppctl show nat44 sessions，确认会话正常）
- [ ] T063 [US3] Git 提交：feat(localize): P3.1 - 本地化 nat_types (v1.1.0)

**Verification**:
```bash
# 编译验证（确认 replace 生效）
go build ./... 2>&1 | grep -q "internal/binapi_nat_types" && echo "Replace 生效"

# Docker 镜像验证
docker build -t ifzzh520/vpp-nat44-nat:v1.1.0 .

# K8s 部署验证
kubectl apply -f deployments/
kubectl get pods -n nsm-system | grep vpp-nat

# NAT 功能回归测试
kubectl exec -it <nsc-pod> -- ping 8.8.8.8

# Git 提交验证
git log -1 --oneline
```

---

### Sub-Phase: P3.2 - 本地化 nat44_ed（v1.1.1）

**Goal**: 本地化 nat44_ed 模块，确保依赖本地化的 nat_types

#### Tasks

- [ ] T064 [US3] 查找 govpp 缓存中的 binapi/nat44_ed/ 目录
- [ ] T065 [P] [US3] 复制 binapi/nat44_ed/ 到 internal/binapi_nat44_ed/ 在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/binapi_nat44_ed/
- [ ] T066 [US3] 在 internal/binapi_nat44_ed/ 中创建 go.mod（module github.com/ifzzh/cmd-nse-template/internal/binapi_nat44_ed）
- [ ] T067 [US3] 在 internal/binapi_nat44_ed/go.mod 中添加 replace 指令（nat44_ed 依赖本地化的 nat_types）
- [ ] T068 [P] [US3] 在 internal/binapi_nat44_ed/nat44_ed.ba.go 中添加中文注释（解释 Nat44InterfaceAddDelFeature、Nat44AddDelAddressRange、Nat44AddDelStaticMapping API）
- [ ] T069 [US3] 在项目根 go.mod 中添加 replace 指令（go.fd.io/govpp/binapi/nat44_ed => ./internal/binapi_nat44_ed）
- [ ] T070 [US3] 编译验证（go build ./...，确认 replace 生效）
- [ ] T071 [US3] 构建 Docker 镜像（docker build -t ifzzh520/vpp-nat44-nat:v1.1.1 .）
- [ ] T072 [US3] K8s 部署测试（kubectl apply -f deployments/）
- [ ] T073 [US3] NAT 功能回归测试（NSC → NAT NSE → 外部服务器 ping 测试，确认无功能变化）
- [ ] T074 [US3] VPP CLI 验证（vppctl show nat44 sessions，确认会话正常）
- [ ] T075 [US3] Git 提交：feat(localize): P3.2 - 本地化 nat44_ed (v1.1.1)

**Verification**:
```bash
# 编译验证（确认 replace 生效）
go build ./... 2>&1 | grep -q "internal/binapi_nat44_ed" && echo "Replace 生效"

# 依赖验证（确认 nat44_ed 依赖本地化的 nat_types）
grep -r "github.com/ifzzh/cmd-nse-template/internal/binapi_nat_types" internal/binapi_nat44_ed/go.mod

# Docker 镜像验证
docker build -t ifzzh520/vpp-nat44-nat:v1.1.1 .

# K8s 部署验证
kubectl apply -f deployments/
kubectl get pods -n nsm-system | grep vpp-nat

# NAT 功能回归测试
kubectl exec -it <nsc-pod> -- ping 8.8.8.8

# Git 提交验证
git log -1 --oneline
```

---

## Phase 6: User Story 4 - 静态端口映射（P4）

**Priority**: P4
**Goal**: 实现静态端口映射功能（端口转发）
**Version**: v1.2.0
**Independent Test**: 外部客户端 → 公网 IP:端口 → 内部服务器 访问成功

### Acceptance Scenarios

1. **Given** 配置文件包含静态映射规则（公网 8080 → 内部 80）, **When** NAT NSE 启动, **Then** 静态映射配置成功
2. **Given** 静态映射已配置, **When** 外部客户端访问公网 IP:8080, **Then** 请求被转发到内部服务器:80，响应正常返回
3. **Given** 多个静态映射规则（TCP/UDP 混合）, **When** 配置加载, **Then** 所有映射规则正确应用，协议类型转换无误
4. **Given** 静态映射公网 IP 不在地址池中, **When** 配置验证, **Then** 验证失败，程序拒绝启动并记录错误日志

### Tasks

- [ ] T076 [P] [US4] 在 internal/nat/common.go 中实现 configureStaticMapping() 函数（调用 Nat44AddDelStaticMapping，参考 contracts/static-mapping-api.md）
- [ ] T077 [P] [US4] 在 internal/nat/common.go 中实现 cleanupStaticMapping() 函数（清理静态映射，IsAdd=false）
- [ ] T078 [P] [US4] 在 internal/nat/common.go 中实现 protocolStringToNumber() 函数（TCP → 6, UDP → 17, ICMP → 1）
- [ ] T079 [US4] 在 internal/nat/server.go 的 Request() 方法中集成 configureStaticMapping()（遍历 config.StaticMappings）
- [ ] T080 [US4] 在 internal/nat/server.go 的 Close() 方法中集成 cleanupStaticMapping()
- [ ] T081 [US4] 在 internal/config/config.go 中扩展 NATConfig 结构体（添加 StaticMappings []StaticMapping 字段）
- [ ] T082 [P] [US4] 在 internal/config/config.go 中定义 StaticMapping 结构体（ExternalIP, ExternalPort, InternalIP, InternalPort, Protocol）
- [ ] T083 [US4] 在 validateNATConfig() 中添加静态映射验证规则（协议、端口、IP 格式）在 internal/config/config.go
- [ ] T084 [US4] 更新 examples/nat-config.yaml（添加静态映射示例）
- [ ] T085 [US4] 添加中文日志在 internal/nat/common.go（"配置静态端口映射: 公网 X:Y → 内部 A:B (协议)"）
- [ ] T086 [US4] 编译验证（go build ./...）
- [ ] T087 [US4] 构建 Docker 镜像（docker build -t ifzzh520/vpp-nat44-nat:v1.2.0 .）
- [ ] T088 [US4] K8s 部署测试（kubectl apply -f deployments/，ConfigMap 包含静态映射配置）
- [ ] T089 [US4] 端到端验证（从外部客户端访问公网 IP:8080 → 内部服务器:80）
- [ ] T090 [US4] VPP CLI 验证（vppctl show nat44 static mappings）
- [ ] T091 [US4] Git 提交：feat(nat): P4 - 静态端口映射 (v1.2.0)

**Verification**:
```bash
# 配置文件示例（examples/nat-config.yaml）
cat <<EOF > examples/nat-config.yaml
nat:
  publicIPs:
    - 192.168.1.100
  portRangeStart: 10000
  portRangeEnd: 20000
  staticMappings:
    - externalIP: 192.168.1.100
      externalPort: 8080
      internalIP: 10.0.0.5
      internalPort: 80
      protocol: TCP
EOF

# 编译验证
go build ./...

# Docker 镜像验证
docker build -t ifzzh520/vpp-nat44-nat:v1.2.0 .

# K8s 部署验证
kubectl apply -f deployments/
kubectl logs <nat-nse-pod> | grep "配置静态端口映射"

# 端到端验证（需要部署内部 Web 服务器 Pod）
kubectl run test-client --image=curlimages/curl --rm -it -- curl http://192.168.1.100:8080

# VPP CLI 验证
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 static mappings

# Git 提交验证
git log -1 --oneline
```

---

## Phase 7: User Story 5 - ACL 代码清理（P5）

**Priority**: P5
**Goal**: 彻底删除所有 ACL 相关代码，完成项目转型
**Version**: v1.3.0
**Independent Test**: 代码编译通过，所有 NAT 功能测试通过，代码库无 ACL 痕迹

### Acceptance Scenarios

1. **Given** ACL 代码清理完成, **When** 编译项目, **Then** 编译通过，无 ACL 相关导入错误
2. **Given** ACL 目录已删除, **When** 搜索代码库（grep -ri "acl" .）, **Then** 无任何 ACL 痕迹（忽略大小写）
3. **Given** 清理后的代码部署到 K8s, **When** 运行所有 NAT 功能测试, **Then** 测试 100% 通过，无功能回归
4. **Given** README.md 已更新, **When** 阅读文档, **Then** 文档说明项目已从 ACL 防火墙转型为 NAT 网络服务端点

### Tasks

- [ ] T092 [P] [US5] 删除 internal/acl/ 目录（包括 server.go、common.go）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/internal/acl/
- [ ] T093 [P] [US5] 删除 main.go 中的 ACL 导入语句（import "github.com/ifzzh/cmd-nse-template/internal/acl"）
- [ ] T094 [P] [US5] 删除配置文件示例中的 ACL 规则（如果存在）
- [ ] T095 [US5] 更新 README.md（移除 ACL 相关说明，添加 NAT 功能描述，说明项目转型）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/README.md
- [ ] T096 [US5] 更新 Dockerfile（移除 ACL 相关构建步骤，如果存在）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/Dockerfile
- [ ] T097 [US5] 搜索验证（grep -ri "acl" . --exclude-dir=.git，确认无结果）
- [ ] T098 [US5] 编译验证（go build ./...）
- [ ] T099 [US5] 构建 Docker 镜像（docker build -t ifzzh520/vpp-nat44-nat:v1.3.0 .）
- [ ] T100 [US5] K8s 部署测试（kubectl apply -f deployments/）
- [ ] T101 [US5] NAT 功能完整测试（P1: 基础 SNAT、P2: 配置管理、P3: 模块本地化、P4: 静态端口映射）
- [ ] T102 [US5] Git 提交：refactor(cleanup): P5 - 彻底删除 ACL 遗留代码 (v1.3.0)

**Verification**:
```bash
# 删除验证
ls internal/acl/ 2>/dev/null && echo "ACL 目录仍存在" || echo "ACL 目录已删除"

# 搜索验证（忽略 .git 目录）
grep -ri "acl" . --exclude-dir=.git --exclude-dir=.specify && echo "发现 ACL 痕迹" || echo "无 ACL 痕迹"

# 编译验证
go build ./...

# Docker 镜像验证
docker build -t ifzzh520/vpp-nat44-nat:v1.3.0 .

# K8s 部署验证
kubectl apply -f deployments/
kubectl get pods -n nsm-system | grep vpp-nat

# NAT 功能完整测试
# P1: 基础 SNAT
kubectl exec -it <nsc-pod> -- ping 8.8.8.8

# P2: 配置管理
kubectl logs <nat-nse-pod> | grep "加载 NAT 配置成功"

# P3: 模块本地化
kubectl exec -it <nat-nse-pod> -- vppctl show nat44 interfaces

# P4: 静态端口映射
kubectl run test-client --image=curlimages/curl --rm -it -- curl http://192.168.1.100:8080

# Git 提交验证
git log -1 --oneline

# 最终验证：所有 NAT 功能测试通过率 100%
echo "NAT 功能测试完成，转型成功！"
```

---

## Phase 8: Polish & Cross-Cutting Concerns

**Goal**: 完善文档、优化日志、性能测试

### Tasks

- [ ] T103 [P] 更新 README.md（添加完整的使用指南、配置示例、故障排查）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/README.md
- [ ] T104 [P] 创建 CHANGELOG.md（记录 v1.0.0 → v1.3.0 的变更历史）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/CHANGELOG.md
- [ ] T105 [P] 性能测试（使用 iperf3 测试 NAT 吞吐量，目标 > 1 Gbps）
- [ ] T106 [P] 并发测试（使用 kubectl scale 测试 1000 个并发 NSC 连接）
- [ ] T107 [P] 日志优化（确保所有关键操作都有简体中文日志，覆盖率 > 95%）
- [ ] T108 [P] 文档最终审查（确认所有文档使用简体中文，除代码标识符外）
- [ ] T109 创建 Git 标签 v1.3.0（标记 NAT 项目完成）在 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp
- [ ] T110 推送到远程仓库（git push origin 003-vpp-nat --tags）

**Verification**:
```bash
# 性能测试
kubectl exec -it <nsc-pod> -- iperf3 -c 8.8.8.8

# 并发测试
kubectl scale deployment nsc --replicas=100

# 日志覆盖率检查
kubectl logs <nat-nse-pod> | grep -E "(配置|加载|成功|失败|错误)" | wc -l

# Git 标签验证
git tag | grep v1.3.0

# 最终验证
git log --oneline --graph --decorate --all
```

---

## Dependencies & Parallel Execution

### User Story Completion Order

```
Phase 1 (Setup)
    ↓
Phase 2 (Foundational) - 跳过（无阻塞性前置任务）
    ↓
Phase 3 (US1: P1.1 → P1.2 → P1.3) ← 必须先完成
    ↓
Phase 4 (US2: P2) ← 依赖 US1
    ↓
Phase 5 (US3: P3.1 → P3.2) ← 依赖 US2
    ↓
Phase 6 (US4: P4) ← 依赖 US3
    ↓
Phase 7 (US5: P5) ← 依赖 US4
    ↓
Phase 8 (Polish) ← 依赖所有 User Stories
```

**Critical Path**: US1 → US2 → US3 → US4 → US5（必须按顺序完成）

### Parallel Execution Opportunities

#### Within Phase 3 (US1)

**P1.1（v1.0.1）**:
- T007 (创建目录) 后，T008-T011 可部分并行
- T011 (common.go) 可与 T008-T010 并行执行

**P1.2（v1.0.2）**:
- T015 (configureNATInterface) 和 T016 (disableNATInterface) 可独立实现

**P1.3（v1.0.3）**:
- T023 (configureNATAddressPool) 和 T024 (cleanupNATAddressPool) 可独立实现

#### Within Phase 4 (US2)

- T036-T038 (删除 ACL 配置字段) 可并行执行
- T039-T040 (添加 NAT 配置字段) 可在删除完成后并行执行

#### Within Phase 5 (US3)

**P3.1（v1.1.0）**:
- T054 (复制 nat_types) 和 T056 (添加中文注释) 可并行执行

**P3.2（v1.1.1）**:
- T065 (复制 nat44_ed) 和 T068 (添加中文注释) 可并行执行

#### Within Phase 6 (US4)

- T076-T078 (实现静态映射函数) 可并行执行
- T081-T082 (定义配置结构体) 可并行执行

#### Within Phase 7 (US5)

- T092-T094 (删除 ACL 代码和导入) 可并行执行

#### Within Phase 8 (Polish)

- T103-T108 (文档和测试任务) 可完全并行执行

---

## Implementation Strategy

### MVP Scope（Minimum Viable Product）

**MVP = User Story 1（P1）完成**
- **目标**: 实现基础 SNAT 功能，端到端 NAT 转换成功
- **版本**: v1.0.3
- **交付物**:
  - NAT 框架创建（v1.0.1）
  - 接口角色配置（v1.0.2）
  - 地址池配置与集成（v1.0.3）
- **验证**: NSC → NAT NSE → 外部服务器 ping 测试成功

### Incremental Delivery

1. **Iteration 1（MVP）**: US1（P1）- 基础 SNAT
   - **工作量**: 8 小时
   - **交付**: v1.0.3
   - **演示**: NSC 可通过 NAT NSE 访问外部网络

2. **Iteration 2**: US2（P2）- NAT 配置管理
   - **工作量**: 4 小时
   - **交付**: v1.0.4
   - **演示**: 从配置文件加载 NAT 配置

3. **Iteration 3**: US3（P3）- 模块本地化
   - **工作量**: 6 小时
   - **交付**: v1.1.1
   - **演示**: 本地化模块无功能回归

4. **Iteration 4**: US4（P4）- 静态端口映射
   - **工作量**: 4 小时
   - **交付**: v1.2.0
   - **演示**: 外部客户端访问内部服务器成功

5. **Iteration 5（Final）**: US5（P5）- ACL 代码清理
   - **工作量**: 2 小时
   - **交付**: v1.3.0
   - **演示**: 项目完全转型为 NAT 服务端点

---

## Validation Checklist

### Format Validation

✅ **所有任务遵循 checklist 格式**:
- [x] 每个任务以 `- [ ]` 开头
- [x] 每个任务有唯一的 Task ID（T001-T110）
- [x] 可并行任务标记 [P]
- [x] 用户故事阶段任务标记 [US1]-[US5]
- [x] 每个任务包含具体文件路径或操作描述

### Completeness Validation

✅ **每个用户故事包含完整任务**:
- [x] US1（P1）: 29 个任务（T007-T035），涵盖 P1.1/P1.2/P1.3
- [x] US2（P2）: 17 个任务（T036-T052），涵盖配置管理
- [x] US3（P3）: 23 个任务（T053-T075），涵盖 P3.1/P3.2
- [x] US4（P4）: 16 个任务（T076-T091），涵盖静态端口映射
- [x] US5（P5）: 11 个任务（T092-T102），涵盖 ACL 清理
- [x] Setup: 6 个任务（T001-T006）
- [x] Polish: 8 个任务（T103-T110）

✅ **每个用户故事独立可测试**:
- [x] US1: NSC → NAT NSE → 外部服务器 ping 测试
- [x] US2: 配置文件加载和验证测试
- [x] US3: 模块本地化无功能回归测试
- [x] US4: 静态端口映射端到端测试
- [x] US5: NAT 功能完整测试（100% 通过率）

---

## Summary

- **Total Tasks**: 110
- **User Stories**: 5（P1-P5）
- **Parallel Opportunities**: 30+ 个任务可并行执行
- **Estimated Timeline**: 26 小时（3-5 天，单人开发）
- **MVP Scope**: User Story 1（v1.0.3）
- **Critical Path**: US1 → US2 → US3 → US4 → US5

**下一步**: 按照任务清单顺序执行，从 T001 开始，每完成一个任务打勾 ✅。
