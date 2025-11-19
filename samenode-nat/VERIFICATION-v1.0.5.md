# NAT Outside 接口配置修复验证指南 (v1.0.5)

## 修复概述

**问题**：NAT 会话数为 0，VPP 只做 L2 转发而非 NAT 转换

**根因**：
- NAT NSE 只在 server 端链中配置了 NAT Server
- client 端链缺少 NAT 配置
- 导致只有 inside 接口配置，缺少 outside 接口

**解决方案**：
1. 创建 `internal/nat/client.go` 实现 `NetworkServiceClient` 接口
2. 在 `main.go` 的客户端链中添加 `nat.NewClient(vppConn)`
3. 现在两个接口都会配置 NAT 角色（inside + outside）

## 部署步骤

### 1. 更新部署配置使用 v1.0.5 镜像

```bash
cd /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp/samenode-nat
```

确认 `nse-nat/patch-nse-nat-vpp.yaml` 中的镜像版本：
```yaml
image: ifzzh520/vpp-nat44-nat:v1.0.5
```

### 2. 重新部署 NAT NSE

```bash
# 删除旧的 NAT NSE
kubectl delete -f nse-nat/ -n ns-nse-composition

# 等待 Pod 完全删除
kubectl wait --for=delete pod -l app=nse-nat-vpp -n ns-nse-composition --timeout=60s

# 重新部署
kubectl apply -f nse-nat/ -n ns-nse-composition

# 等待 Pod 就绪
kubectl wait --for=condition=ready pod -l app=nse-nat-vpp -n ns-nse-composition --timeout=120s
```

### 3. 重新部署 Alpine Client（触发连接）

```bash
# 删除 alpine
kubectl delete -f alpine-nsc.yaml -n ns-nse-composition

# 等待删除
kubectl wait --for=delete pod -l app=alpine -n ns-nse-composition --timeout=60s

# 重新部署
kubectl apply -f alpine-nsc.yaml -n ns-nse-composition

# 等待就绪
kubectl wait --for=condition=ready pod -l app=alpine -n ns-nse-composition --timeout=120s
```

## 验证步骤

### 验证 1: 检查两个接口都配置了 NAT 角色

**预期结果**：应该看到两个接口，一个 inside，一个 outside

```bash
NAT_POD=$(kubectl get pod -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 interfaces
```

**预期输出**：
```
NAT44 interfaces:
 memif1196435762/0 in       ← Interface A (server 端，连接 NSC)
 memif1013904223/0 out      ← Interface B (client 端，连接下游 NSE)
```

**对比 v1.0.4**（修复前）：
```
NAT44 interfaces:
 memif1196435762/0 in       ← 只有一个接口！
```

### 验证 2: 检查 NAT 会话是否创建

**预期结果**：应该能看到 NAT 会话（不再是 0）

```bash
# 从 alpine 发送 ICMP 测试流量
kubectl exec -n ns-nse-composition alpine -- ping -c 4 172.16.1.100

# 检查 NAT 会话
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions
```

**预期输出**（示例）：
```
NAT44 ED sessions:
-------- thread 0 vpp_main: 1 sessions --------
  i2o 172.16.0.100:12345 -> 172.16.1.100:0 proto icmp fib 0
    o2i 172.16.1.100:0 -> 192.168.1.100:12345 fib 0
    index 0, total pkts 8, total bytes 672
```

**对比 v1.0.4**（修复前）：
```
NAT44 ED sessions:
-------- thread 0 vpp_main: 0 sessions --------
```

### 验证 3: 检查 VPP 接口统计

```bash
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface
```

**预期结果**：
- 两个 MEMIF 接口都有流量
- 接口名称应该与 NAT interfaces 输出一致

### 验证 4: 检查 NAT 地址池配置

```bash
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 addresses
```

**预期输出**：
```
NAT44 pool addresses:
192.168.1.100
  tenant VRF independent
  10 translations, 10 sessions
```

### 验证 5: 端到端连通性测试

```bash
# ICMP 测试
kubectl exec -n ns-nse-composition alpine -- ping -c 4 172.16.1.100

# iperf3 性能测试（如果 kernel server 运行 iperf3）
kubectl exec -n ns-nse-composition alpine -- iperf3 -c 172.16.1.100 -t 10
```

**预期结果**：
- ping 成功，0% 丢包
- iperf3 吞吐量正常（>= 1 Gbps）

## 日志验证

### 检查 NAT NSE 启动日志

```bash
kubectl logs -n ns-nse-composition $NAT_POD | grep -E "NAT44|nat_client|nat_server"
```

**预期日志关键行**：

1. **NAT44 插件启用**：
   ```
   [INFO] NAT44 ED 插件启用成功
   ```

2. **地址池配置**：
   ```
   [INFO] 配置 NAT 地址池成功: 192.168.1.100
   ```

3. **Server 端接口配置（inside）**：
   ```
   [INFO] [nat_server:configure] 配置 NAT 接口成功: swIfIndex=2, role=inside
   ```

