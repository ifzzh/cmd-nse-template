# API: Nat44InterfaceAddDelFeature

**阶段**: P1.2
**用途**: 配置 VPP 接口的 NAT 角色（inside/outside）
**VPP 版本**: VPP v23.10-rc0~170 / v24.10.0

## API 定义

### 请求结构

```go
type Nat44InterfaceAddDelFeature struct {
    IsAdd     bool                           // true = 启用 NAT 功能, false = 禁用 NAT 功能
    Flags     nat_types.NatConfigFlags       // NAT_IS_INSIDE(32) 或 NAT_IS_OUTSIDE(16)
    SwIfIndex interface_types.InterfaceIndex // VPP 接口索引
}
```

### 字段说明

| 字段名 | 类型 | 必填 | 描述 | 有效值 |
|--------|------|------|------|--------|
| IsAdd | bool | ✅ | 启用或禁用 NAT 功能 | true（启用）、false（禁用） |
| Flags | nat_types.NatConfigFlags | ✅ | NAT 接口角色标志 | NAT_IS_INSIDE(32)、NAT_IS_OUTSIDE(16) |
| SwIfIndex | interface_types.InterfaceIndex | ✅ | VPP 接口索引 | 由 NSM 创建接口时分配，通过 ifindex.Load 获取 |

### 响应结构

```go
type Nat44InterfaceAddDelFeatureReply struct {
    Retval int32 // 返回值：0 = 成功，负数 = 错误码
}
```

## 使用示例

### P1.2 实现

```go
// configureNATInterface 配置 NAT 接口角色（inside/outside）
func configureNATInterface(ctx context.Context, vppConn *govppapi.Connection, swIfIndex interface_types.InterfaceIndex, role NATInterfaceRole) error {
    logger := log.FromContext(ctx).WithFields(logrus.Fields{
        "nat_server": "configure",
        "swIfIndex":  swIfIndex,
        "role":       role,
    })

    // 1. 创建 API 客户端
    client := nat44_ed.NewServiceClient(vppConn)

    // 2. 确定 NAT 角色标志
    var flags nat_types.NatConfigFlags
    if role == NATRoleInside {
        flags = nat_types.NAT_IS_INSIDE
    } else {
        flags = nat_types.NAT_IS_OUTSIDE
    }

    // 3. 创建请求
    req := &nat44_ed.Nat44InterfaceAddDelFeature{
        IsAdd:     true,
        Flags:     flags,
        SwIfIndex: swIfIndex,
    }

    // 4. 调用 VPP API
    reply := &nat44_ed.Nat44InterfaceAddDelFeatureReply{}
    if err := vppConn.Invoke(ctx, req, reply); err != nil {
        logger.WithError(err).Error("VPP API 调用失败")
        return errors.Wrap(err, "VPP API Nat44InterfaceAddDelFeature 调用失败")
    }

    // 5. 检查返回值
    if reply.Retval != 0 {
        logger.WithField("retval", reply.Retval).Error("VPP API 返回错误")
        return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
    }

    logger.Info(fmt.Sprintf("NAT %s 接口配置成功", role))
    return nil
}
```

### Request() 方法集成

```go
func (n *natServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
    postponeCtxFunc := postpone.ContextWithValues(ctx)

    // 1. 调用下一个服务器（创建 VPP 接口）
    conn, err := next.Server(ctx).Request(ctx, request)
    if err != nil {
        return nil, err
    }

    // 2. 获取接口索引
    isClient := metadata.IsClient(n)
    swIfIndex, ok := ifindex.Load(ctx, isClient)
    if !ok {
        closeCtx, cancelClose := postponeCtxFunc()
        defer cancelClose()
        if _, closeErr := n.Close(closeCtx, conn); closeErr != nil {
            err = errors.Wrapf(err, "连接关闭时发生错误: %s", closeErr.Error())
        }
        return nil, errors.New("未找到接口索引")
    }

    // 3. 确定接口角色
    var role NATInterfaceRole
    if isClient {
        role = NATRoleOutside // client 端连接下游 NSE，配置为 outside
    } else {
        role = NATRoleInside // server 端连接 NSC，配置为 inside
    }

    // 4. 配置 NAT 接口
    if err := configureNATInterface(ctx, n.vppConn, swIfIndex, role); err != nil {
        closeCtx, cancelClose := postponeCtxFunc()
        defer cancelClose()
        if _, closeErr := n.Close(closeCtx, conn); closeErr != nil {
            err = errors.Wrapf(err, "连接关闭时发生错误: %s", closeErr.Error())
        }
        return nil, err
    }

    // 5. 记录接口状态
    n.interfaceStates.Store(conn.GetId(), &natInterfaceState{
        swIfIndex:  swIfIndex,
        role:       role,
        configured: true,
    })

    return conn, nil
}
```

### Close() 方法集成

