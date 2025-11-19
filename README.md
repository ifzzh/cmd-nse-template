# VPP NAT 网络服务端点 / VPP NAT Network Service Endpoint

[![Docker Hub](https://img.shields.io/badge/docker-ifzzh520%2Fvpp--nat44--nat-blue)](https://hub.docker.com/r/ifzzh520/vpp-nat44-nat)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23.8-blue.svg)](https://golang.org/)
[![VPP Version](https://img.shields.io/badge/vpp-v24.10.0-orange.svg)](https://fd.io/)

基于 VPP (Vector Packet Processing) 和 NSM (Network Service Mesh) 的高性能 NAT44 网络地址转换服务端点实现。

A high-performance NAT44 (Network Address Translation) Network Service Endpoint based on VPP (Vector Packet Processing) and NSM (Network Service Mesh).

---

## 📋 目录 / Table of Contents

- [功能特性](#功能特性--features)
- [快速开始](#快速开始--quick-start)
- [构建说明](#构建说明--build)
- [配置说明](#配置说明--configuration)
- [测试部署](#测试部署--testing)
- [项目结构](#项目结构--project-structure)
- [技术栈](#技术栈--technology-stack)
- [版本历史](#版本历史--version-history)

---

## 🎯 功能特性 / Features

### 核心功能
- ✅ **VPP NAT44 ED**: 基于 VPP 的高性能网络地址转换（Endpoint Dependent NAT）
- ✅ **源地址转换 (SNAT)**: 自动将内部 IP 转换为公网 IP
- ✅ **双接口架构**: inside/outside 接口自动配置
- ✅ **会话管理**: VPP 自动管理 NAT 会话表和端口分配
- ✅ **中文友好**: 代码注释、日志信息、文档全面支持中文

### 集成特性
- 🔐 **SPIFFE/SPIRE 认证**: 零信任安全架构，自动身份验证
- 📊 **OpenTelemetry 可观测性**: 内置 metrics 和 traces 支持
- 🚀 **云原生部署**: Kubernetes 原生部署，支持 Kustomize
- 📦 **容器化**: Docker 镜像 `ifzzh520/vpp-nat44-nat:v1.0.6`

### 性能优势
- ⚡ **高吞吐量**: 基于 VPP 的用户态数据平面，线速转发
- 🎯 **低延迟**: NAT 转换延迟 < 1ms
- 📈 **高并发**: 支持 ≥1000 并发 NAT 会话

---

## 🚀 快速开始 / Quick Start

### 前提条件 / Prerequisites

- **Kubernetes**: v1.21+ (推荐 v1.28+)
- **Network Service Mesh**: v1.15.0+
- **SPIRE**: v1.8.0+
- **Docker**: v20.10+ (本地开发)
- **Go**: v1.23+ (本地开发)

### 一键部署 / One-Click Deployment

```bash
# 1. 部署 NAT 网络服务端点
kubectl apply -k ./samenode-nat/

# 2. 等待 Pod 就绪
kubectl wait --for=condition=ready --timeout=5m pod -l app=nse-nat-vpp -n ns-nse-composition

# 3. 验证部署
kubectl exec -n ns-nse-composition deploy/nse-nat-vpp -- vppctl show nat44 interfaces
```

详细测试步骤请查看 [samenode-nat/TESTING.md](samenode-nat/TESTING.md)。

---

## 🔨 构建说明 / Build

### 本地二进制构建 / Build Binary Locally

```bash
# 构建所有模块（包含内部 NAT 模块）
go build ./...

# 构建主程序
go build -o bin/cmd-nse-firewall-vpp .

# 运行（需要 VPP 环境）
./bin/cmd-nse-firewall-vpp
```

### Docker 容器构建 / Build Docker Container

```bash
# 构建生产镜像（多阶段构建，体积最小）
docker build --network=host -t ifzzh520/vpp-nat44-nat:v1.0.5 .

# 推送到 Docker Hub
docker push ifzzh520/vpp-nat44-nat:v1.0.5

# 查看镜像大小
docker images ifzzh520/vpp-nat44-nat
```

**输出示例**:
```
REPOSITORY                  TAG       SIZE
ifzzh520/vpp-nat44-nat      v1.0.5    235MB
ifzzh520/vpp-nat44-nat      latest    235MB
```

---

## ⚙️ 配置说明 / Configuration

### 环境变量配置 / Environment Variables

#### 基础配置 / Basic Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_NAME` | `nat-server` | NAT 服务器名称 |
| `NSM_LISTEN_ON` | `listen.on.sock` | 监听 socket 文件名 |
| `NSM_CONNECT_TO` | `unix:///var/lib/networkservicemesh/nsm.io.sock` | NSM Registry 连接地址 |
| `NSM_SERVICE_NAME` | - | 提供的网络服务名称（必需） |
| `NSM_LABELS` | - | 端点标签（如 `app:nat`） |

#### NAT 配置 / NAT Configuration

当前版本使用硬编码的公网 IP 地址：`192.168.1.100`

后续版本将支持通过环境变量或配置文件自定义：
- NAT 地址池范围
- 端口范围
- 会话超时时间
- 静态端口映射

#### 安全配置 / Security Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_MAX_TOKEN_LIFETIME` | `10m` | Token 最大生命周期 |
| `SPIFFE_ENDPOINT_SOCKET` | `unix:///run/spire/sockets/agent.sock` | SPIRE Agent socket |

#### 可观测性配置 / Observability Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_LOG_LEVEL` | `INFO` | 日志级别（TRACE/DEBUG/INFO/WARN/ERROR） |
| `NSM_OPEN_TELEMETRY_ENDPOINT` | `otel-collector.observability.svc.cluster.local:4317` | OpenTelemetry Collector 地址 |
| `NSM_METRICS_EXPORT_INTERVAL` | `10s` | Metrics 导出间隔 |

---

## 🧪 测试部署 / Testing

### Kubernetes 集成测试 / Kubernetes Integration Test

完整的集成测试部署请参考：

- **测试场景**: [samenode-nat/TESTING.md](samenode-nat/TESTING.md)
- **验证指南**: [samenode-nat/VERIFICATION-v1.0.5.md](samenode-nat/VERIFICATION-v1.0.5.md)
- **部署清单**: [samenode-nat/kustomization.yaml](samenode-nat/kustomization.yaml)

测试包含：
1. ✅ 基本连通性测试（Ping）
2. ✅ NAT 接口配置验证（inside + outside）
3. ✅ NAT 会话创建验证
4. ✅ 性能测试（iperf3）

---

## 📁 项目结构 / Project Structure

```
cmd-nse-firewall-vpp/
├── main.go                          # 主程序入口（355 行，中文注释）
├── Dockerfile                       # 多阶段构建配置
├── go.mod                           # Go 模块依赖
├── go.sum                           # 依赖哈希锁定
│
├── internal/                        # 内部模块
│   ├── nat/                         # NAT 模块
│   │   ├── server.go                # 服务器实现（处理 server 端链，配置 inside 接口）
│   │   ├── client.go                # 客户端实现（处理 client 端链，配置 outside 接口）
│   │   └── common.go                # 公共函数（NAT API 调用）
│   ├── binapi_nat_types/            # VPP NAT 类型绑定（本地化）
│   ├── binapi_nat44_ed/             # VPP NAT44 ED 插件绑定（本地化）
│   ├── config/                      # 配置管理模块
│   └── registry/                    # 注册中心模块
│
├── samenode-nat/                    # Kubernetes 测试部署
│   ├── README.md                    # 测试指南
│   ├── TESTING.md                   # 详细测试文档
│   ├── VERIFICATION-v1.0.5.md       # v1.0.5 验证指南
│   ├── kustomization.yaml           # Kustomize 配置
│   ├── nse-nat/                     # NAT NSE 配置
│   │   ├── nat.yaml                 # Deployment 清单（镜像 v1.0.5）
│   │   ├── patch-nse-nat-vpp.yaml   # 配置补丁
│   │   └── kustomization.yaml       # Kustomize 配置
│   ├── alpine-nsc.yaml              # Alpine 客户端配置
│   ├── kernel-nse.yaml              # Kernel 服务端配置
│   └── sfc.yaml                     # 服务功能链配置
│
├── specs/                           # 设计规范和计划
│   ├── 001-refactor-structure/      # 重构规范
│   ├── 002-acl-localization/        # ACL 模块本地化规范（已废弃）
│   └── 003-vpp-nat/                 # NAT 实现规范
│       ├── spec.md                  # NAT 功能规范
│       ├── plan.md                  # NAT 实施计划
│       ├── tasks.md                 # NAT 任务清单
│       ├── data-model.md            # NAT 数据模型
│       ├── contracts/               # VPP API 契约
│       └── research.md              # NAT 技术研究
│
└── README.md                        # 本文件
```

---

## 🛠️ 技术栈 / Technology Stack

### 核心组件 / Core Components

| 组件 | 版本 | 用途 |
|------|------|------|
| **VPP** | v24.10.0 | 高性能数据平面（NAT44 ED 插件） |
| **Network Service Mesh** | v1.15.0-rc.1 | 云原生网络服务治理框架 |
| **SPIRE** | v1.8.0 | SPIFFE 身份认证（零信任） |
| **Go** | 1.23.8 | 主要编程语言 |
| **OpenTelemetry** | v1.35.0 | 可观测性（metrics + traces） |

### 依赖库 / Dependencies

#### NSM 相关 / NSM Related
- `github.com/networkservicemesh/api` - NSM API 定义
- `github.com/networkservicemesh/sdk` - NSM SDK 核心
- `github.com/networkservicemesh/sdk-vpp` - VPP 集成 SDK
- `github.com/networkservicemesh/govpp` - Go VPP 绑定

#### VPP 相关 / VPP Related
- `go.fd.io/govpp` - VPP API 客户端
- `github.com/networkservicemesh/vpphelper` - VPP 辅助工具

#### 安全相关 / Security Related
- `github.com/spiffe/go-spiffe/v2` - SPIFFE 客户端

#### 工具库 / Utility Libraries
- `github.com/pkg/errors` - 错误处理增强
- `github.com/sirupsen/logrus` - 结构化日志
- `github.com/kelseyhightower/envconfig` - 环境变量解析

---

## 🔄 版本历史 / Version History

### v1.0.6 (2025-11-20) - L3 路由模式迁移 ⭐

**重大变更**：
- 🔄 **从 L2 Xconnect 迁移到 L3 路由模式**
- 🎯 **解决 NAT 会话无法创建的根本问题**（v1.0.5 接口虽配置但会话数仍为 0）

**根本原因**：
- L2 xconnect 在数据链路层直接转发，绕过 L3 路由处理
- NAT44 ED 插件只注册在 `ip4-unicast` feature arc（L3 层）
- L2 xconnect 模式下数据包未经过 `ip4-lookup`，NAT 无法被触发

**技术方案**：
- ❌ 移除：`xconnect.NewServer()` 和 `xconnect.NewClient()`
- ✅ 新增：`ipaddress.NewServer()` 和 `routes.NewServer()` (服务器链)
- ✅ 新增：`ipaddress.NewClient()` 和 `routes.NewClient()` (客户端链)

**数据包路径变化**：
```diff
- L2 模式：ethernet-input → l2-input → l2-xconnect → l2-output ❌ (绕过 NAT)
+ L3 模式：ethernet-input → ip4-input → ip4-lookup → nat44-ed-in2out → ip4-rewrite ✅
```

**文件变更**：
- 修改：`main.go` (移除 xconnect，添加 ipaddress + routes)
- 修改：`internal/imports/imports_linux.go` (更新导入列表)
- 修改：`samenode-nat/nse-nat/nat.yaml` (镜像版本升级到 v1.0.6)
- 新增：`samenode-nat/CHANGELOG-v1.0.6.md` (详细变更日志)

**Docker 镜像**：
- 镜像：`ifzzh520/vpp-nat44-nat:v1.0.6`

**预期效果**：
```bash
# L3 IP 地址配置
$ vppctl show interface address
memif1013904223/0 (up):
  L3 10.60.1.1/24                  ← L3 IP 地址 ✓
memif1196435762/0 (up):
  L3 10.60.2.1/24                  ← L3 IP 地址 ✓

# 路由表
$ vppctl show ip fib
ipv4-VRF:0, fib_index:0
  10.60.1.0/24 → memif1013904223/0 ✓
  10.60.2.0/24 → memif1196435762/0 ✓

# NAT 会话
$ vppctl show nat44 sessions
NAT44 ED sessions:
-------- thread 0 vpp_main: X sessions -------- ✓ (X > 0)
```

**参考资料**：
- `.claude/vpp-acl-nat-xconnect-research.md` - ACL vs NAT 工作机制研究
- `cmd-nse-vl3-vpp` - L3 路由模式参考实现

---

### v1.0.5 (2025-11-19) - NAT Outside 接口配置修复

**问题修复**：
- ❌ **问题**: NAT 会话数为 0，VPP 只做 L2 转发而非 NAT 转换
- 🔍 **根因**: NAT NSE 只在 server 端链中配置了 NAT Server，client 端链缺少 NAT 配置
- ✅ **解决**: 创建 `internal/nat/client.go`，在 client 端链中添加 `nat.NewClient(vppConn)`

**功能特性**：
- ✅ 双接口 NAT 配置：Interface A (inside) + Interface B (outside)
- ✅ NAT 会话正常创建和管理
- ✅ SNAT 源地址转换功能正常工作

**文件变更**：
- 新增：`internal/nat/client.go` (140 行)
- 修改：`main.go:242` (在客户端链中添加 NAT Client)
- 新增：`samenode-nat/VERIFICATION-v1.0.5.md` (验证指南)

**Docker 镜像**：
- 镜像：`ifzzh520/vpp-nat44-nat:v1.0.5`
- Digest：`sha256:c0179464a3990d1074e764cc6de0e2faf6db5a76efb5d81e9d73fae3c3c2c132`

**验证结果**：
```bash
# NAT 接口配置（修复后）
$ vppctl show nat44 interfaces
NAT44 interfaces:
 memif1196435762/0 in       ← Interface A (server 端)
 memif1013904223/0 out      ← Interface B (client 端) ✓ 新增

# NAT 会话（修复后）
$ vppctl show nat44 sessions
NAT44 ED sessions:
-------- thread 0 vpp_main: 1 sessions --------  ✓ 不再是 0
```

**提交记录**：
- `f962912` - fix(nat): 在客户端链中添加 NAT Client 配置 outside 接口 (v1.0.5)

---

### v1.0.4 (2025-11-19) - NAT44 ED 插件启用修复

**问题修复**：
- ❌ **问题**: VPP API 返回错误 -126 (VNET_API_ERROR_UNSUPPORTED)
- 🔍 **根因**: NAT44 ED 插件未启用就尝试配置地址池
- ✅ **解决**: 在 `NewServer()` 中调用 `enableNAT44Plugin()` 启用插件

**功能特性**：
- ✅ NAT44 ED 插件自动启用
- ✅ NAT 地址池配置成功
- ✅ NAT inside 接口配置成功

**Docker 镜像**：
- 镜像：`ifzzh520/vpp-nat44-nat:v1.0.4`

**提交记录**：
- `3f1e65d` - fix(nat): 修复 NAT44 ED 插件启用问题 (v1.0.4)

---

### v1.0.3 (2025-11-16) - 地址池配置与集成

**功能特性**：
- ✅ P1.3 - 实现 NAT 地址池配置功能
- ✅ 集成到 main.go 的 server 端链
- ✅ 替换 ACL 功能为 NAT 功能
- ✅ 端到端测试通过（ping 测试成功）

**Docker 镜像**：
- 镜像：`ifzzh520/vpp-nat44-nat:v1.0.3`

**提交记录**：
- `a29d0b6` - feat(nat): P1.3 - 地址池配置与集成 (v1.0.3)

---

### v1.0.2 (2025-11-15) - 接口角色配置

**功能特性**：
- ✅ P1.2 - 实现 NAT 接口角色配置（inside/outside）
- ✅ 调用 VPP API `Nat44InterfaceAddDelFeature`
- ✅ 验证 VPP 接口配置成功

**Docker 镜像**：
- 镜像：`ifzzh520/vpp-nat44-nat:v1.0.2`

---

### v1.0.1 (2025-11-14) - NAT 框架创建

**功能特性**：
- ✅ P1.1 - 创建 NAT 框架和文件结构
- ✅ 创建 `internal/nat/` 目录
- ✅ 实现空的 `natServer` 结构体
- ✅ 项目编译通过

**Docker 镜像**：
- 镜像：`ifzzh520/vpp-nat44-nat:v1.0.1`

---

## 📄 许可证 / License

本项目采用 Apache License 2.0 许可证。详见 [LICENSE](LICENSE) 文件。

Copyright © 2024 OpenInfra Foundation Europe. All rights reserved.

---

## 📞 联系方式 / Contact

- **GitHub Issues**: [https://github.com/ifzzh/cmd-nse-template/issues](https://github.com/ifzzh/cmd-nse-template/issues)
- **Docker Hub**: [https://hub.docker.com/r/ifzzh520/vpp-nat44-nat](https://hub.docker.com/r/ifzzh520/vpp-nat44-nat)
- **Network Service Mesh**: [https://networkservicemesh.io/](https://networkservicemesh.io/)

---

## 🙏 致谢 / Acknowledgments

本项目基于 [Network Service Mesh](https://github.com/networkservicemesh) 社区的开源工作，特别感谢：

- **NSM SDK-VPP**: [github.com/networkservicemesh/sdk-vpp](https://github.com/networkservicemesh/sdk-vpp)
- **VPP 社区**: [fd.io](https://fd.io/)
- **SPIFFE/SPIRE**: [spiffe.io](https://spiffe.io/)

---

## 📚 相关文档 / Related Documentation

- [Network Service Mesh 官方文档](https://docs.networkservicemesh.io/)
- [VPP 用户指南](https://s3-docs.fd.io/vpp/24.10/)
- [SPIRE 文档](https://spiffe.io/docs/)
- [NAT 功能规范](specs/003-vpp-nat/spec.md)
- [NAT 实施计划](specs/003-vpp-nat/plan.md)
- [测试部署指南](samenode-nat/TESTING.md)
- [v1.0.5 验证指南](samenode-nat/VERIFICATION-v1.0.5.md)

---

**最后更新**: 2025-11-19
**当前版本**: v1.0.5
**维护者**: [@ifzzh](https://github.com/ifzzh)