4. **Client 端接口配置（outside）** ← **新增日志**：
   ```
   [INFO] [nat_client:request] 客户端链接口角色: outside, swIfIndex=1
   [INFO] [nat_client:configure] 配置 NAT 接口成功: swIfIndex=1, role=outside
   ```

## 故障排查

### 问题 1: 仍然只有一个接口配置为 NAT

**检查**：
```bash
kubectl describe pod -n ns-nse-composition $NAT_POD | grep Image:
```

**原因**：可能使用了旧镜像

**解决**：
```bash
# 强制拉取新镜像
kubectl delete pod -n ns-nse-composition $NAT_POD
```

### 问题 2: NAT 会话仍然为 0

**检查 1**：确认两个接口都配置了 NAT
```bash
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 interfaces
```

**检查 2**：查看日志是否有错误
```bash
kubectl logs -n ns-nse-composition $NAT_POD | grep -i error
```

**检查 3**：确认流量路径
```bash
# 检查 xconnect 配置
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show l2 xconnect
```

### 问题 3: Pod 启动失败

**检查日志**：
```bash
kubectl logs -n ns-nse-composition $NAT_POD
```

**常见错误**：
- VPP API 调用失败：检查 VPP 连接
- 接口索引未找到：可能是链顺序问题

## 成功标准

✅ **修复成功** 如果满足以下所有条件：

1. **两个接口都配置了 NAT**：
   - `show nat44 interfaces` 显示 1 个 `in` 和 1 个 `out`

2. **NAT 会话正常创建**：
   - `show nat44 sessions` 显示会话数 > 0
   - 会话包含 `i2o` 和 `o2i` 方向

3. **连通性测试通过**：
   - ping 成功，0% 丢包
   - iperf3 吞吐量正常

4. **日志无错误**：
   - 启动日志中包含 `nat_client` 相关的成功日志
   - 无 VPP API 错误

## 架构对比

### v1.0.4（修复前）

```
┌─────────────────────────────────────┐
│         NAT NSE Pod                 │
│                                     │
│  Server 端链（Interface A）          │
│  ├─ recvfd/sendfd                   │
│  ├─ up.NewServer                    │
│  ├─ nat.NewServer ✓ (配置 inside)   │  ← 只有这里配置了 NAT
│  └─ xconnect                        │
│           ↓                         │
│  Client 端链（Interface B）          │
│  ├─ metadata.NewClient              │
│  ├─ up.NewClient                    │
│  ├─ xconnect.NewClient              │  ← 缺少 NAT 配置！
│  └─ memif.NewClient                 │
└─────────────────────────────────────┘

结果：
- VPP 接口 A：NAT inside ✓
- VPP 接口 B：无 NAT 配置 ✗
- VPP 行为：L2 转发（无 NAT 转换）
- NAT 会话：0
```

### v1.0.5（修复后）

```
┌─────────────────────────────────────┐
│         NAT NSE Pod                 │
│                                     │
│  Server 端链（Interface A）          │
│  ├─ recvfd/sendfd                   │
│  ├─ up.NewServer                    │
│  ├─ nat.NewServer ✓ (配置 inside)   │  ← Server 端配置 inside
│  └─ xconnect                        │
│           ↓                         │
│  Client 端链（Interface B）          │
│  ├─ metadata.NewClient              │
│  ├─ up.NewClient                    │
│  ├─ nat.NewClient ✓ (配置 outside)  │  ← Client 端配置 outside ✓
│  ├─ xconnect.NewClient              │
│  └─ memif.NewClient                 │
└─────────────────────────────────────┘

结果：
- VPP 接口 A：NAT inside ✓
- VPP 接口 B：NAT outside ✓
- VPP 行为：NAT 转换（SNAT）
- NAT 会话：> 0 ✓
```

## 代码变更摘要

### 新增文件

**internal/nat/client.go** (140 行)
```go
// NewClient 创建新的 NAT 客户端
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient

// Request 处理客户端请求，配置接口为 NAT outside
func (n *natClient) Request(...)

// Close 清理 NAT 接口配置
func (n *natClient) Close(...)
```

### 修改文件

**main.go:242**
```diff
  client.WithAdditionalFunctionality(
      metadata.NewClient(),
      mechanismtranslation.NewClient(),
      passthrough.NewClient(config.Labels),
      up.NewClient(ctx, vppConn),
+     nat.NewClient(vppConn),  // ← 新增：配置 NAT outside 接口
      xconnect.NewClient(vppConn),
      memif.NewClient(ctx, vppConn),
      sendfd.NewClient(),
      recvfd.NewClient(),
  )
```

## 相关文档

- 原始设计：`specs/003-vpp-nat/spec.md` Q2, Q6（双接口架构）
- 测试计划：`samenode-nat/TESTING.md`
- 提交记录：`f962912` - fix(nat): 在客户端链中添加 NAT Client 配置 outside 接口 (v1.0.5)

---

**版本**: v1.0.5
**日期**: 2025-11-19
**镜像**: ifzzh520/vpp-nat44-nat:v1.0.5
**Digest**: sha256:c0179464a3990d1074e764cc6de0e2faf6db5a76efb5d81e9d73fae3c3c2c132