```go
func (n *natServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
    // 1. 获取接口状态
    state, loaded := n.interfaceStates.LoadAndDelete(conn.GetId())
    if loaded && state.configured {
        // 2. 禁用 NAT 接口（IsAdd = false）
        if err := disableNATInterface(ctx, n.vppConn, state.swIfIndex, state.role); err != nil {
            log.FromContext(ctx).WithError(err).Error("NAT 接口禁用失败")
            // 继续执行清理流程，不阻断
        }
    }

    // 3. 调用下一个服务器
    return next.Server(ctx).Close(ctx, conn)
}

// disableNATInterface 禁用 NAT 接口功能
func disableNATInterface(ctx context.Context, vppConn *govppapi.Connection, swIfIndex interface_types.InterfaceIndex, role NATInterfaceRole) error {
    logger := log.FromContext(ctx).WithFields(logrus.Fields{
        "nat_server": "cleanup",
        "swIfIndex":  swIfIndex,
        "role":       role,
    })

    client := nat44_ed.NewServiceClient(vppConn)

    var flags nat_types.NatConfigFlags
    if role == NATRoleInside {
        flags = nat_types.NAT_IS_INSIDE
    } else {
        flags = nat_types.NAT_IS_OUTSIDE
    }

    req := &nat44_ed.Nat44InterfaceAddDelFeature{
        IsAdd:     false, // 禁用 NAT 功能
        Flags:     flags,
        SwIfIndex: swIfIndex,
    }

    reply := &nat44_ed.Nat44InterfaceAddDelFeatureReply{}
    if err := vppConn.Invoke(ctx, req, reply); err != nil {
        logger.WithError(err).Error("VPP API 调用失败")
        return errors.Wrap(err, "VPP API Nat44InterfaceAddDelFeature 调用失败")
    }

    if reply.Retval != 0 {
        logger.WithField("retval", reply.Retval).Error("VPP API 返回错误")
        return fmt.Errorf("VPP API 返回错误: %d", reply.Retval)
    }

    logger.Info(fmt.Sprintf("NAT %s 接口禁用成功", role))
    return nil
}
```

## 错误处理

### 常见错误码

| 错误码 | 含义 | 可能原因 | 解决方法 |
|--------|------|----------|----------|
| 0 | 成功 | - | - |
| -1 | 通用错误 | 接口索引无效、VPP 内部错误 | 检查接口是否存在，查看 VPP 日志 |
| -2 | 资源不存在 | 接口不存在或已被删除 | 确认接口已创建，检查 ifindex |
| -3 | 资源已存在 | 接口已配置为 NAT 接口 | 避免重复调用，检查接口状态 |

### 错误日志示例

```
错误: NAT inside 接口配置失败: VPP API Nat44InterfaceAddDelFeature 调用失败: rpc error: code = Unknown desc = invalid argument
```

## VPP CLI 验证

### 验证命令

```bash
vppctl show nat44 interfaces
```

### 预期输出

```
NAT44 interfaces:
  memif0/0 in   (inside 接口)
  memif0/1 out  (outside 接口)
```

### 异常输出示例

```
NAT44 interfaces:
  (无输出 - 表示未配置 NAT 接口)
```

## 测试用例

### 正常场景

```go
// 测试 NAT inside 接口配置成功
func TestConfigureNATInterface_Inside_Success(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    swIfIndex := interface_types.InterfaceIndex(1)
    err := configureNATInterface(ctx, vppConn, swIfIndex, NATRoleInside)
    assert.NoError(t, err)

    // 验证 VPP 配置
    output := execVPPCLI(t, "show nat44 interfaces")
    assert.Contains(t, output, "in")
}
```

### 边界场景

```go
// 测试重复配置 NAT 接口
func TestConfigureNATInterface_Duplicate(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    swIfIndex := interface_types.InterfaceIndex(1)

    // 第一次配置成功
    err := configureNATInterface(ctx, vppConn, swIfIndex, NATRoleInside)
    assert.NoError(t, err)

    // 第二次配置可能返回错误（取决于 VPP 实现）
    err = configureNATInterface(ctx, vppConn, swIfIndex, NATRoleInside)
    // VPP 可能允许重复配置或返回错误，根据实际行为调整
}
```

### 错误场景

```go
// 测试无效接口索引
func TestConfigureNATInterface_InvalidSwIfIndex(t *testing.T) {
    ctx := context.Background()
    vppConn := setupVPPConnection(t)
    defer vppConn.Disconnect()

    invalidSwIfIndex := interface_types.InterfaceIndex(9999)
    err := configureNATInterface(ctx, vppConn, invalidSwIfIndex, NATRoleInside)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "VPP API 返回错误")
}
```

## 依赖

- **binapi 模块**: nat44_ed, nat_types, interface_types
- **前置条件**: VPP 接口已创建（由 NSM 创建 memif/kernel 接口）
- **后续操作**: P1.3 配置地址池（Nat44AddDelAddressRange）

## 参考

- VPP 官方文档: https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- govpp binapi: `internal/binapi_nat44_ed/nat44_ed.ba.go` (P3.2 本地化后)
- Data Model: [data-model.md](../data-model.md) - VPP 接口实体定义

---

**更新记录**:
- 2025-01-15: 初始版本
