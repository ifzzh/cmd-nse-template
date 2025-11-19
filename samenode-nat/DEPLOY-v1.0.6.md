# v1.0.6 部署指南

## 概述

v1.0.6 是一个重大更新，从 L2 xconnect 模式迁移到 L3 路由模式，解决了 v1.0.5 中 NAT 会话无法创建的根本问题。

**关键变化**：
- ✅ 移除 L2 xconnect，采用 L3 路由模式
- ✅ NAT 现在可以正常工作（会话数 > 0）
- ✅ 数据包经过正确的 L3 路径，触发 NAT 转换

## 快速部署

### 1. 部署 v1.0.6

```bash
# 更新部署
kubectl apply -f samenode-nat/nse-nat/nat.yaml

# 等待 Pod 就绪
kubectl wait --for=condition=ready pod -l app=nse-nat-vpp -n ns-nse-composition --timeout=60s

# 验证镜像版本
kubectl get pod -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].spec.containers[0].image}'
# 应该输出: ifzzh520/vpp-nat44-nat:v1.0.6
```

### 2. 运行验证脚本

```bash
cd samenode-nat
./verify-v1.0.6.sh
```

验证脚本会检查：
- ✅ Pod 状态和镜像版本
- ✅ NAT44 ED 插件状态
- ✅ **接口模式（L3 vs L2）** - 关键检查
- ✅ NAT 接口配置（inside + outside）
- ✅ **路由表（L3 模式特有）** - 新增检查
- ✅ NAT 会话状态
- ✅ **确认 L2 xconnect 已移除** - 新增检查

### 3. 预期验证结果

#### v1.0.5（L2 模式，失败）

```bash
$ vppctl show interface address
memif1013904223/0 (up):
  L2 xconnect memif1196435762/0    ❌ L2 转发，NAT 无效
memif1196435762/0 (up):
  L2 xconnect memif1013904223/0    ❌ L2 转发，NAT 无效

$ vppctl show nat44 sessions
NAT44 ED sessions:
-------- thread 0 vpp_main: 0 sessions --------   ❌ 会话数为 0
```

#### v1.0.6（L3 模式，成功）✅

```bash
$ vppctl show interface address
memif1013904223/0 (up):
  L3 10.60.1.1/24                  ✅ L3 IP 地址
memif1196435762/0 (up):
  L3 10.60.2.1/24                  ✅ L3 IP 地址

$ vppctl show ip fib
ipv4-VRF:0, fib_index:0
  10.60.1.0/24                      ✅ 路由条目
    unicast-ip4-chain
      [@0]: dpo-load-balance: ...
        [0] [@2]: dpo-receive: 10.60.1.0/24 on memif1013904223/0
  10.60.2.0/24                      ✅ 路由条目
    unicast-ip4-chain
      [@0]: dpo-load-balance: ...
        [0] [@2]: dpo-receive: 10.60.2.0/24 on memif1196435762/0

$ vppctl show l2 xconnect
                                   ✅ 空输出，xconnect 已移除

$ vppctl show nat44 sessions
NAT44 ED sessions:
-------- thread 0 vpp_main: X sessions -------- ✅ X > 0（有流量后）
in2out 10.60.1.2:12345 192.168.1.100:12345 fib 0 proto 6 ...
```

## 功能测试

### 测试 NAT 会话创建

```bash
# 1. 获取 NSC Pod 名称
NSC_POD=$(kubectl get pods -n ns-nse-composition -l app=alpine-nsc -o jsonpath='{.items[0].metadata.name}')

# 2. 从 NSC 发送测试流量（示例）
kubectl exec -n ns-nse-composition $NSC_POD -- ping -c 5 8.8.8.8

# 3. 检查 NAT 会话
NAT_POD=$(kubectl get pods -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions
```

### 验证 NAT 转换

```bash
# 查看 NAT 统计信息
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 statistics

# 查看 NAT 地址池
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 addresses
```

## 对比 L2 vs L3 模式

### 数据包处理路径

**L2 Xconnect 模式（v1.0.5）**：
```
ethernet-input → l2-input → l2-xconnect → l2-output
                    ↑
                    └─ NAT 未触发（NAT 在 L3 层）
```

**L3 路由模式（v1.0.6）**：
```
ethernet-input → ip4-input → ip4-lookup → nat44-ed-in2out → ip4-rewrite
                                  ↑
                                  └─ NAT 在此触发
```

### VPP Feature Arc

**L2 模式**：
- Feature Arc: `l2-input-feat-arc` → `l2-output-feat-arc`
- NAT 节点：**不在此路径**（NAT 只注册在 `ip4-unicast`）
- 结果：NAT 无法触发

