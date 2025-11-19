# VPP NAT 网络服务端点 - 快速开始指南

**项目**: VPP NAT 网络服务端点
**版本**: 1.0
**日期**: 2025-01-15
**预计完成时间**: 本地开发 < 10 分钟，Kubernetes 部署 < 10 分钟

---

## 目录

1. [环境要求](#1-环境要求)
2. [本地开发快速开始](#2-本地开发快速开始)
3. [Kubernetes 部署快速开始](#3-kubernetes-部署快速开始)
4. [功能验证清单](#4-功能验证清单)
5. [故障排查](#5-故障排查)
6. [版本管理和回退](#6-版本管理和回退)

---

## 1. 环境要求

### 1.1 硬件要求

| 资源 | 最低配置 | 推荐配置 |
|------|---------|---------|
| CPU | 2 核 | 4 核 |
| 内存 | 4 GB | 8 GB |
| 磁盘 | 20 GB | 50 GB |
| 网络 | 单网卡 | 双网卡 |

### 1.2 软件依赖

#### 必需软件

| 软件 | 版本 | 用途 | 安装命令 |
|------|------|------|---------|
| **Go** | 1.23.8+ | 编译源代码 | `wget https://go.dev/dl/go1.23.8.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.23.8.linux-amd64.tar.gz` |
| **Docker** | 20.10+ | 构建和运行容器镜像 | `curl -fsSL https://get.docker.com \| sh` |
| **kubectl** | 1.28+ | Kubernetes 集群管理 | `curl -LO https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl && chmod +x kubectl && mv kubectl /usr/local/bin/` |
| **Git** | 2.x | 版本控制 | `apt-get install git` (Ubuntu) 或 `yum install git` (CentOS) |

#### 验证安装

```bash
# 验证 Go 环境
go version  # 预期输出: go version go1.23.8 linux/amd64

# 验证 Docker
docker --version  # 预期输出: Docker version 20.10.x

# 验证 kubectl
kubectl version --client  # 预期输出: Client Version: v1.28.0
```

### 1.3 Kubernetes 集群

#### 方式1：kind（本地开发推荐）

```bash
# 安装 kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# 创建集群
kind create cluster --name nsm-cluster

# 验证集群状态
kubectl cluster-info
kubectl get nodes
```

#### 方式2：k3s（轻量级生产环境）

```bash
# 安装 k3s
curl -sfL https://get.k3s.io | sh -

# 配置 kubectl
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

# 验证集群状态
kubectl get nodes
```

### 1.4 SPIRE 部署

VPP NAT NSE 依赖 SPIRE 进行身份认证和 mTLS 通信。

```bash
# 克隆 NSM 部署仓库
git clone https://github.com/networkservicemesh/deployments-k8s.git
cd deployments-k8s

# 部署 SPIRE
kubectl apply -k examples/spire/single_cluster

# 等待 SPIRE 就绪
kubectl wait --for=condition=ready --timeout=5m pod -n spire -l app=spire-server
kubectl wait --for=condition=ready --timeout=5m pod -n spire -l app=spire-agent

# 验证 SPIRE 状态
kubectl get pods -n spire
```

**预期输出**:
```
NAME                            READY   STATUS    RESTARTS   AGE
spire-agent-xxxxx               1/1     Running   0          2m
spire-server-0                  2/2     Running   0          2m
```

---

## 2. 本地开发快速开始

### 2.1 克隆代码仓库

```bash
# 克隆项目代码
git clone https://github.com/ifzzh/cmd-nse-firewall-vpp.git
cd cmd-nse-firewall-vpp

# 切换到 NAT 开发分支
git checkout 003-vpp-nat

# 查看当前版本
git log -1 --oneline
```

### 2.2 构建 Docker 镜像

```bash
# 构建 runtime 镜像
docker build --target runtime -t ifzzh520/vpp-nat44-nat:v1.0.3 .

# 验证镜像构建成功
docker images | grep vpp-nat44-nat
```

**预期输出**:
```
ifzzh520/vpp-nat44-nat   v1.0.3    xxxxxxxxx   1 minute ago   450MB
```

### 2.3 本地 VPP 测试

如果需要在本地调试 VPP 配置（可选步骤）：

```bash
# 运行 VPP 容器
docker run -it --privileged \
  --name vpp-nat-test \
  ifzzh520/vpp-nat44-nat:v1.0.3 \
  /bin/bash

# 在容器内启动 VPP
vpp unix {cli-listen /run/vpp/cli.sock} api-segment { prefix vpp }

# 在另一个终端连接到 VPP CLI
docker exec -it vpp-nat-test vppctl

# 清理测试容器
docker rm -f vpp-nat-test
```

### 2.4 VPP CLI 验证

在 VPP CLI 中验证 NAT 插件可用性：

```bash
# 检查 NAT 插件是否加载
vppctl show plugins | grep nat44

# 查看 NAT 接口配置（初始为空）
vppctl show nat44 interfaces

# 查看 NAT 地址池（初始为空）
vppctl show nat44 addresses

# 查看 NAT 会话（初始为空）
vppctl show nat44 sessions
```

**预期输出**:
```bash
# show plugins | grep nat44
nat44-ed_plugin.so               NAT44 endpoint-dependent

# show nat44 interfaces
NAT44 interfaces:
  (空)

# show nat44 addresses
NAT44 pool addresses:
  (空)

# show nat44 sessions
NAT44 ED sessions:
  (空)
```

---

## 3. Kubernetes 部署快速开始

### 3.1 创建 Namespace

```bash
# 创建 NSM 命名空间
kubectl create namespace nsm-system

# 验证命名空间
kubectl get namespace nsm-system
```

### 3.2 部署 SPIRE

如果尚未部署 SPIRE（参考 [1.4 SPIRE 部署](#14-spire-部署)）：

```bash
# 克隆 NSM 部署仓库
git clone https://github.com/networkservicemesh/deployments-k8s.git
cd deployments-k8s

# 部署 SPIRE
kubectl apply -k examples/spire/single_cluster

# 等待就绪
kubectl wait --for=condition=ready --timeout=5m pod -n spire -l app=spire-server
```

### 3.3 部署 NAT NSE

#### 步骤1：创建 NAT 配置 ConfigMap

```bash
# 创建 NAT 配置文件
cat <<EOF > /tmp/nat-config.yaml
public_ips:
  - 192.168.1.100
port_range:
  min: 1024
  max: 65535
vrf_id: 0
EOF

# 创建 ConfigMap
kubectl create configmap nat-config-file \
  --from-file=config.yaml=/tmp/nat-config.yaml \
  -n nsm-system

# 验证 ConfigMap
kubectl get configmap nat-config-file -n nsm-system -o yaml
```

#### 步骤2：部署 NAT NSE

```bash
# 返回项目目录
cd /path/to/cmd-nse-firewall-vpp

# 部署 NAT NSE
kubectl apply -f deployments/nat/nse-nat.yaml

# 等待 NAT NSE 就绪
kubectl wait --for=condition=ready --timeout=5m pod -n nsm-system -l app=nse-nat-vpp

# 验证部署状态
kubectl get pods -n nsm-system -l app=nse-nat-vpp
```

**预期输出**:
```
NAME                          READY   STATUS    RESTARTS   AGE
nse-nat-vpp-xxxxxxxxx-xxxxx   1/1     Running   0          1m
```

### 3.4 验证 NSM 注册

```bash
# 查看 NSE 注册状态
kubectl logs -n nsm-system -l app=nse-nat-vpp | grep "NSE registered"

# 查看 NSM Registry（如果部署了 NSM Registry）
kubectl get networkservices -n nsm-system
```

**预期日志输出**:
```
INFO[0005] NSE registered successfully    service="nat-service"
```

### 3.5 部署测试 NSC

#### 步骤1：部署测试服务器

```bash
# 部署外部测试服务器（用于接收 NAT 转换后的流量）
kubectl apply -f deployments/test/test-server.yaml

# 验证测试服务器就绪
kubectl wait --for=condition=ready --timeout=3m pod -n nsm-system -l app=test-server
```

#### 步骤2：部署 NSC

```bash
# 部署 NSC
kubectl apply -f deployments/nat/client.yaml

# 等待 NSC 就绪
kubectl wait --for=condition=ready --timeout=3m pod -n nsm-system -l app=nsc-nat

# 验证 NSC 状态
kubectl get pods -n nsm-system -l app=nsc-nat
```

**预期输出**:
```
NAME                       READY   STATUS    RESTARTS   AGE
nsc-nat-xxxxxxxxx-xxxxx    1/1     Running   0          30s
```

### 3.6 端到端验证

#### 验证1：NSC 到 NAT NSE 连接

```bash
# 查看 NSC 日志，确认连接建立
kubectl logs -n nsm-system -l app=nsc-nat | grep "Connection established"
```

**预期输出**:
```
INFO[0010] Connection established    connection_id="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

#### 验证2：NAT 地址转换功能

```bash
# 从 NSC 发起 ping 测试
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nsc-nat -o name | head -n1) -- ping -c 3 <test-server-ip>
```

**预期输出**:
```
PING <test-server-ip> (<test-server-ip>) 56(84) bytes of data.
64 bytes from <test-server-ip>: icmp_seq=1 ttl=64 time=0.5 ms
64 bytes from <test-server-ip>: icmp_seq=2 ttl=64 time=0.3 ms
64 bytes from <test-server-ip>: icmp_seq=3 ttl=64 time=0.4 ms

--- <test-server-ip> ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2050ms
rtt min/avg/max/mdev = 0.3/0.4/0.5/0.1 ms
```

#### 验证3：VPP NAT 会话

```bash
# 进入 NAT NSE 容器
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- bash

# 查看 NAT 接口配置
vppctl show nat44 interfaces
```

**预期输出**:
```
NAT44 interfaces:
  memif0/0 in   (inside 接口)
  memif0/1 out  (outside 接口)
```

```bash
# 查看 NAT 地址池
vppctl show nat44 addresses
```

**预期输出**:
```
NAT44 pool addresses:
  192.168.1.100
    tenant VRF independent
    0 busy udp ports
    0 busy tcp ports
    1 busy icmp ports
```

```bash
# 查看 NAT 会话
vppctl show nat44 sessions
```

**预期输出**:
```
NAT44 ED sessions:
  i2o 172.16.1.1:12345 -> 192.168.1.100:12345 [protocol ICMP]
      external: <test-server-ip>:12345
      state: ESTABLISHED
      timeout: 60s
```

#### 验证4：测试服务器接收到的源 IP

```bash
# 在测试服务器抓包验证源 IP 已转换
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=test-server -o name | head -n1) -- tcpdump -i eth0 icmp -n
```

**预期输出**（源 IP 应为 192.168.1.100，而非 NSC 的内部 IP）:
```
tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on eth0, link-type EN10MB (Ethernet), capture size 262144 bytes
10:30:45.123456 IP 192.168.1.100 > <test-server-ip>: ICMP echo request, id 1, seq 1, length 64
10:30:45.123789 IP <test-server-ip> > 192.168.1.100: ICMP echo reply, id 1, seq 1, length 64
```

---

## 4. 功能验证清单

### 4.1 P1 - 基础 SNAT 验证

#### ✅ P1.1 - NAT 框架创建（v1.0.1）

```bash
# 验证项目编译通过
cd /path/to/cmd-nse-firewall-vpp
go build .

# 预期：编译成功，无错误
```

#### ✅ P1.2 - 接口角色配置（v1.0.2）

```bash
# 部署 v1.0.2 镜像
kubectl set image deployment/nse-nat-vpp -n nsm-system \
  nse-nat-vpp=ifzzh520/vpp-nat44-nat:v1.0.2

# 验证 VPP 接口配置
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 interfaces
```

**预期输出**:
```
NAT44 interfaces:
  memif0/0 in   (inside 接口)
  memif0/1 out  (outside 接口)
```

#### ✅ P1.3 - 地址池配置与集成（v1.0.3）

```bash
# 部署 v1.0.3 镜像
kubectl set image deployment/nse-nat-vpp -n nsm-system \
  nse-nat-vpp=ifzzh520/vpp-nat44-nat:v1.0.3

# 端到端测试（参考 3.6 端到端验证）
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nsc-nat -o name | head -n1) -- ping -c 3 <test-server-ip>

# 预期：成功接收响应
```

### 4.2 P2 - 配置管理验证（v1.0.4）

```bash
# 修改 NAT 配置 ConfigMap
kubectl edit configmap nat-config-file -n nsm-system

# 修改 public_ips 为新的 IP 地址
# public_ips:
#   - 192.168.1.200

# 重启 NAT NSE 应用新配置
kubectl rollout restart deployment/nse-nat-vpp -n nsm-system
kubectl rollout status deployment/nse-nat-vpp -n nsm-system

# 验证新配置生效
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 addresses
```

**预期输出**:
```
NAT44 pool addresses:
  192.168.1.200
```

### 4.3 P3 - 模块本地化验证

#### ✅ P3.1 - 本地化 nat_types（v1.1.0）

```bash
# 编译验证
cd /path/to/cmd-nse-firewall-vpp
go build .

# Docker 镜像构建
docker build --target runtime -t ifzzh520/vpp-nat44-nat:v1.1.0 .

# K8s 部署验证
kubectl set image deployment/nse-nat-vpp -n nsm-system \
  nse-nat-vpp=ifzzh520/vpp-nat44-nat:v1.1.0

# 功能测试（确认无回归）
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nsc-nat -o name | head -n1) -- ping -c 3 <test-server-ip>
```

#### ✅ P3.2 - 本地化 nat44_ed（v1.1.1）

```bash
# 同 P3.1 验证流程，镜像版本改为 v1.1.1
docker build --target runtime -t ifzzh520/vpp-nat44-nat:v1.1.1 .
kubectl set image deployment/nse-nat-vpp -n nsm-system \
  nse-nat-vpp=ifzzh520/vpp-nat44-nat:v1.1.1
```

### 4.4 P4 - 静态端口映射验证（v1.2.0）

```bash
# 更新配置 ConfigMap 添加静态映射
kubectl edit configmap nat-config-file -n nsm-system

# 添加静态映射配置
# static_mappings:
#   - protocol: tcp
#     public_ip: 192.168.1.100
#     public_port: 8080
#     internal_ip: 172.16.1.10
#     internal_port: 80
#     tag: "web-server-mapping"

# 重启 NAT NSE
kubectl rollout restart deployment/nse-nat-vpp -n nsm-system

# 验证静态映射配置
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 static mappings
```

**预期输出**:
```
NAT44 static mappings:
  tcp local 172.16.1.10:80 external 192.168.1.100:8080 vrf 0
```

```bash
# 从外部客户端测试静态端口映射
curl http://192.168.1.100:8080

# 预期：返回内部服务器 172.16.1.10:80 的 HTTP 响应
```

---

## 5. 故障排查

### 5.1 常见问题与解决方法

#### 问题1：NAT NSE Pod 无法启动

```bash
# 查看 Pod 状态
kubectl describe pod -n nsm-system -l app=nse-nat-vpp

# 查看日志
kubectl logs -n nsm-system -l app=nse-nat-vpp
```

**可能原因**：
- SPIRE 未就绪 → 检查 `kubectl get pods -n spire`
- ConfigMap 挂载失败 → 检查 `kubectl get configmap nat-config-file -n nsm-system`
- 镜像拉取失败 → 检查 `kubectl describe pod` 中的 Events

**解决方法**：
```bash
# 确保 SPIRE 就绪
kubectl wait --for=condition=ready --timeout=5m pod -n spire -l app=spire-server

# 重新创建 ConfigMap
kubectl delete configmap nat-config-file -n nsm-system
kubectl create configmap nat-config-file --from-file=config.yaml=/tmp/nat-config.yaml -n nsm-system

# 重启 Pod
kubectl rollout restart deployment/nse-nat-vpp -n nsm-system
```

#### 问题2：NSC 无法连接到 NAT NSE

```bash
# 查看 NSC 日志
kubectl logs -n nsm-system -l app=nsc-nat

# 查看 NAT NSE 日志
kubectl logs -n nsm-system -l app=nse-nat-vpp
```

**可能原因**：
- NAT NSE 未注册到 NSM Registry
- SPIRE 身份认证失败
- 网络策略阻止连接

**解决方法**：
```bash
# 检查 NAT NSE 注册状态
kubectl logs -n nsm-system -l app=nse-nat-vpp | grep "NSE registered"

# 检查 SPIRE Agent
kubectl get pods -n spire -l app=spire-agent

# 重新部署 NSC
kubectl delete -f deployments/nat/client.yaml
kubectl apply -f deployments/nat/client.yaml
```

#### 问题3：NAT 地址转换不生效

```bash
# 检查 VPP NAT 接口配置
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 interfaces

# 检查 NAT 地址池
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 addresses

# 检查 NAT 会话
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 sessions
```

**可能原因**：
- NAT 接口角色配置错误（inside/outside 颠倒）
- NAT 地址池未配置或 IP 地址无效
- VPP 端口池耗尽

**解决方法**：
```bash
# 检查配置文件
kubectl get configmap nat-config-file -n nsm-system -o yaml

# 验证公网 IP 格式是否正确（必须是 IPv4）
# 扩大端口范围（如 port_range.min: 1024, port_range.max: 65535）

# 重启 NAT NSE 应用新配置
kubectl rollout restart deployment/nse-nat-vpp -n nsm-system
```

#### 问题4：VPP 启动失败

```bash
# 查看 VPP 启动日志
kubectl logs -n nsm-system -l app=nse-nat-vpp | grep "vpp"

# 进入容器手动启动 VPP
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- bash
vpp unix {cli-listen /run/vpp/cli.sock} api-segment { prefix vpp }
```

**可能原因**：
- VPP 配置文件错误
- 权限不足（需要 privileged 模式）
- 内存资源不足

**解决方法**：
```bash
# 检查 Pod 是否以 privileged 模式运行
kubectl get pod -n nsm-system -l app=nse-nat-vpp -o yaml | grep privileged

# 增加 Pod 内存限制
kubectl edit deployment nse-nat-vpp -n nsm-system
# 修改 resources.limits.memory 为更大值（如 1Gi）
```

### 5.2 日志查看

```bash
# 查看 NAT NSE 日志（最近 100 行）
kubectl logs -n nsm-system -l app=nse-nat-vpp --tail=100

# 查看 NSC 日志
kubectl logs -n nsm-system -l app=nsc-nat --tail=100

# 实时跟踪日志
kubectl logs -n nsm-system -l app=nse-nat-vpp -f

# 查看特定时间段日志
kubectl logs -n nsm-system -l app=nse-nat-vpp --since=10m
```

### 5.3 VPP 调试

```bash
# 进入 NAT NSE 容器
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- bash

# 查看 VPP 版本
vppctl show version

# 查看 VPP 接口
vppctl show interface

# 查看 VPP 接口地址
vppctl show interface address

# 查看 VPP 路由
vppctl show ip fib

# 查看 VPP 连接（xconnect）
vppctl show l2patch

# 启用 NAT 调试日志
vppctl set logging class nat level debug

# 查看 NAT 调试日志
vppctl show logging
```

**VPP 常用调试命令**：

| 命令 | 用途 |
|------|------|
| `show nat44 interfaces` | 查看 NAT 接口配置 |
| `show nat44 addresses` | 查看 NAT 地址池 |
| `show nat44 sessions` | 查看 NAT 会话表 |
| `show nat44 static mappings` | 查看静态端口映射 |
| `show errors` | 查看 VPP 错误统计 |
| `show trace` | 查看数据包跟踪 |
| `clear nat44 sessions` | 清除所有 NAT 会话 |

---

## 6. 版本管理和回退

### 6.1 Git 标签管理

```bash
# 查看所有标签
git tag -l

# 查看特定版本的标签信息
git show v1.0.3

# 切换到特定版本
git checkout v1.0.3

# 基于特定版本创建新分支
git checkout -b bugfix/v1.0.3 v1.0.3
```

**NAT 项目版本标签约定**：

| 标签 | 版本 | 内容 |
|------|------|------|
| `v1.0.0-acl-final` | ACL 最后稳定版本 | P1.3 之前的 ACL 代码快照 |
| `v1.0.1` | P1.1 - NAT 框架创建 | 空实现，编译通过 |
| `v1.0.2` | P1.2 - 接口角色配置 | VPP 接口配置 |
| `v1.0.3` | P1.3 - 地址池配置与集成 | 基础 SNAT 功能 |
| `v1.0.4` | P2 - 配置管理 | 配置文件加载 |
| `v1.1.0` | P3.1 - 本地化 nat_types | 基础类型本地化 |
| `v1.1.1` | P3.2 - 本地化 nat44_ed | NAT44 API 本地化 |
| `v1.2.0` | P4 - 静态端口映射 | 静态映射功能 |
| `v1.3.0` | P5 - 清理 ACL 遗留代码 | 彻底删除 ACL |

### 6.2 Docker 镜像版本回退

```bash
# 查看当前使用的镜像版本
kubectl get deployment nse-nat-vpp -n nsm-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# 回退到上一版本（例如从 v1.0.3 回退到 v1.0.2）
kubectl set image deployment/nse-nat-vpp -n nsm-system \
  nse-nat-vpp=ifzzh520/vpp-nat44-nat:v1.0.2

# 等待回退完成
kubectl rollout status deployment/nse-nat-vpp -n nsm-system

# 验证回退成功
kubectl get deployment nse-nat-vpp -n nsm-system -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### 6.3 Kubernetes 部署回退

```bash
# 查看部署历史
kubectl rollout history deployment/nse-nat-vpp -n nsm-system

# 回退到上一版本
kubectl rollout undo deployment/nse-nat-vpp -n nsm-system

# 回退到指定版本（例如 revision 2）
kubectl rollout undo deployment/nse-nat-vpp -n nsm-system --to-revision=2

# 查看回退状态
kubectl rollout status deployment/nse-nat-vpp -n nsm-system

# 验证回退后的 Pod
kubectl get pods -n nsm-system -l app=nse-nat-vpp
```

### 6.4 紧急回退流程

当遇到严重问题需要紧急回退时：

```bash
# 1. 立即回退到已知稳定版本（例如 v1.0.3）
kubectl set image deployment/nse-nat-vpp -n nsm-system \
  nse-nat-vpp=ifzzh520/vpp-nat44-nat:v1.0.3

# 2. 等待 Pod 重启
kubectl rollout status deployment/nse-nat-vpp -n nsm-system

# 3. 验证基础功能
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nsc-nat -o name | head -n1) -- ping -c 3 <test-server-ip>

# 4. 检查日志确认无错误
kubectl logs -n nsm-system -l app=nse-nat-vpp --tail=50

# 5. 记录回退原因和问题现象
echo "$(date): 回退到 v1.0.3，原因: [填写原因]" >> /tmp/rollback-log.txt
```

### 6.5 版本对比

```bash
# 对比两个版本的配置差异
kubectl diff -f deployments/nat/nse-nat-v1.0.2.yaml
kubectl diff -f deployments/nat/nse-nat-v1.0.3.yaml

# 对比两个镜像的构建历史
docker history ifzzh520/vpp-nat44-nat:v1.0.2
docker history ifzzh520/vpp-nat44-nat:v1.0.3
```

---

## 附录 A：环境变量配置

NAT NSE 支持以下环境变量：

| 变量名 | 默认值 | 描述 | 示例 |
|--------|--------|------|------|
| `NSM_NAT_CONFIG_PATH` | `/etc/nat/config.yaml` | NAT 配置文件路径 | `/custom/path/config.yaml` |
| `NSM_LOG_LEVEL` | `INFO` | 日志级别 | `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `NSM_CONNECT_TO` | `unix:///var/lib/networkservicemesh/nsm.io.sock` | NSM Manager 连接地址 | `tcp://nsm-manager:5001` |
| `NSM_NAME` | `nat-service` | 网络服务名称 | `custom-nat` |

**在 K8s Deployment 中设置环境变量**：

```yaml
env:
  - name: NSM_NAT_CONFIG_PATH
    value: "/etc/nat/config.yaml"
  - name: NSM_LOG_LEVEL
    value: "DEBUG"
```

---

## 附录 B：性能测试

### B.1 并发连接测试

```bash
# 部署多个 NSC 实例测试并发性能
kubectl scale deployment nsc-nat -n nsm-system --replicas=10

# 监控 NAT NSE 性能
kubectl top pod -n nsm-system -l app=nse-nat-vpp

# 查看 NAT 会话数量
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nse-nat-vpp -o name | head -n1) -- vppctl show nat44 sessions | wc -l
```

### B.2 吞吐量测试

```bash
# 使用 iperf3 测试 NAT 吞吐量
# 在测试服务器启动 iperf3 服务端
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=test-server -o name | head -n1) -- iperf3 -s

# 在 NSC 启动 iperf3 客户端
kubectl exec -n nsm-system -it $(kubectl get pod -n nsm-system -l app=nsc-nat -o name | head -n1) -- iperf3 -c <test-server-ip> -t 30
```

**预期吞吐量**：
- TCP: > 1 Gbps
- UDP: > 1 Gbps
- 延迟增加: < 1 ms

---

## 附录 C：完整部署示例

### C.1 完整 YAML 配置

**NAT 配置 ConfigMap**（`nat-config.yaml`）：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nat-config-file
  namespace: nsm-system
data:
  config.yaml: |
    public_ips:
      - 192.168.1.100
      - 192.168.1.101
    port_range:
      min: 1024
      max: 65535
    vrf_id: 0
    static_mappings:
      - protocol: tcp
        public_ip: 192.168.1.100
        public_port: 8080
        internal_ip: 172.16.1.10
        internal_port: 80
        tag: "web-server-mapping"
```

**NAT NSE Deployment**（`nse-nat.yaml`）：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nse-nat-vpp
  namespace: nsm-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nse-nat-vpp
  template:
    metadata:
      labels:
        app: nse-nat-vpp
    spec:
      containers:
      - name: nse-nat-vpp
        image: ifzzh520/vpp-nat44-nat:v1.0.3
        imagePullPolicy: IfNotPresent
        env:
          - name: NSM_NAT_CONFIG_PATH
            value: "/etc/nat/config.yaml"
          - name: NSM_LOG_LEVEL
            value: "INFO"
        volumeMounts:
          - name: nat-config
            mountPath: /etc/nat
          - name: spire-agent-socket
            mountPath: /run/spire/sockets
        securityContext:
          privileged: true
      volumes:
        - name: nat-config
          configMap:
            name: nat-config-file
        - name: spire-agent-socket
          hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
```

**NSC Deployment**（`client.yaml`）：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nsc-nat
  namespace: nsm-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nsc-nat
  template:
    metadata:
      labels:
        app: nsc-nat
    spec:
      containers:
      - name: nsc-nat
        image: networkservicemesh/cmd-nsc:v1.15.0-rc.1
        imagePullPolicy: IfNotPresent
        env:
          - name: NSM_NETWORK_SERVICES
            value: "kernel://nat-service/nsm0"
        volumeMounts:
          - name: spire-agent-socket
            mountPath: /run/spire/sockets
      volumes:
        - name: spire-agent-socket
          hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
```

### C.2 一键部署脚本

```bash
#!/bin/bash
# quick-deploy.sh - VPP NAT NSE 一键部署脚本

set -e

echo "=== VPP NAT 网络服务端点快速部署 ==="

# 1. 创建命名空间
echo "[1/6] 创建命名空间..."
kubectl create namespace nsm-system --dry-run=client -o yaml | kubectl apply -f -

# 2. 部署 SPIRE
echo "[2/6] 部署 SPIRE..."
if ! kubectl get namespace spire &> /dev/null; then
    git clone https://github.com/networkservicemesh/deployments-k8s.git /tmp/deployments-k8s
    kubectl apply -k /tmp/deployments-k8s/examples/spire/single_cluster
    kubectl wait --for=condition=ready --timeout=5m pod -n spire -l app=spire-server
fi

# 3. 创建 NAT 配置
echo "[3/6] 创建 NAT 配置..."
cat <<EOF > /tmp/nat-config.yaml
public_ips:
  - 192.168.1.100
port_range:
  min: 1024
  max: 65535
vrf_id: 0
EOF

kubectl create configmap nat-config-file \
  --from-file=config.yaml=/tmp/nat-config.yaml \
  -n nsm-system --dry-run=client -o yaml | kubectl apply -f -

# 4. 部署 NAT NSE
echo "[4/6] 部署 NAT NSE..."
kubectl apply -f deployments/nat/nse-nat.yaml
kubectl wait --for=condition=ready --timeout=5m pod -n nsm-system -l app=nse-nat-vpp

# 5. 部署测试服务器
echo "[5/6] 部署测试服务器..."
kubectl apply -f deployments/test/test-server.yaml
kubectl wait --for=condition=ready --timeout=3m pod -n nsm-system -l app=test-server

# 6. 部署 NSC
echo "[6/6] 部署 NSC..."
kubectl apply -f deployments/nat/client.yaml
kubectl wait --for=condition=ready --timeout=3m pod -n nsm-system -l app=nsc-nat

echo "=== 部署完成 ==="
echo "验证命令："
echo "  kubectl get pods -n nsm-system"
echo "  kubectl exec -n nsm-system -it \$(kubectl get pod -n nsm-system -l app=nsc-nat -o name | head -n1) -- ping -c 3 <test-server-ip>"
```

---

## 结语

本快速开始指南涵盖了 VPP NAT 网络服务端点从本地开发到 Kubernetes 部署的完整流程。如果遇到问题，请参考 [故障排查](#5-故障排查) 章节。

**相关文档**：
- [Feature Spec](./spec.md) - 功能规格说明
- [Research](./research.md) - 技术研究和验证
- [Plan](./plan.md) - 实施计划
- [Contracts](./contracts/vpp-nat44-api.md) - VPP API 调用规范

**获取支持**：
- GitHub Issues: https://github.com/ifzzh/cmd-nse-firewall-vpp/issues
- VPP 官方文档: https://docs.fd.io/vpp/
- NSM 官方文档: https://networkservicemesh.io/

**版本信息**：
- 文档版本: 1.0
- 最后更新: 2025-01-15
- 对应代码版本: v1.0.3
