# 数据模型：项目结构重构

**日期**: 2025-11-11
**目标**: 定义重构后各模块的核心数据结构和关系

---

## 概述

本重构为**代码组织变更**，不涉及新的数据模型设计。所有数据结构保持与main.go中完全一致，仅从main包迁移到internal包。

---

## 核心实体

### 1. Config（配置对象）

**定义位置**: `internal/config.go`

**职责**: 表示应用的完整配置，包括服务名称、连接URL、ACL规则、日志级别等

**结构**:
```go
type Config struct {
    // 服务配置
    Name                   string              `default:"firewall-server" desc:"防火墙服务名称"`
    ServiceName            string              `default:"" desc:"提供的服务名称" split_words:"true"`
    Labels                 map[string]string   `default:"" desc:"端点标签"`

    // 网络配置
    ListenOn               string              `default:"listen.on.sock" desc:"监听socket路径" split_words:"true"`
    ConnectTo              url.URL             `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"NSM连接地址" split_words:"true"`

    // 安全配置
    MaxTokenLifetime       time.Duration       `default:"10m" desc:"令牌最大有效期" split_words:"true"`
    RegistryClientPolicies []string            `default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego" desc:"注册客户端策略文件路径" split_words:"true"`

    // 防火墙配置
    ACLConfigPath          string              `default:"/etc/firewall/config.yaml" desc:"ACL配置文件路径" split_words:"true"`
    ACLConfig              []acl_types.ACLRule `default:"" desc:"已配置的ACL规则" split_words:"true"`

    // 日志和监控配置
    LogLevel               string              `default:"INFO" desc:"日志级别" split_words:"true"`
    OpenTelemetryEndpoint  string              `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry收集器端点" split_words:"true"`
    MetricsExportInterval  time.Duration       `default:"10s" desc:"指标导出间隔" split_words:"true"`

    // 性能分析配置
    PprofEnabled           bool                `default:"false" desc:"是否启用pprof" split_words:"true"`
    PprofListenOn          string              `default:"localhost:6060" desc:"pprof监听地址" split_words:"true"`
}
```

**字段分类**:
- **服务配置** (3个字段): Name, ServiceName, Labels
- **网络配置** (2个字段): ListenOn, ConnectTo
- **安全配置** (2个字段): MaxTokenLifetime, RegistryClientPolicies
- **防火墙配置** (2个字段): ACLConfigPath, ACLConfig
- **日志和监控配置** (3个字段): LogLevel, OpenTelemetryEndpoint, MetricsExportInterval
- **性能分析配置** (2个字段): PprofEnabled, PprofListenOn

**关系**:
- Config对象在应用启动时创建一次
- 被所有其他模块引用（只读）
- 通过envconfig从环境变量加载

**验证规则**:
- Name不为空（由envconfig默认值保证）
- ConnectTo必须是有效的URL
- ACLConfigPath必须是有效的文件路径
- LogLevel必须是有效的logrus级别（TRACE/DEBUG/INFO/WARN/ERROR/FATAL）

**状态转换**: 无（不可变对象，加载后只读）

---

### 2. VPPManager（VPP连接管理器）

**定义位置**: `internal/vppconn.go`

**职责**: 管理与VPP的连接生命周期，提供连接获取和错误监控接口

**结构**:
```go
type VPPManager struct {
    conn   api.Connection      // VPP连接对象
    errCh  <-chan error         // 错误通道（监听VPP运行时错误）
    cancel context.CancelFunc   // 取消函数（用于错误时停止应用）
}
```

**字段说明**:
- `conn`: VPP API连接，由vpphelper.StartAndDialContext()返回
- `errCh`: 错误通道，VPP运行时错误通过此通道传递
- `cancel`: 上下文取消函数，VPP错误时调用以停止应用

**关系**:
- VPPManager在应用启动时创建一次（Phase 4）
- endpoint和xconnect模块通过GetConnection()获取VPP连接
- main.go通过GetErrorChannel()监听VPP错误

**生命周期**:
1. **创建**: StartVPP(ctx) → VPPManager
2. **使用**: GetConnection() → api.Connection
3. **监控**: GetErrorChannel() → <-chan error
4. **销毁**: ctx.Done() → 自动清理

**错误处理**:
- VPP启动失败 → errCh立即接收错误
- VPP运行时崩溃 → errCh接收错误，cancel()被调用
- 应用正常退出 → ctx.Done()触发，goroutine退出

---

### 3. GRPCServer（gRPC服务器包装器）

**定义位置**: `internal/server.go`

**职责**: 封装gRPC服务器的创建、TLS配置、启动和停止逻辑

**结构**:
```go
type GRPCServer struct {
    server   *grpc.Server     // gRPC服务器实例
    listenOn *url.URL          // 监听地址（Unix socket）
    tmpDir   string            // 临时目录（存放socket文件）
}
```

**字段说明**:
- `server`: gRPC服务器实例，由grpc.NewServer()创建
- `listenOn`: 监听地址（Unix socket URL），格式为unix:///tmp/.../listen.on.sock
- `tmpDir`: 临时目录路径，需要在应用退出时清理

**关系**:
- GRPCServer在应用启动时创建一次（Phase 5）
- FirewallEndpoint通过Register()方法注册到此服务器
- main.go通过ListenAndServe()启动服务器

**生命周期**:
1. **创建**: NewGRPCServer(ctx, config, tlsServerConfig) → GRPCServer
2. **注册服务**: endpoint.Register(server.server)
3. **启动**: server.ListenAndServe(ctx) → <-chan error
4. **停止**: ctx.Done() → server.GracefulStop()
5. **清理**: 删除tmpDir

**TLS配置**:
- 使用SPIFFE mTLS（双向TLS认证）
- tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny())
- 集成grpcfd.TransportCredentials（支持文件描述符传递）

---

### 4. FirewallEndpoint（防火墙端点）

**定义位置**: `internal/endpoint.go`

**职责**: 构建NSM防火墙端点的完整chain，包含ACL、xconnect、memif等组件

**结构**:
```go
// FirewallEndpoint 封装NSM端点
// 注意：这是匿名结构体的包装，与main.go保持一致
type FirewallEndpoint struct {
    endpoint.Endpoint  // 嵌入NSM端点接口
}
```

**关系**:
- FirewallEndpoint在应用启动时创建一次（Phase 4）
- 依赖Config（获取ACL规则、连接配置等）
- 依赖VPPManager（获取VPP连接）
- 依赖SPIFFE source（获取token生成器）
- 被GRPCServer注册（通过Register()方法）

**组件chain（保持与main.go一致）**:
```
endpoint.NewServer(ctx, tokenGenerator,
    endpoint.WithName(config.Name),
    endpoint.WithAuthorizeServer(authorize.NewServer()),
    endpoint.WithAdditionalFunctionality(
        recvfd.NewServer(),
        sendfd.NewServer(),
        up.NewServer(ctx, vppConn),
        clienturl.NewServer(&config.ConnectTo),
        xconnect.NewServer(vppConn),
        acl.NewServer(vppConn, config.ACLConfig),
        mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
            memif.MECHANISM: chain.NewNetworkServiceServer(memif.NewServer(ctx, vppConn)),
        }),
        connect.NewServer(
            client.NewClient(
                ctx,
                client.WithoutRefresh(),
                client.WithName(config.Name),
                client.WithDialOptions(clientOptions...),
                client.WithAdditionalFunctionality(
                    metadata.NewClient(),
                    mechanismtranslation.NewClient(),
                    passthrough.NewClient(config.Labels),
                    up.NewClient(ctx, vppConn),
                    xconnect.NewClient(vppConn),
                    memif.NewClient(ctx, vppConn),
                    sendfd.NewClient(),
                    recvfd.NewClient(),
                )),
        )),
)
```

**特点**:
- 复杂的链式构建（60+行）
- 服务端和客户端混合chain
- 支持memif机制
- 包含ACL和xconnect功能

---

### 5. RegistryClient（NSM注册客户端）

**定义位置**: `internal/registry.go`

**职责**: 管理与NSM Registry的交互，包括注册、注销和策略应用

**结构**:
```go
// RegistryClient 封装NSM注册客户端
type RegistryClient struct {
    client registryapi.NetworkServiceEndpointRegistryClient  // NSM注册客户端
}
```

**关系**:
- RegistryClient在应用启动时创建一次（Phase 6）
- 依赖Config（获取连接地址、策略文件等）
- 依赖clientOptions（gRPC拨号选项）
- 调用Register()方法向NSM Manager注册端点

**注册信息**:
```go
&registryapi.NetworkServiceEndpoint{
    Name:                config.Name,                 // 端点名称
    NetworkServiceNames: []string{config.ServiceName}, // 提供的服务列表
    NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
        config.ServiceName: {
            Labels: config.Labels,  // 端点标签
        },
    },
    Url: listenOn.String(),  // 监听地址（gRPC服务器的socket地址）
}
```

**生命周期**:
1. **创建**: NewRegistryClient(ctx, config, clientOptions) → RegistryClient
2. **注册**: client.Register(ctx, nseInfo) → registryapi.NetworkServiceEndpoint
3. **注销**: ctx.Done() → 自动注销（由NSM SDK处理）

---

## 数据流图

### 应用启动流程（6个阶段）

```
Phase 1: 配置加载
    环境变量 → LoadConfig() → Config对象 → 存储在main()

