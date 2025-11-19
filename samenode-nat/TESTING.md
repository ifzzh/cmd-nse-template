# VPP NAT44 测试报告

**测试日期**: 2025-11-19
**镜像版本**: `ifzzh520/vpp-nat44-nat:v1.0.4`
**测试环境**: samenode-nat (Kubernetes)

---

## 测试结果总结

### ✅ 测试通过

**v1.0.4 版本成功修复了 VPP NAT44 ED 插件启用问题！**

---

## 问题修复验证

### 问题描述（v1.0.3）

- **错误代码**: VPP API 返回 -126 (`VNET_API_ERROR_UNSUPPORTED`)
- **错误位置**: `configureNATAddressPool()` 配置地址池时
- **影响**: alpine 容器无法启动，NAT 服务链无法建立

### 修复方案（v1.0.4）

1. 新增 `enableNAT44Plugin()` 函数
2. 在 `NewServer()` 初始化时显式启用 NAT44 ED 插件
3. 确保插件在配置地址池前启用

---

## 测试日志分析

### 1. NAT44 ED 插件启用成功 ✅

**日志时间**: 2025/11/19 07:56:43

```log
[INFO] [nat_server:enable_plugin] NAT44 ED 插件启用成功
```

**验证**: 插件在服务器初始化时成功启用，无报错。

---

### 2. NAT 地址池配置成功 ✅

**日志时间**: Nov 19 07:56:45.026

```log
[INFO] [nat_server:configure_pool] 配置 NAT 地址池成功: 192.168.1.100
```

**验证**:
- 公网 IP 192.168.1.100 成功添加到地址池
- 不再出现 -126 错误

---

### 3. NAT 接口配置成功 ✅

**日志时间**: Nov 19 07:56:45.027

```log
[INFO] [nat_server:configure] 配置 NAT 接口成功: swIfIndex=2, role=inside
```

**验证**:
- 接口索引: 2
- 接口角色: inside (NSC 侧)
- 配置成功无报错

---

### 4. NSC 连接成功建立 ✅

**日志时间**: Nov 19 07:56:45.299

```log
[INFO] successfully connected to nse-composition
```

**验证**:
- alpine 客户端成功连接到服务链
- IP 分配:
  - NSC (alpine): 172.16.1.101/32
  - Server (nginx): 172.16.1.100/32

---

## 服务链路径验证

完整的网络服务链路径：

```
alpine (NSC)
  ↓
nsmgr-5v547
  ↓
forwarder-vpp-f6pdv
  ↓
nse-nat-vpp-5b9c4b76d9-mf9s9 ← 我们的 NAT NSE ✅
  ↓
nsmgr-5v547
  ↓
forwarder-vpp-f6pdv
  ↓
nse-kernel-5d96b8d6d5-tm654 (nginx server)
```

**验证**:
- 服务链完整建立
- NAT NSE 正确集成到服务链中
- 数据路径通畅

---

## 问题对比

### v1.0.3 (失败)

```log
❌ VPP API 返回错误: -126（IP: 192.168.1.100）
❌ Error returned from nat/natServer.Request
❌ alpine 容器无法启动
```

### v1.0.4 (成功)

```log
✅ NAT44 ED 插件启用成功
✅ 配置 NAT 地址池成功: 192.168.1.100
✅ 配置 NAT 接口成功: swIfIndex=2, role=inside
✅ successfully connected to nse-composition
```

---

## 测试结论

### 修复验证 ✅

1. **NAT44 ED 插件启用**: ✅ 成功
2. **地址池配置**: ✅ 成功（不再出现 -126 错误）
3. **接口配置**: ✅ 成功
4. **NSC 连接建立**: ✅ 成功
5. **服务链完整性**: ✅ 完整

### 关键改进

- **启动时插件启用**: 确保后续操作正常
- **错误提前暴露**: 插件启用失败时触发 panic
- **代码可维护性**: 插件启用逻辑独立封装

---

## 用户实际测试验证 (2025-11-19)

### 基础功能测试 ✅