**L3 模式**：
- Feature Arc: `ip4-input` → `ip4-unicast` → `ip4-output`
- NAT 节点：**在 `ip4-unicast` 上**（`nat44-ed-in2out`、`nat44-ed-out2in`）
- 结果：NAT 正常工作

## 常见问题

### Q1: 验证脚本显示仍在使用 L2 xconnect？

**原因**：镜像未正确更新

**解决**：
```bash
# 1. 检查镜像版本
kubectl get pod -n ns-nse-composition $NAT_POD -o jsonpath='{.spec.containers[0].image}'

# 2. 如果不是 v1.0.6，强制重新拉取
kubectl delete pod -n ns-nse-composition $NAT_POD

# 3. 等待新 Pod 启动
kubectl wait --for=condition=ready pod -l app=nse-nat-vpp -n ns-nse-composition --timeout=60s

# 4. 重新验证
./verify-v1.0.6.sh
```

### Q2: 路由表中没有条目？

**原因**：NSM 连接未建立或 IP 地址未分配

**排查**：
```bash
# 1. 查看 NSM 日志
kubectl logs -n ns-nse-composition $NAT_POD | grep -i "ip address"

# 2. 查看 NSM 连接
kubectl logs -n ns-nse-composition $NAT_POD | grep -i "connection"

# 3. 查看接口状态
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface
```

### Q3: NAT 会话一直为 0？

**可能原因**：
1. **没有流量**：NAT 会话只在有流量时才创建（正常）
2. **路由问题**：检查 `show ip fib` 是否有路由
3. **防火墙规则**：检查上游是否有 ACL 阻止流量

**解决**：
```bash
# 1. 发送测试流量
kubectl exec -n ns-nse-composition $NSC_POD -- ping -c 5 8.8.8.8

# 2. 立即检查会话
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions

# 3. 如果仍为 0，检查数据包计数
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface
# 查看 memif 接口的 rx packets 和 tx packets
```

## 回滚到 v1.0.5

如果 v1.0.6 出现问题，可以快速回滚：

```bash
# 1. 修改 nat.yaml
sed -i 's/v1.0.6/v1.0.5/g' samenode-nat/nse-nat/nat.yaml

# 2. 重新部署
kubectl delete -f samenode-nat/nse-nat/nat.yaml
kubectl apply -f samenode-nat/nse-nat/nat.yaml

# 3. 验证
kubectl get pod -n ns-nse-composition -l app=nse-nat-vpp
```

**注意**：回滚后 NAT 会话将再次变为 0（v1.0.5 的已知问题）

## 技术细节

### 代码变更摘要

**main.go**：
```diff
- "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"
+ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
+ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/routes"

# 服务器链
- xconnect.NewServer(vppConn),
+ ipaddress.NewServer(vppConn),
  nat.NewServer(vppConn, []net.IP{net.ParseIP("192.168.1.100")}),
  mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
      memif.MECHANISM: chain.NewNetworkServiceServer(
          memif.NewServer(ctx, vppConn),
+         routes.NewServer(vppConn),
      ),
  }),

# 客户端链
- xconnect.NewClient(vppConn),
+ ipaddress.NewClient(vppConn),
+ routes.NewClient(vppConn),
```

### 参考资料

- **详细变更日志**：[CHANGELOG-v1.0.6.md](CHANGELOG-v1.0.6.md)
- **技术研究报告**：[.claude/vpp-acl-nat-xconnect-research.md](../.claude/vpp-acl-nat-xconnect-research.md)
- **L3 路由参考实现**：[cmd-nse-vl3-vpp](https://github.com/networkservicemesh/cmd-nse-vl3-vpp)
- **VPP NAT44 ED 文档**：[VPP NAT44](https://fd.io/docs/vpp/master/nat44.html)

## 下一步

部署成功后，建议：

1. **性能测试**：使用 iperf3 测试 NAT 吞吐量和延迟
2. **压力测试**：创建大量并发连接，验证 NAT 会话管理
3. **监控集成**：接入 Prometheus/Grafana 监控 NAT 指标
4. **生产优化**：
   - 调整 NAT 地址池（当前硬编码为 192.168.1.100）
   - 配置 NAT 会话超时参数
   - 优化 VPP 内存和线程配置

## 获取帮助

遇到问题？

1. 查看日志：`kubectl logs -n ns-nse-composition $NAT_POD`
2. 检查 VPP 状态：`kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show ...`
3. 参考文档：[README.md](../README.md)、[CHANGELOG-v1.0.6.md](CHANGELOG-v1.0.6.md)