Phase 2: SPIFFE认证
    workloadapi → GetX509SVID() → SVID → 生成TLS配置

Phase 3: 客户端选项
    TLS配置 + Token生成器 → clientOptions → 用于gRPC拨号

Phase 4: 创建VPP和端点
    StartVPP() → VPPManager → GetConnection() → VPP连接
    NewFirewallEndpoint() → FirewallEndpoint → 使用VPP连接

Phase 5: 启动gRPC服务器
    NewGRPCServer() → GRPCServer
    endpoint.Register(server) → 注册服务
    server.ListenAndServe() → 启动监听

Phase 6: 注册到NSM
    NewRegistryClient() → RegistryClient
    client.Register() → 向NSM Manager注册端点
```

### 模块间依赖关系

```
main.go
  ├─→ config (LoadConfig)
  ├─→ vppconn (StartVPP, GetConnection, GetErrorChannel)
  ├─→ server (NewGRPCServer, ListenAndServe)
  ├─→ endpoint (NewFirewallEndpoint, Register)
  └─→ registry (NewRegistryClient, Register)

模块间依赖:
  config: 无依赖
  vppconn: 无依赖
  server: 依赖config（获取监听地址）
  endpoint: 依赖config（ACL规则）+ vppconn（VPP连接）
  registry: 依赖config（注册信息）+ server（获取listenOn）
