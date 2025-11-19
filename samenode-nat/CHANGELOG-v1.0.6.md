# v1.0.6 变更日志 - L3 路由模式迁移

**发布日期**: 2025-11-20

## 重大变更

### 从 L2 Xconnect 迁移到 L3 路由模式

**背景**:
- v1.0.5 虽然成功配置了 NAT inside 和 outside 接口，但 NAT 会话数始终为 0
- 根本原因：L2 xconnect 在数据链路层直接转发数据包，绕过了 L3 路由处理
- NAT44 ED 插件只注册在 `ip4-unicast` feature arc（L3 层），无法在 L2 xconnect 环境下被触发

**解决方案**:
- 移除 L2 xconnect 组件
- 引入 `ipaddress` 和 `routes` 组件，启用 L3 路由模式
- 参考 `cmd-nse-vl3-vpp` 的实现模式

## 代码变更

### 1. main.go - 移除 xconnect 导入，添加 ipaddress 和 routes

```diff
// 导入部分（第32-36行）
- "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"
+ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
+ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/routes"
```

### 2. main.go - 服务器链修改（第220-232行）

```diff
// 服务器端功能链
  recvfd.NewServer(),
  sendfd.NewServer(),
  up.NewServer(ctx, vppConn),
  clienturl.NewServer(&config.ConnectTo),
- xconnect.NewServer(vppConn),                 // VPP交叉连接（L2转发）
+ ipaddress.NewServer(vppConn),                // 为接口配置 IP 地址（L3 模式）
  nat.NewServer(vppConn, []net.IP{net.ParseIP("192.168.1.100")}),
  mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
      memif.MECHANISM: chain.NewNetworkServiceServer(
          memif.NewServer(ctx, vppConn),
+         routes.NewServer(vppConn),          // 配置路由表（L3 模式）
      ),
  }),
```

### 3. main.go - 客户端链修改（第245-249行）

```diff
// 客户端功能链
  up.NewClient(ctx, vppConn),
- nat.NewClient(vppConn),
- xconnect.NewClient(vppConn),                 // VPP交叉连接（客户端）
+ ipaddress.NewClient(vppConn),                // 为接口配置 IP 地址（客户端，L3 模式）
+ routes.NewClient(vppConn),                   // 配置路由表（客户端，L3 模式）
+ nat.NewClient(vppConn),                      // NAT44 接口配置（客户端侧，配置为 outside）
  memif.NewClient(ctx, vppConn),
  sendfd.NewClient(),
  recvfd.NewClient(),
```

### 4. internal/imports/imports_linux.go - 更新导入列表

```diff
  _ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl"
+ _ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
+ _ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/routes"
  _ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
  _ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
- _ "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"
```

### 5. samenode-nat/nse-nat/nat.yaml - 镜像版本升级

```diff
  containers:
    - name: nse
-     image: ifzzh520/vpp-nat44-nat:v1.0.5
+     image: ifzzh520/vpp-nat44-nat:v1.0.6
```

## 技术原理

### L2 Xconnect vs L3 路由

**L2 Xconnect 模式（v1.0.5 及之前）**:
```
memif_inside → l2-input → l2-xconnect → l2-output → memif_outside
                  ↑
                  └─ 数据包在 L2 层直接转发，绕过 L3 路由
                  └─ NAT 插件注册在 ip4-unicast（L3），无法被触发
```

**L3 路由模式（v1.0.6）**:
```
memif_inside → ip4-input → ip4-lookup → ip4-rewrite → memif_outside
                              ↑
                              └─ ip4-unicast feature arc
                              └─ NAT 插件在此节点被触发
                              └─ 查询 FIB 路由表，创建 NAT 会话
```

### VPP 数据包处理路径

**L2 模式**:
- 数据包处理路径：`ethernet-input → l2-input → acl-plugin-in-ip4-l2 → l2-xconnect`
- ACL 可以工作：因为 ACL 同时注册了 L2 和 L3 feature arc
- NAT 无法工作：NAT 只注册 L3 feature arc（`ip4-unicast`）

**L3 模式**:
- 数据包处理路径：`ethernet-input → ip4-input → ip4-lookup → nat44-ed-in2out → ip4-rewrite`
- NAT 正常工作：数据包经过 `ip4-unicast` feature arc，触发 NAT 转换
- 需要 FIB：L3 路由需要查询转发信息库（Forwarding Information Base）

## 预期效果

### v1.0.5（L2 Xconnect）

```bash
$ vppctl show interface address
memif1013904223/0 (up):
  L2 xconnect memif1196435762/0    ← L2 转发
memif1196435762/0 (up):
  L2 xconnect memif1013904223/0    ← L2 转发

$ vppctl show nat44 sessions
NAT44 ED sessions:
-------- thread 0 vpp_main: 0 sessions --------   ← 会话数 = 0 ❌
```

