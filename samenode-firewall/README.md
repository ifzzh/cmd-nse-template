# NSE 防火墙组合测试 / Test NSE Firewall Composition

本示例演示了一个基于 VPP ACL 的防火墙网络服务端点，展示了如何使用 NSM 进行服务组合（链式调用）。
它涉及 kernel 和 memif 机制的组合，以及支持 VPP 的端点。

This example demonstrates a Network Service Endpoint with VPP ACL-based firewall functionality.
It demonstrates how NSM allows for service composition (chaining).
It involves a combination of kernel and memif mechanisms, as well as VPP enabled endpoints.

## 功能特性 / Features

- ✅ **VPP ACL 防火墙**: 基于 VPP 的高性能访问控制列表
- ✅ **灵活的规则配置**: 通过 ConfigMap 配置防火墙规则
- ✅ **中文友好**: 代码注释和日志信息支持中文
- ✅ **容器化部署**: 使用 Docker 镜像 `ifzzh520/vpp-acl-firewall:v1.0.0`

## Requires

Make sure that you have completed steps from [basic](../../basic) or [memory](../../memory) setup.

## 部署步骤 / Run

### 1. 部署 NSC 和 NSE / Deploy NSC and NSE
```bash
kubectl apply -k ./samenode-firewall/
```

### 2. 查看部署状态 / Check Deployment Status
```bash
# 实时监控 Pod 状态
watch kubectl get pod -n ns-nse-composition -o wide

# 查看 VPP ACL 规则
kubectl exec -n ns-nse-composition deploy/nse-firewall-vpp -- vppctl show acl-plugin acl

# 保存日志到本地
kubectl logs -n ns-nse-composition alpine -c cmd-nsc-init > logs/cmd-nsc-init.log
kubectl logs -n ns-nse-composition deploy/nse-firewall-vpp > logs/nse-firewall-vpp.log
```

### 3. 等待应用就绪 / Wait for Applications Ready
```bash
kubectl wait --for=condition=ready --timeout=5m pod -l app=alpine -n ns-nse-composition
kubectl wait --for=condition=ready --timeout=1m pod -l app=nse-kernel -n ns-nse-composition
```

## 功能测试 / Testing

### 4. 基本连通性测试 / Basic Connectivity Test

#### 4.1 从 NSC ping NSE / Ping from NSC to NSE
```bash
kubectl exec pods/alpine -n ns-nse-composition -- ping -c 4 172.16.1.100
```

#### 4.2 从 NSE ping NSC / Ping from NSE to NSC
```bash
kubectl exec deployments/nse-kernel -n ns-nse-composition -- ping -c 4 172.16.1.101
```

### 5. 防火墙规则测试 / Firewall Rule Test

#### 5.1 测试 TCP 端口 5201（应该允许）/ Test TCP Port 5201 (Should Allow)
```bash
kubectl exec pods/alpine -n ns-nse-composition -- wget -O /dev/null --timeout 5 "172.16.1.100:5201"
```

#### 5.2 测试 TCP 端口 80（应该被阻止）/ Test TCP Port 80 (Should Block)
```bash
kubectl exec pods/alpine -n ns-nse-composition -- wget -O /dev/null --timeout 5 "172.16.1.100:80"
if [ 0 -eq $? ]; then
  echo "错误: 端口 80 可访问（应该被阻止）/ error: port :80 is available" >&2
  false
else
  echo "成功: 端口 80 不可访问（防火墙规则生效）/ success: port :80 is blocked"
fi
```

#### 5.3 测试 TCP 端口 8080（应该被阻止）/ Test TCP Port 8080 (Should Block)
```bash
kubectl exec pods/alpine -n ns-nse-composition -- wget -O /dev/null --timeout 5 "172.16.1.100:8080"
if [ 0 -eq $? ]; then
  echo "错误: 端口 8080 可访问（应该被阻止）/ error: port :8080 is available" >&2
  false
else
  echo "成功: 端口 8080 不可访问（防火墙规则生效）/ success: port :8080 is blocked"
fi
```

### 6. 性能测试 / Performance Test (iperf3)

#### 6.1 安装 iperf3 / Install iperf3
```bash
# 客户端（NSC）
kubectl exec -it pods/alpine -n ns-nse-composition -- apk add iperf3

# 服务端（NSE）
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- apk add iperf3
```

#### 6.2 启动 iperf3 服务端 / Start iperf3 Server
```bash
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- iperf3 -s -p 5201
```

#### 6.3 运行 iperf3 客户端测试 / Run iperf3 Client Test
```bash
# TCP 测试（端口 5201 允许通过防火墙）
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -p 5201 -t 30

# UDP 测试（端口 5201 允许通过防火墙）
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -p 5201 -t 30 -u -b 1G
```

## 防火墙配置说明 / Firewall Configuration

防火墙规则在 [config-file.yaml](config-file.yaml) 中定义。当前配置允许以下流量：

- ✅ **TCP 5201**: 允许（iperf3 性能测试）
- ✅ **UDP 5201**: 允许（iperf3 性能测试）
- ✅ **ICMP**: 允许（ping 测试）
- ❌ **TCP 8080**: 禁止
- ❌ **TCP 80**: 禁止

修改 `config-file.yaml` 后重新部署即可应用新规则。

## 清理 / Cleanup

删除命名空间及所有资源 / Delete namespace and all resources:
```bash
kubectl delete ns ns-nse-composition
```

## 镜像信息 / Image Information

- **镜像仓库**: [ifzzh520/vpp-acl-firewall](https://hub.docker.com/r/ifzzh520/vpp-acl-firewall)
- **当前版本**: v1.0.0
- **拉取命令**: `docker pull ifzzh520/vpp-acl-firewall:v1.0.0`
- **基础镜像**: ghcr.io/networkservicemesh/govpp/vpp:v24.10.0-4-ga9d527a67
- **镜像大小**: 235MB

## 技术栈 / Technology Stack

- **VPP**: v24.10.0（高性能数据平面）
- **Network Service Mesh**: v1.15.0-rc.1
- **SPIRE**: v1.8.0（SPIFFE 身份认证）
- **Go**: 1.23.8
- **OpenTelemetry**: 可观测性支持

## 故障排查 / Troubleshooting

### 1. Pod 无法启动
```bash
# 查看 Pod 状态
kubectl describe pod -n ns-nse-composition -l app=nse-firewall-vpp

# 查看日志
kubectl logs -n ns-nse-composition deploy/nse-firewall-vpp
```

### 2. ACL 规则不生效
```bash
# 检查 VPP ACL 配置
kubectl exec -n ns-nse-composition deploy/nse-firewall-vpp -- vppctl show acl-plugin acl

# 检查接口配置
kubectl exec -n ns-nse-composition deploy/nse-firewall-vpp -- vppctl show int

# 检查 ConfigMap
kubectl get cm firewall-config-file -n ns-nse-composition -o yaml
```

### 3. SPIRE 认证问题
```bash
# 检查 SPIRE Agent
kubectl get pod -n spire

# 查看 SPIRE Agent 日志
kubectl logs -n spire daemonset/spire-agent
```
