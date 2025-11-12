# VPP ACL 防火墙网络服务端点 / VPP ACL Firewall Network Service Endpoint

[![Docker Hub](https://img.shields.io/badge/docker-ifzzh520%2Fvpp--acl--firewall-blue)](https://hub.docker.com/r/ifzzh520/vpp-acl-firewall)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23.8-blue.svg)](https://golang.org/)
[![VPP Version](https://img.shields.io/badge/vpp-v24.10.0-orange.svg)](https://fd.io/)

基于 VPP (Vector Packet Processing) 和 NSM (Network Service Mesh) 的高性能 ACL 防火墙网络服务端点实现。

A high-performance ACL firewall Network Service Endpoint based on VPP (Vector Packet Processing) and NSM (Network Service Mesh).

---

## 📋 目录 / Table of Contents

- [功能特性](#功能特性--features)
- [快速开始](#快速开始--quick-start)
- [远程拉取与部署](#远程拉取与部署--remote-deployment)
- [构建说明](#构建说明--build)
- [配置说明](#配置说明--configuration)
- [测试部署](#测试部署--testing)
- [开发调试](#开发调试--debugging)
- [项目结构](#项目结构--project-structure)
- [技术栈](#技术栈--technology-stack)
- [贡献指南](#贡献指南--contributing)

---

## 🎯 功能特性 / Features

### 核心功能
- ✅ **VPP ACL 防火墙**: 基于 VPP 的高性能访问控制列表（L3/L4 流量过滤）
- ✅ **灵活的规则配置**: 支持通过 YAML 文件或环境变量配置防火墙规则
- ✅ **双向流量控制**: 自动生成入站（ingress）和出站（egress）ACL 规则
- ✅ **热更新支持**: 通过 ConfigMap 更新规则，无需重启服务
- ✅ **中文友好**: 代码注释、日志信息、文档全面支持中文

### 集成特性
- 🔐 **SPIFFE/SPIRE 认证**: 零信任安全架构，自动身份验证
- 📊 **OpenTelemetry 可观测性**: 内置 metrics 和 traces 支持
- 🚀 **云原生部署**: Kubernetes 原生部署，支持 Helm 和 Kustomize
- 🔧 **OPA 策略引擎**: 灵活的访问控制策略
- 📦 **容器化**: Docker 镜像 `ifzzh520/vpp-acl-firewall:v1.0.0`

### 性能优势
- ⚡ **高吞吐量**: 基于 VPP 的用户态数据平面，线速转发
- 🎯 **低延迟**: 微秒级数据包处理延迟
- 📈 **高扩展性**: 支持大规模 ACL 规则集

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
# 1. 部署防火墙网络服务端点
kubectl apply -k ./samenode-firewall/

# 2. 等待 Pod 就绪
kubectl wait --for=condition=ready --timeout=5m pod -l app=nse-firewall-vpp -n ns-nse-composition

# 3. 验证部署
kubectl exec -n ns-nse-composition deploy/nse-firewall-vpp -- vppctl show acl-plugin acl
```

详细测试步骤请查看 [samenode-firewall/README.md](samenode-firewall/README.md)。

---

## 🌐 远程拉取与部署 / Remote Deployment

### 从 GitHub 拉取最新分支 / Pull Latest Branch from GitHub

#### 方法 1: 克隆仓库（推荐新用户）/ Clone Repository (Recommended for New Users)

```bash
# 克隆主仓库
git clone git@github.com:ifzzh/cmd-nse-template.git
cd cmd-nse-template

# 切换到开发分支（包含最新的重构和优化）
git checkout 001-refactor-structure

# 查看分支状态
git status
```

#### 方法 2: 拉取远程分支（已有本地仓库）/ Pull Remote Branch (Existing Local Repository)

```bash
# 进入项目目录
cd /path/to/cmd-nse-template

# 获取远程最新分支信息
git fetch origin

# 切换到远程分支
git checkout -b 001-refactor-structure origin/001-refactor-structure

# 或者，如果本地已有该分支，拉取最新更新
git checkout 001-refactor-structure
git pull origin 001-refactor-structure
```

#### 方法 3: 使用 HTTPS（无需 SSH 密钥）/ Using HTTPS (No SSH Key Required)

```bash
# 克隆仓库（HTTPS）
git clone https://github.com/ifzzh/cmd-nse-template.git
cd cmd-nse-template

# 切换到开发分支
git checkout 001-refactor-structure
```

### 验证拉取成功 / Verify Pull Success

```bash
# 查看当前分支
git branch

# 查看最新提交
git log --oneline -5

# 验证文件结构
ls -la internal/acl/
```

应该看到以下输出：
```
* 001-refactor-structure
  main

internal/acl/
├── common.go   (185 行，包含中文注释)
└── server.go   (168 行，包含中文注释)
```

### 远程环境快速部署 / Quick Deployment in Remote Environment

#### 使用 Docker Hub 镜像部署（最快）/ Deploy with Docker Hub Image (Fastest)

```bash
# 1. 进入测试目录
cd cmd-nse-template/samenode-firewall/

# 2. 确认镜像配置（已自动配置为 ifzzh520/vpp-acl-firewall:v1.0.0）
grep "image:" nse-firewall/firewall.yaml

# 3. 部署到 Kubernetes
kubectl apply -k .

# 4. 监控部署状态
watch kubectl get pod -n ns-nse-composition -o wide
```

#### 从源码构建并部署 / Build from Source and Deploy

```bash
# 1. 构建 Docker 镜像
docker build -t ifzzh520/vpp-acl-firewall:v1.0.0 .

# 2. 推送到私有仓库（可选）
docker tag ifzzh520/vpp-acl-firewall:v1.0.0 your-registry/vpp-acl-firewall:v1.0.0
docker push your-registry/vpp-acl-firewall:v1.0.0

# 3. 更新 Kubernetes 配置
sed -i 's|ifzzh520/vpp-acl-firewall:v1.0.0|your-registry/vpp-acl-firewall:v1.0.0|g' \
  samenode-firewall/nse-firewall/firewall.yaml

# 4. 部署
kubectl apply -k ./samenode-firewall/
```

### 常见问题排查 / Troubleshooting

#### 问题 1: 拉取失败 "Permission denied (publickey)"

**原因**: SSH 密钥未配置

**解决方案**:
```bash
# 方法 1: 使用 HTTPS 代替 SSH
git clone https://github.com/ifzzh/cmd-nse-template.git

# 方法 2: 配置 SSH 密钥
ssh-keygen -t ed25519 -C "your_email@example.com"
cat ~/.ssh/id_ed25519.pub  # 复制公钥到 GitHub Settings
```

#### 问题 2: 远程分支不存在 "remote branch not found"

**原因**: 本地 Git 信息过期

**解决方案**:
```bash
# 刷新远程分支列表
git fetch origin --prune

# 查看所有远程分支
git branch -r

# 重新拉取
git checkout -b 001-refactor-structure origin/001-refactor-structure
```

#### 问题 3: Kubernetes 镜像拉取失败 "ImagePullBackOff"

**原因**: 无法访问 Docker Hub 或镜像不存在

**解决方案**:
```bash
# 方法 1: 验证镜像存在
docker pull ifzzh520/vpp-acl-firewall:v1.0.0

# 方法 2: 配置镜像拉取策略
kubectl edit deployment nse-firewall-vpp -n ns-nse-composition
# 修改 imagePullPolicy: IfNotPresent 为 Always

# 方法 3: 配置镜像仓库代理
kubectl create secret docker-registry regcred \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=<your-username> \
  --docker-password=<your-password> \
  -n ns-nse-composition
```

---

## 🔨 构建说明 / Build

### 本地二进制构建 / Build Binary Locally

```bash
# 构建所有模块（包含内部 ACL 模块）
go build ./...

# 构建主程序
go build -o bin/cmd-nse-firewall-vpp .

# 运行（需要 VPP 环境）
./bin/cmd-nse-firewall-vpp
```

### Docker 容器构建 / Build Docker Container

```bash
# 构建生产镜像（多阶段构建，体积最小）
docker build --target runtime -t ifzzh520/vpp-acl-firewall:v1.0.0 .

# 构建测试镜像
docker build --target test -t ifzzh520/vpp-acl-firewall:test .

# 构建调试镜像（包含 dlv 调试器）
docker build --target debug -t ifzzh520/vpp-acl-firewall:debug .

# 查看镜像大小
docker images ifzzh520/vpp-acl-firewall
```

**输出示例**:
```
REPOSITORY                      TAG       SIZE
ifzzh520/vpp-acl-firewall       v1.0.0    235MB
ifzzh520/vpp-acl-firewall       test      520MB
ifzzh520/vpp-acl-firewall       debug     580MB
```

### 构建架构说明 / Build Architecture

项目采用 **多阶段 Docker 构建**，包含以下 target：

| Target | 用途 | 包含内容 | 镜像大小 |
|--------|------|----------|---------|
| `go` | Go 编译环境 | Go 1.23.1 + VPP + SPIRE | ~450MB |
| `build` | 编译二进制 | 源码 + 依赖 | ~520MB |
| `test` | 单元测试 | 测试框架 + 测试用例 | ~520MB |
| `debug` | 调试环境 | dlv 调试器 + 测试 | ~580MB |
| `runtime` | **生产运行** | 仅二进制 + VPP 运行时 | **235MB** |

---

## ⚙️ 配置说明 / Configuration

### 环境变量配置 / Environment Variables

#### 基础配置 / Basic Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_NAME` | `firewall-server` | 防火墙服务器名称 |
| `NSM_LISTEN_ON` | `listen.on.sock` | 监听 socket 文件名 |
| `NSM_CONNECT_TO` | `unix:///var/lib/networkservicemesh/nsm.io.sock` | NSM Registry 连接地址 |
| `NSM_SERVICE_NAME` | - | 提供的网络服务名称（必需） |
| `NSM_LABELS` | - | 端点标签（如 `app:firewall`） |

#### ACL 防火墙配置 / ACL Firewall Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_ACL_CONFIG_PATH` | `/etc/firewall/config.yaml` | ACL 配置文件路径 |
| `NSM_ACL_CONFIG` | - | 直接配置 ACL 规则（YAML 格式） |

#### 安全配置 / Security Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_MAX_TOKEN_LIFETIME` | `10m` | Token 最大生命周期 |
| `NSM_REGISTRY_CLIENT_POLICIES` | `etc/nsm/opa/...` | OPA 策略文件路径 |
| `SPIFFE_ENDPOINT_SOCKET` | `unix:///run/spire/sockets/agent.sock` | SPIRE Agent socket |

#### 可观测性配置 / Observability Configuration

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NSM_LOG_LEVEL` | `INFO` | 日志级别（TRACE/DEBUG/INFO/WARN/ERROR） |
| `NSM_OPEN_TELEMETRY_ENDPOINT` | `otel-collector.observability.svc.cluster.local:4317` | OpenTelemetry Collector 地址 |
| `NSM_METRICS_EXPORT_INTERVAL` | `10s` | Metrics 导出间隔 |
| `NSM_PPROF_ENABLED` | `false` | 是否启用 pprof 性能分析 |
| `NSM_PPROF_LISTEN_ON` | `localhost:6060` | pprof 监听地址 |

### ACL 规则配置示例 / ACL Rule Configuration Examples

#### 配置文件方式 / Configuration File Method

创建 `config.yaml` 文件：

```yaml
# 允许 iperf3 性能测试端口（TCP）
allow tcp5201:
    proto: 6                        # TCP 协议
    srcportoricmptypelast: 65535    # 源端口: 任意
    dstportoricmpcodefirst: 5201    # 目标端口: 5201
    dstportoricmpcodelast: 5201
    ispermit: 1                     # 允许

# 允许 ICMP ping 测试
allow icmp:
    ispermit: 1                     # 允许
    proto: 1                        # ICMP 协议
    srcportoricmptypelast: 65535
    dstportoricmpcodelast: 65535

# 禁止 HTTP 标准端口
forbid tcp80:
    proto: 6                        # TCP 协议
    srcportoricmptypelast: 65535    # 源端口: 任意
    dstportoricmpcodefirst: 80      # 目标端口: 80
    dstportoricmpcodelast: 80
    ispermit: 0                     # 拒绝
```

挂载到容器：
```yaml
volumeMounts:
  - name: acl-config
    mountPath: /etc/firewall/config.yaml
    subPath: config.yaml
```

#### 环境变量方式 / Environment Variable Method

在 Kubernetes Deployment 中配置：

```yaml
env:
  - name: NSM_ACL_CONFIG
    value: |
      allow tcp5201:
          proto: 6
          dstportoricmpcodefirst: 5201
          dstportoricmpcodelast: 5201
          ispermit: 1
```

#### ConfigMap 方式 / ConfigMap Method

使用 Kubernetes ConfigMap（推荐）：

```bash
# 1. 创建 ConfigMap
kubectl create configmap firewall-config-file \
  --from-file=config.yaml=./config.yaml \
  -n ns-nse-composition

# 2. 在 Deployment 中引用
# 参见 samenode-firewall/nse-firewall/config-patch.yaml
```

### 规则字段说明 / Rule Field Description

| 字段名 | 类型 | 说明 | 示例值 |
|--------|------|------|--------|
| `proto` | uint8 | 协议号 | 6 (TCP), 17 (UDP), 1 (ICMP) |
| `srcprefix` | IP Prefix | 源 IP 地址前缀 | `192.168.1.0/24` |
| `dstprefix` | IP Prefix | 目标 IP 地址前缀 | `10.0.0.0/8` |
| `srcportoricmptypefirst` | uint16 | 源端口范围起始 | `1024` |
| `srcportoricmptypelast` | uint16 | 源端口范围结束 | `65535` |
| `dstportoricmpcodefirst` | uint16 | 目标端口范围起始 | `80` |
| `dstportoricmpcodelast` | uint16 | 目标端口范围结束 | `80` |
| `ispermit` | uint8 | 动作：1=允许, 0=拒绝 | `1` |

---

## 🧪 测试部署 / Testing

### 运行测试容器 / Run Test Container

```bash
# 运行所有单元测试
docker run --privileged --rm $(docker build -q --target test .)

# 运行特定测试
docker run --privileged --rm $(docker build -q --target test .) \
  go test -v ./internal/acl/...
```

### Kubernetes 集成测试 / Kubernetes Integration Test

完整的集成测试部署请参考：

- **测试场景**: [samenode-firewall/README.md](samenode-firewall/README.md)
- **配置示例**: [samenode-firewall/config-file.yaml](samenode-firewall/config-file.yaml)
- **部署清单**: [samenode-firewall/kustomization.yaml](samenode-firewall/kustomization.yaml)

测试包含：
1. ✅ 基本连通性测试（Ping）
2. ✅ 防火墙规则验证（端口过滤）
3. ✅ 性能测试（iperf3）
4. ✅ VPP ACL 规则检查

---

## 🐛 开发调试 / Debugging

### 调试测试代码 / Debugging Tests

```bash
# 启动调试容器（dlv 监听端口 40000）
docker run --privileged --rm -p 40000:40000 $(docker build -q --target debug .)

# 使用 IDE 连接到 localhost:40000
# 例如 VS Code launch.json:
{
  "name": "Attach to Docker dlv",
  "type": "go",
  "request": "attach",
  "mode": "remote",
  "remotePath": "/build",
  "port": 40000,
  "host": "localhost"
}
```

### 调试主程序 / Debugging Main Program

```bash
# 启动调试容器（dlv 监听端口 50000）
docker run --privileged \
  -e DLV_LISTEN_FORWARDER=:50000 \
  -p 50000:50000 \
  --rm $(docker build -q --target test .)

# IDE 连接配置同上，端口改为 50000
```

### 同时调试测试和主程序 / Debug Both Tests and Main Program

```bash
docker run --privileged \
  -e DLV_LISTEN_FORWARDER=:50000 \
  -p 40000:40000 \
  -p 50000:50000 \
  --rm $(docker build -q --target debug .)
```

**注意**:
- 端口 40000 用于调试测试代码
- 端口 50000 用于调试主程序
- 测试会启动主程序，因此需要先连接 40000，运行测试到启动主程序后，才能连接 50000

### 本地调试（需要 VPP 环境）/ Local Debugging (Requires VPP)

```bash
# 1. 安装 VPP
sudo apt install vpp vpp-plugin-core vpp-plugin-dpdk

# 2. 启动 VPP
sudo systemctl start vpp

# 3. 使用 dlv 调试
dlv debug . -- \
  --name=firewall-test \
  --log-level=TRACE

# 4. 在 dlv 中设置断点
(dlv) break internal/acl/server.go:100
(dlv) continue
```

---

## 📁 项目结构 / Project Structure

```
cmd-nse-firewall-vpp/
├── main.go                          # 主程序入口（373 行，中文注释）
├── Dockerfile                       # 多阶段构建配置
├── go.mod                           # Go 模块依赖
├── go.sum                           # 依赖哈希锁定
│
├── internal/                        # 内部模块（本地化）
│   ├── acl/                         # ACL 防火墙模块
│   │   ├── common.go                # 公共函数（185 行，+69 注释）
│   │   └── server.go                # 服务器实现（168 行，+75 注释）
│   ├── config/                      # 配置管理模块
│   │   └── config.go                # 配置加载（104 行）
│   ├── registry/                    # 注册中心模块
│   │   └── registry.go              # 服务注册（66 行）
│   └── imports/                     # 依赖导入
│
├── samenode-firewall/               # Kubernetes 测试部署
│   ├── README.md                    # 测试指南（180 行，中英双语）
│   ├── config-file.yaml             # ACL 规则配置（54 行，中文注释）
│   ├── kustomization.yaml           # Kustomize 配置
│   ├── nse-firewall/                # 防火墙 NSE 配置
│   │   ├── firewall.yaml            # Deployment 清单
│   │   ├── patch-nse-firewall-vpp.yaml  # 配置补丁
│   │   └── kustomization.yaml       # Kustomize 配置
│   ├── client.yaml                  # NSC 客户端配置
│   ├── server.yaml                  # NSE 服务端配置
│   └── ...                          # 其他测试资源
│
├── specs/                           # 设计规范和计划
│   └── 001-refactor-structure/      # 重构规范
│       ├── spec.md                  # 功能规范
│       ├── plan.md                  # 实施计划
│       ├── tasks.md                 # 任务清单
│       ├── REFACTOR_SUMMARY.md      # 重构总结
│       └── NF-IMPLEMENTATIONS.md    # 网络功能实现分析（1241 行）
│
└── README.md                        # 本文件
```

### 代码统计 / Code Statistics

| 模块 | 文件数 | 代码行数 | 注释行数 | 注释率 |
|------|--------|---------|---------|--------|
| main.go | 1 | 260 | 113 | 30.3% |
| internal/acl/ | 2 | 246 | 144 | 36.9% |
| internal/config/ | 1 | 104 | - | - |
| internal/registry/ | 1 | 66 | - | - |
| **总计** | **5** | **676** | **257** | **27.5%** |

---

## 🛠️ 技术栈 / Technology Stack

### 核心组件 / Core Components

| 组件 | 版本 | 用途 |
|------|------|------|
| **VPP** | v24.10.0 | 高性能数据平面（用户态转发） |
| **Network Service Mesh** | v1.15.0-rc.1 | 云原生网络服务治理框架 |
| **SPIRE** | v1.8.0 | SPIFFE 身份认证（零信任） |
| **Go** | 1.23.8 | 主要编程语言 |
| **OpenTelemetry** | v1.35.0 | 可观测性（metrics + traces） |
| **OPA** | v1.4.0 | 策略引擎（访问控制） |

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
- `github.com/golang-jwt/jwt/v4` - JWT 令牌处理

#### 工具库 / Utility Libraries
- `github.com/pkg/errors` - 错误处理增强
- `github.com/sirupsen/logrus` - 结构化日志
- `github.com/kelseyhightower/envconfig` - 环境变量解析
- `gopkg.in/yaml.v3` - YAML 解析

---

## 🤝 贡献指南 / Contributing

### 分支策略 / Branch Strategy

| 分支名 | 用途 | 合并目标 |
|--------|------|---------|
| `main` | 主分支（稳定版本） | - |
| `001-refactor-structure` | 重构分支（开发中） | `main` |
| `feature/*` | 功能开发分支 | `001-refactor-structure` |
| `bugfix/*` | 缺陷修复分支 | `main` 或对应开发分支 |

### 提交规范 / Commit Convention

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型（type）**:
- `feat`: 新功能
- `fix`: 缺陷修复
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**:
```bash
git commit -m "feat(acl): 添加 IPv6 ACL 规则支持

- 扩展 ACL 规则结构体支持 IPv6 地址
- 更新 VPP API 调用逻辑
- 添加 IPv6 规则测试用例

Closes #123"
```

### 开发流程 / Development Workflow

1. **Fork 仓库** / Fork the repository
   ```bash
   # 在 GitHub 上点击 Fork 按钮
   ```

2. **克隆 Fork** / Clone your fork
   ```bash
   git clone git@github.com:your-username/cmd-nse-template.git
   cd cmd-nse-template
   ```

3. **添加上游仓库** / Add upstream remote
   ```bash
   git remote add upstream git@github.com:ifzzh/cmd-nse-template.git
   ```

4. **创建功能分支** / Create feature branch
   ```bash
   git checkout -b feature/your-feature-name
   ```

5. **开发和测试** / Develop and test
   ```bash
   # 编写代码
   vim internal/acl/server.go

   # 运行测试
   go test ./...

   # 构建镜像
   docker build .
   ```

6. **提交更改** / Commit changes
   ```bash
   git add .
   git commit -m "feat(acl): your feature description"
   ```

7. **同步上游** / Sync with upstream
   ```bash
   git fetch upstream
   git rebase upstream/001-refactor-structure
   ```

8. **推送到 Fork** / Push to your fork
   ```bash
   git push origin feature/your-feature-name
   ```

9. **创建 Pull Request** / Create Pull Request
   - 在 GitHub 上创建 PR
   - 目标分支: `001-refactor-structure`
   - 填写 PR 模板，说明更改内容

### 代码审查清单 / Code Review Checklist

- [ ] 代码遵循项目规范（[CLAUDE.md](.claude/CLAUDE.md)）
- [ ] 添加了中文注释（关键函数和复杂逻辑）
- [ ] 通过所有单元测试（`go test ./...`）
- [ ] 通过 Docker 构建（`docker build .`）
- [ ] 更新了相关文档（README、specs）
- [ ] 日志级别保持不变（Debug 不能改为 Warn）
- [ ] 没有引入新的安全风险

---

## 📄 许可证 / License

本项目采用 Apache License 2.0 许可证。详见 [LICENSE](LICENSE) 文件。

Copyright © 2024 OpenInfra Foundation Europe. All rights reserved.

---

## 📞 联系方式 / Contact

- **GitHub Issues**: [https://github.com/ifzzh/cmd-nse-template/issues](https://github.com/ifzzh/cmd-nse-template/issues)
- **Docker Hub**: [https://hub.docker.com/r/ifzzh520/vpp-acl-firewall](https://hub.docker.com/r/ifzzh520/vpp-acl-firewall)
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
- [项目重构总结](specs/001-refactor-structure/REFACTOR_SUMMARY.md)
- [NF 实现分析](specs/001-refactor-structure/NF-IMPLEMENTATIONS.md)
- [测试部署指南](samenode-firewall/README.md)

---

**最后更新**: 2025-01-12
**版本**: v1.0.0
**维护者**: [@ifzzh](https://github.com/ifzzh)