### v1.0.6（L3 路由）预期结果

```bash
$ vppctl show interface address
memif1013904223/0 (up):
  L3 10.60.1.1/24                  ← L3 IP 地址
memif1196435762/0 (up):
  L3 10.60.2.1/24                  ← L3 IP 地址

$ vppctl show ip fib
ipv4-VRF:0, fib_index:0, flow hash:[...] epoch:0 flags:none locks:[...]
  10.60.1.0/24                      ← 路由条目
    unicast-ip4-chain
      [@0]: dpo-load-balance: [proto:ip4 index:10 buckets:1 uRPF:9 to:[0:0]]
        [0] [@2]: dpo-receive: 10.60.1.0/24 on memif1013904223/0
  10.60.2.0/24                      ← 路由条目
    unicast-ip4-chain
      [@0]: dpo-load-balance: [proto:ip4 index:11 buckets:1 uRPF:10 to:[0:0]]
        [0] [@2]: dpo-receive: 10.60.2.0/24 on memif1196435762/0

$ vppctl show nat44 sessions
NAT44 ED sessions:
-------- thread 0 vpp_main: X sessions --------   ← 会话数 > 0 ✅
in2out 10.60.1.2:12345 192.168.1.100:12345 fib 0 proto 6 port 80 ...
```

## 验证步骤

1. **部署 v1.0.6 镜像**:
   ```bash
   kubectl apply -f samenode-nat/nse-nat/nat.yaml
   kubectl wait --for=condition=ready pod -l app=nse-nat-vpp -n ns-nse-composition --timeout=60s
   ```

2. **查看接口地址配置**:
   ```bash
   kubectl exec -n ns-nse-composition <NAT_POD> -- vppctl show interface address
   ```
   预期：应该看到 L3 IP 地址（10.60.x.x/24），而不是 "L2 xconnect"

3. **查看路由表**:
   ```bash
   kubectl exec -n ns-nse-composition <NAT_POD> -- vppctl show ip fib
   ```
   预期：应该看到路由条目

4. **测试连通性并检查 NAT 会话**:
   ```bash
   # 从 NSC 发送流量
   kubectl exec -n ns-nse-composition <NSC_POD> -- ping -c 5 <目标IP>

   # 检查 NAT 会话
   kubectl exec -n ns-nse-composition <NAT_POD> -- vppctl show nat44 sessions
   ```
   预期：NAT 会话数应该 > 0

## 已知问题

### 客户端链中 NAT 配置顺序

当前实现中，客户端链的顺序是：
```
up → nat → ipaddress → routes → memif
```

理论上更合理的顺序应该是：
```
up → memif → ipaddress → routes → nat
```

原因：
- 先配置接口（up → memif）
- 再配置 IP 地址（ipaddress）
- 再配置路由表（routes）
- 最后配置 NAT（nat）

如果 v1.0.6 测试发现问题，可能需要调整这个顺序。

## 参考资料

- `.claude/vpp-acl-nat-xconnect-research.md` - ACL vs NAT 在 xconnect 环境下的工作机制差异研究
- `cmd-nse-vl3-vpp` - NSM L3 VPP 实现参考
- [VPP NAT44 ED Plugin](https://fd.io/docs/vpp/master/nat44.html)
- [VPP Feature Arcs](https://fd.io/docs/vpp/master/plugindoc/features.html)

## 升级路径

从 v1.0.5 升级到 v1.0.6：
```bash
# 1. 删除旧版本
kubectl delete -f samenode-nat/nse-nat/nat.yaml

# 2. 部署新版本
kubectl apply -f samenode-nat/nse-nat/nat.yaml

# 3. 验证部署
kubectl get pods -n ns-nse-composition -l app=nse-nat-vpp
kubectl logs -n ns-nse-composition <NAT_POD> | grep "NAT44 ED 插件已启用"
```

## 回滚方案

如果 v1.0.6 出现问题，可以回滚到 v1.0.5：
```bash
# 1. 修改 nat.yaml
sed -i 's/v1.0.6/v1.0.5/g' samenode-nat/nse-nat/nat.yaml

# 2. 重新部署
kubectl delete -f samenode-nat/nse-nat/nat.yaml
kubectl apply -f samenode-nat/nse-nat/nat.yaml
```

## 下一步计划

- [ ] 测试 v1.0.6 的 NAT 会话创建功能
- [ ] 如果当前客户端链顺序有问题，调整为：up → memif → ipaddress → routes → nat
- [ ] 验证 NAT 地址池配置（当前硬编码为 192.168.1.100）
- [ ] 性能测试和优化