```

---

## 状态管理

### 应用级状态

**全局状态（在main()中管理）**:
- `config`: Config对象（不可变，启动时加载一次）
- `vppManager`: VPPManager对象（生命周期与应用一致）
- `grpcServer`: GRPCServer对象（生命周期与应用一致）
- `endpoint`: FirewallEndpoint对象（生命周期与应用一致）
- `registryClient`: RegistryClient对象（生命周期与应用一致）

**上下文状态**:
- `ctx`: context.Context（从notifyContext()获取，监听信号）
- `cancel`: context.CancelFunc（用于主动取消）

**错误通道**:
- `vppErrCh`: VPP错误通道（从VPPManager获取）
- `srvErrCh`: gRPC服务器错误通道（从ListenAndServe获取）

### 并发安全

**只读对象（无需加锁）**:
- Config（加载后不修改）
- VPPManager.conn（连接建立后不修改）

**写保护对象**:
- VPP API调用通过govpp的channel机制保护
- gRPC调用由gRPC框架保护

**注意事项**:
- 不需要显式的锁机制
- 使用context.Context传递取消信号
- 使用channel传递错误

---

## 验证规则

### 配置验证
- [x] Config.Name不为空
- [x] Config.ConnectTo是有效的URL
- [x] Config.ACLConfigPath文件存在（可选，不存在时记录错误）
- [x] Config.LogLevel是有效的日志级别

### 运行时验证
- [x] VPP连接建立成功
- [x] gRPC服务器启动成功
- [x] 端点注册到NSM成功
- [x] 临时目录创建成功

### 错误处理
- **致命错误** (log.Fatal): 配置加载失败、端点创建失败、注册失败
- **可恢复错误** (log.Error): ACL配置文件读取失败（继续运行）
- **运行时错误** (通过errCh): VPP连接断开、gRPC服务器异常

---

## 迁移说明

### 从main.go迁移到internal/

**Config结构体**:
- ✅ 保持所有字段不变
- ✅ 保持所有struct tag不变
- ✅ 注释改为中文

**数据流保持不变**:
- ✅ 配置从环境变量加载（envconfig）
- ✅ ACL规则从YAML文件加载
- ✅ VPP连接通过vpphelper建立
- ✅ gRPC服务器使用SPIFFE mTLS
- ✅ NSM注册使用registry client

**新增的封装**:
- VPPManager（封装VPP连接和错误通道）
- GRPCServer（封装gRPC服务器和临时目录）
- RegistryClient（封装注册客户端）

---

## 总结

### 数据模型特点
1. **不可变性**: Config对象加载后只读
2. **简单性**: 无复杂的状态机或继承关系
3. **依赖注入**: 通过构造函数参数传递依赖
4. **生命周期清晰**: 所有对象与应用同生命周期

### 与main.go的一致性
- ✅ 所有数据结构保持不变
- ✅ 所有字段和类型保持不变
- ✅ 所有依赖关系保持不变
- ✅ 仅添加封装层（VPPManager、GRPCServer、RegistryClient）

### 符合宪章
- ✅ 功能完整性：数据结构零变化
- ✅ 一致性优先：继承现有设计
- ✅ 简洁架构：无复杂的抽象层