#### 1. NAT 地址池验证
```bash
$ kubectl exec -n ns-nse-composition nse-nat-vpp-5b9c4b76d9-mf9s9 -- vppctl show nat44 addresses
NAT44 pool addresses:
192.168.1.100
  tenant VRF: 0
NAT44 twice-nat pool addresses:
```
**结果**: ✅ 地址池配置正确

#### 2. NAT 接口验证
```bash
$ kubectl exec -n ns-nse-composition nse-nat-vpp-5b9c4b76d9-mf9s9 -- vppctl show nat44 interfaces
NAT44 interfaces:
 memif1196435762/0 in
```
**结果**: ✅ Inside 接口配置正确

#### 3. NSC 网络接口验证
```bash
$ kubectl exec -n ns-nse-composition alpine -- ip addr show nsm-1
3: nsm-1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9000 qdisc mq state UNKNOWN qlen 1000
    link/ether 02:fe:b4:68:1d:9b brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.101/32 scope global nsm-1
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:b4ff:fe68:1d9b/64 scope link
       valid_lft forever preferred_lft forever
```
**结果**: ✅ NSC IP 地址: 172.16.1.101/32

#### 4. Server 网络接口验证
```bash
$ kubectl exec -n ns-nse-composition deployment/nse-kernel -- ip addr show | grep -A2 "nse-compos"
3: nse-compos-tAWd: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9000 qdisc mq state UNKNOWN qlen 1000
    link/ether 02:fe:1e:98:ae:0c brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.100/32 scope global nse-compos-tAWd
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:1eff:fe98:ae0c/64 scope link
```
**结果**: ✅ Server IP 地址: 172.16.1.100/32

#### 5. ICMP 连通性测试
```bash
$ kubectl exec -n ns-nse-composition alpine -- ping -c 4 172.16.1.100
PING 172.16.1.100 (172.16.1.100): 56 data bytes
64 bytes from 172.16.1.100: seq=0 ttl=64 time=0.393 ms
64 bytes from 172.16.1.100: seq=1 ttl=64 time=0.445 ms
64 bytes from 172.16.1.100: seq=2 ttl=64 time=0.585 ms
64 bytes from 172.16.1.100: seq=3 ttl=64 time=0.230 ms

--- 172.16.1.100 ping statistics ---
4 packets transmitted, 4 packets received, 0% packet loss
round-trip min/avg/max = 0.230/0.413/0.585 ms
```
**结果**: ✅ Ping 成功
- **丢包率**: 0%
- **平均延迟**: 0.413ms
- **最小延迟**: 0.230ms
- **最大延迟**: 0.585ms

---

## 附录：测试命令

### VPP 验证

```bash
NAT_POD=$(kubectl get pods -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].metadata.name}')

# 查看 NAT 地址池
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 addresses

# 查看 NAT 接口
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 interfaces

# 查看 NAT 会话
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions
```

### iperf3 性能测试（可选）

```bash
# 1. 安装 iperf3
kubectl exec -n ns-nse-composition alpine -- sh -c "apk update && apk add iperf3"
kubectl exec -n ns-nse-composition deployment/nse-kernel -- sh -c "apt-get update && apt-get install -y iperf3"

# 2. 启动服务器
kubectl exec -n ns-nse-composition deployment/nse-kernel -- sh -c "iperf3 -s -D"

# 3. 运行测试
SERVER_IP=$(kubectl exec -n ns-nse-composition deployment/nse-kernel -- ip addr show | grep 'inet ' | grep nse-compos | awk '{print $2}' | cut -d/ -f1)
kubectl exec -n ns-nse-composition alpine -- iperf3 -c $SERVER_IP -t 10

# 4. 查看 NAT 会话
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions

# 5. 清理
kubectl exec -n ns-nse-composition deployment/nse-kernel -- pkill iperf3
```

---

**测试人员**: Claude Code + User
**审核状态**: 通过 ✅
**版本状态**: v1.0.4 已推送到 Docker Hub
**实际验证**: 用户在真实 K8s 环境中验证通过
