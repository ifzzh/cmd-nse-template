# VPP NAT44 网络服务端点测试

本目录包含 VPP NAT44 网络服务端点的 Kubernetes 部署和测试文件。

## 架构说明

```
NSC (Client) → NAT NSE (VPP NAT44) → Server (nginx)
                   ↓
               公网 IP: 192.168.1.100
```

**功能**:
- NSC 发送的数据包源 IP 被转换为 NAT 公网 IP (192.168.1.100)
- 支持 ICMP、TCP、UDP 协议的地址转换
- Server 端看到的源 IP 是 NAT 公网 IP，而非 NSC 内网 IP

## 版本信息

- **NAT NSE 镜像**: `ifzzh520/vpp-nat44-nat:v1.0.4`
- **NAT 公网 IP**: 192.168.1.100（硬编码）
- **VPP 版本**: v24.10.0-4-ga9d527a67

## 部署步骤

### 1. 构建 Docker 镜像

```bash
cd /path/to/cmd-nse-firewall-vpp
docker build -t ifzzh520/vpp-nat44-nat:v1.0.4 -t ifzzh520/vpp-nat44-nat:latest .
```

### 2. 部署到 Kubernetes

```bash
kubectl apply -k samenode-nat/
```

### 3. 验证部署

```bash
kubectl get pods -n ns-nse-composition
```

## 功能测试

### ICMP Ping 测试

```bash
SERVER_IP=$(kubectl exec -n ns-nse-composition deployment/nginx -- ip addr show nsm-1 | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)
kubectl exec -n ns-nse-composition deployment/alpine -- ping -c 4 $SERVER_IP
```

### VPP NAT 会话验证

```bash
NAT_POD=$(kubectl get pods -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].metadata.name}')

# 查看 NAT 地址池
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 addresses

# 查看 NAT 接口
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 interfaces

# 查看 NAT 会话
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions
```

## 清理环境

```bash
kubectl delete -k samenode-nat/
```

## 版本历史

- **v1.0.4** (2025-01-19): 修复 NAT44 ED 插件启用问题（解决 VPP API -126 错误）
- **v1.0.3** (2025-01-15): 地址池配置与 ACL→NAT 转型
- **v1.0.2** (2025-01-15): 接口角色配置
- **v1.0.1** (2025-01-15): NAT 框架创建
