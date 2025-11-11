# 内部模块接口契约

**日期**: 2025-11-11
**范围**: internal/包中各模块的接口定义

---

## 概述

本文档定义重构后internal/包中5个模块的接口契约。这些模块仅供本项目内部使用（不导出到外部），因此接口设计遵循简单性原则，避免过度抽象。

---

## 模块1：config（配置管理）

### 文件位置
`internal/config.go`

### 导出函数

#### LoadConfig

**签名**:
```go
func LoadConfig(ctx context.Context) (*Config, error)
```

**职责**: 从环境变量加载配置并解析ACL规则

**参数**:
- `ctx`: 上下文对象，用于日志记录

**返回值**:
- `*Config`: 完整初始化的配置对象
- `error`: 如果环境变量解析失败，返回错误；ACL文件读取失败不返回错误（仅记录日志）

**行为**:
1. 创建Config结构体实例
2. 调用envconfig.Usage("nsm", config)打印配置说明
3. 调用envconfig.Process("nsm", config)解析环境变量
4. 调用retrieveACLRules(ctx, config)加载ACL规则（失败不返回错误）
5. 返回config对象

**环境变量前缀**: `NSM_`

**错误处理**:
- envconfig.Usage失败 → 返回wrapped error
- envconfig.Process失败 → 返回wrapped error
- ACL文件读取失败 → 记录错误日志，继续运行

**示例**:
```go
config, err := internal.LoadConfig(ctx)
if err != nil {
    log.FromContext(ctx).Fatalf("配置加载失败: %+v", err)
}
log.FromContext(ctx).Infof("配置: %#v", config)
```

**对比main.go**: 与main.go的行106-164完全等价

---

### 导出类型

#### Config

**定义**:
```go
type Config struct {
    Name                   string              `default:"firewall-server" desc:"防火墙服务名称"`
    ListenOn               string              `default:"listen.on.sock" desc:"监听socket路径" split_words:"true"`
    ConnectTo              url.URL             `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"NSM连接地址" split_words:"true"`
    MaxTokenLifetime       time.Duration       `default:"10m" desc:"令牌最大有效期" split_words:"true"`
    RegistryClientPolicies []string            `default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego" desc:"注册客户端策略文件路径" split_words:"true"`
    ServiceName            string              `default:"" desc:"提供的服务名称" split_words:"true"`
    Labels                 map[string]string   `default:"" desc:"端点标签"`
    ACLConfigPath          string              `default:"/etc/firewall/config.yaml" desc:"ACL配置文件路径" split_words:"true"`
    ACLConfig              []acl_types.ACLRule `default:"" desc:"已配置的ACL规则" split_words:"true"`
    LogLevel               string              `default:"INFO" desc:"日志级别" split_words:"true"`
    OpenTelemetryEndpoint  string              `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry收集器端点" split_words:"true"`
    MetricsExportInterval  time.Duration       `default:"10s" desc:"指标导出间隔" split_words:"true"`
    PprofEnabled           bool                `default:"false" desc:"是否启用pprof" split_words:"true"`
    PprofListenOn          string              `default:"localhost:6060" desc:"pprof监听地址" split_words:"true"`
}
```

**不变性**: Config对象加载后应视为只读，不应修改字段

**对比main.go**: 与main.go的行87-103完全一致（除注释改为中文）

---

## 模块2：vppconn（VPP连接管理）

### 文件位置
`internal/vppconn.go`

### 导出函数

#### StartVPP

**签名**:
```go
func StartVPP(ctx context.Context) (*VPPManager, error)
```

**职责**: 启动VPP并建立连接，返回VPPManager用于后续获取连接和错误监控

**参数**:
- `ctx`: 上下文对象，用于控制VPP生命周期

**返回值**:
- `*VPPManager`: VPP连接管理器
- `error`: 理论上不返回错误（VPP错误通过errCh异步传递）

**行为**:
1. 调用vpphelper.StartAndDialContext(ctx)启动VPP并连接
2. 创建VPPManager对象封装conn和errCh
3. 启动goroutine监控VPP错误（exitOnErr逻辑）
4. 返回VPPManager

**错误处理**:
- VPP启动失败 → 错误通过errCh传递，不在函数返回时报错
- 运行时错误 → 通过GetErrorChannel()获取

**示例**:
```go
vppManager, err := internal.StartVPP(ctx)
if err != nil {
    log.FromContext(ctx).Fatalf("VPP启动失败: %+v", err)
}

// 在main()中监听错误
vppErrCh := vppManager.GetErrorChannel()
go func() {
    err := <-vppErrCh
    log.FromContext(ctx).Error(err)
    cancel()
}()
```

**对比main.go**: 封装了main.go的行229-230 + exitOnErr逻辑

---

### 导出类型

#### VPPManager

**定义**:
```go
type VPPManager struct {
    conn   api.Connection      // VPP连接对象（不导出）
    errCh  <-chan error         // 错误通道（不导出）
    cancel context.CancelFunc   // 取消函数（不导出）
}
```

**方法**:

##### GetConnection

**签名**:
```go
func (m *VPPManager) GetConnection() api.Connection
```

**职责**: 返回VPP连接对象，供endpoint和xconnect模块使用

**返回值**: `api.Connection` - VPP API连接

**示例**:
```go
vppConn := vppManager.GetConnection()
endpoint := NewFirewallEndpoint(ctx, config, vppConn, ...)
```

##### GetErrorChannel

**签名**:
```go
func (m *VPPManager) GetErrorChannel() <-chan error
```

**职责**: 返回VPP错误通道，main()监听此通道以处理VPP运行时错误

**返回值**: `<-chan error` - 只读错误通道

**示例**:
```go
vppErrCh := vppManager.GetErrorChannel()
exitOnErr(ctx, cancel, vppErrCh)
```

---

## 模块3：server（gRPC服务器管理）

### 文件位置
`internal/server.go`

### 导出函数

#### NewGRPCServer

**签名**:
```go
func NewGRPCServer(ctx context.Context, config *Config, tlsServerConfig *tls.Config) (*GRPCServer, error)
```

**职责**: 创建gRPC服务器并配置TLS

**参数**:
- `ctx`: 上下文对象
- `config`: 配置对象（获取Name和ListenOn）
- `tlsServerConfig`: TLS服务器配置（SPIFFE mTLS）

**返回值**:
- `*GRPCServer`: gRPC服务器包装器
- `error`: 如果临时目录创建失败，返回错误

**行为**:
1. 创建临时目录用于存放Unix socket
2. 构造listenOn URL（unix:///tmp/.../listen.on.sock）
3. 创建grpc.Server（配置TLS和tracing）
4. 返回GRPCServer对象

**错误处理**:
- 临时目录创建失败 → 返回error

**示例**:
```go
grpcServer, err := internal.NewGRPCServer(ctx, config, tlsServerConfig)
if err != nil {
    log.FromContext(ctx).Fatalf("gRPC服务器创建失败: %+v", err)
}
defer grpcServer.Cleanup()
```

**对比main.go**: 封装了main.go的行268-285

---

### 导出类型

#### GRPCServer

**定义**:
```go
type GRPCServer struct {
    server   *grpc.Server  // gRPC服务器实例（不导出）
    listenOn *url.URL       // 监听地址（不导出）
    tmpDir   string         // 临时目录（不导出）
}
```

**方法**:

##### GetServer

**签名**:
```go
func (s *GRPCServer) GetServer() *grpc.Server
```

**职责**: 返回gRPC服务器实例，供endpoint.Register()使用

**返回值**: `*grpc.Server`

**示例**:
```go
endpoint.Register(grpcServer.GetServer())
```

##### GetListenURL

**签名**:
```go
func (s *GRPCServer) GetListenURL() *url.URL
```

**职责**: 返回监听地址URL，供registry.Register()使用

**返回值**: `*url.URL` - Unix socket URL

**示例**:
```go
listenOn := grpcServer.GetListenURL()
nse, err := registryClient.Register(ctx, config, listenOn)
```

##### ListenAndServe

**签名**:
```go
func (s *GRPCServer) ListenAndServe(ctx context.Context) <-chan error
```

**职责**: 启动gRPC服务器监听

**参数**:
- `ctx`: 上下文对象，用于控制服务器生命周期

**返回值**: `<-chan error` - 错误通道（服务器异常时接收错误）

**示例**:
```go
srvErrCh := grpcServer.ListenAndServe(ctx)
exitOnErr(ctx, cancel, srvErrCh)
```

**对比main.go**: 封装了main.go的行287

##### Cleanup

**签名**:
```go
func (s *GRPCServer) Cleanup() error
```

**职责**: 清理临时目录

**返回值**: `error` - 如果清理失败，返回错误

**示例**:
```go
defer grpcServer.Cleanup()
```

**对比main.go**: 封装了main.go的行284

---

## 模块4：endpoint（端点构建）

### 文件位置
`internal/endpoint.go`

### 导出函数

#### NewFirewallEndpoint

**签名**:
```go
func NewFirewallEndpoint(
    ctx context.Context,
    config *Config,
    vppConn api.Connection,
    source *workloadapi.X509Source,
    clientOptions []grpc.DialOption,
) *FirewallEndpoint
```

**职责**: 构建NSM防火墙端点的完整chain

**参数**:
- `ctx`: 上下文对象
- `config`: 配置对象（获取Name、ACLConfig、Labels等）
- `vppConn`: VPP连接（从VPPManager.GetConnection()获取）
- `source`: SPIFFE X509Source（用于token生成）
- `clientOptions`: gRPC拨号选项（用于connect.NewServer的内部客户端）

**返回值**: `*FirewallEndpoint` - 防火墙端点实例

**行为**:
1. 调用endpoint.NewServer()创建NSM端点
2. 配置授权、recvfd/sendfd、up、clienturl、xconnect、acl等功能
3. 配置memif机制
4. 配置内部客户端（用于connect到上游）
5. 返回FirewallEndpoint封装

**无错误返回**: 构建过程不会失败（NSM SDK保证）

**示例**:
```go
fwEndpoint := internal.NewFirewallEndpoint(
    ctx,
    config,
    vppManager.GetConnection(),
    source,
    clientOptions,
)
fwEndpoint.Register(grpcServer.GetServer())
```

**对比main.go**: 封装了main.go的行232-264

---

### 导出类型

#### FirewallEndpoint

**定义**:
```go
type FirewallEndpoint struct {
    endpoint.Endpoint  // 嵌入NSM端点接口（不导出）
}
```

**方法**:

##### Register

**签名**:
```go
func (e *FirewallEndpoint) Register(server *grpc.Server)
```

**职责**: 将端点注册到gRPC服务器

**参数**:
- `server`: gRPC服务器实例（从GRPCServer.GetServer()获取）

**行为**: 调用endpoint.Endpoint.Register(server)

**示例**:
```go
fwEndpoint.Register(grpcServer.GetServer())
```

**对比main.go**: 封装了main.go的行278

---

## 模块5：registry（NSM注册服务）

### 文件位置
`internal/registry.go`

### 导出函数

#### NewRegistryClient

**签名**:
```go
func NewRegistryClient(
    ctx context.Context,
    config *Config,
    clientOptions []grpc.DialOption,
) registryapi.NetworkServiceEndpointRegistryClient
```

**职责**: 创建NSM注册客户端

**参数**:
- `ctx`: 上下文对象
- `config`: 配置对象（获取ConnectTo、RegistryClientPolicies）
- `clientOptions`: gRPC拨号选项

**返回值**: `registryapi.NetworkServiceEndpointRegistryClient` - NSM注册客户端

**行为**:
1. 调用registryclient.NewNetworkServiceEndpointRegistryClient()
2. 配置连接地址（config.ConnectTo）
3. 配置拨号选项（clientOptions）
4. 配置额外功能（clientinfo、sendfd）
5. 配置授权策略（config.RegistryClientPolicies）
6. 返回客户端实例

**示例**:
```go
nseRegistryClient := internal.NewRegistryClient(ctx, config, clientOptions)
nse, err := nseRegistryClient.Register(ctx, &registryapi.NetworkServiceEndpoint{...})
```

**对比main.go**: 封装了main.go的行295-304

---

#### RegisterEndpoint

**签名**:
```go
func RegisterEndpoint(
    ctx context.Context,
    client registryapi.NetworkServiceEndpointRegistryClient,
    config *Config,
    listenOn *url.URL,
) (*registryapi.NetworkServiceEndpoint, error)
```

**职责**: 向NSM Manager注册端点

**参数**:
- `ctx`: 上下文对象
- `client`: NSM注册客户端（从NewRegistryClient获取）
- `config`: 配置对象（获取Name、ServiceName、Labels）
- `listenOn`: gRPC服务器监听地址（从GRPCServer.GetListenURL()获取）

**返回值**:
- `*registryapi.NetworkServiceEndpoint`: 注册成功的端点信息
- `error`: 如果注册失败，返回错误

**行为**:
1. 构造NetworkServiceEndpoint对象
2. 调用client.Register()
3. 返回注册结果

**错误处理**:
- 注册失败 → 返回error（调用方通过log.Fatal处理）

**示例**:
```go
nse, err := internal.RegisterEndpoint(ctx, nseRegistryClient, config, grpcServer.GetListenURL())
if err != nil {
    log.FromContext(ctx).Fatalf("端点注册失败: %+v", err)
}
logrus.Infof("端点注册成功: %+v", nse)
```

**对比main.go**: 封装了main.go的行305-319

---

## 依赖关系图

```
main.go
  ├─→ config.LoadConfig(ctx)
  ├─→ vppconn.StartVPP(ctx)
  ├─→ server.NewGRPCServer(ctx, config, tlsConfig)
  ├─→ endpoint.NewFirewallEndpoint(ctx, config, vppConn, source, clientOptions)
  └─→ registry.NewRegistryClient(ctx, config, clientOptions)
      └─→ registry.RegisterEndpoint(ctx, client, config, listenOn)

模块间依赖:
  config: 无依赖
  vppconn: 无依赖
  server: 依赖config（获取Name、ListenOn）
  endpoint: 依赖config（ACLConfig）+ vppconn（GetConnection）
  registry: 依赖config（ConnectTo、策略）+ server（GetListenURL）
```

---

## 接口稳定性保证

### 向后兼容承诺
- ✅ 所有导出函数的签名不变
- ✅ Config结构体字段不变
- ✅ 行为与main.go完全等价

### 版本控制
- 本重构不涉及版本号变更（内部模块）
- 如果未来需要修改接口，应先讨论并文档化

### 废弃流程
- 如果需要废弃某个函数，应：
  1. 标记为Deprecated（文档注释）
  2. 提供新的替代函数
  3. 保持旧函数至少一个版本周期

---

## 测试契约

### 单元测试（可选，超出重构范围）
如果未来添加单元测试，建议测试以下功能：
- [ ] LoadConfig：环境变量解析
- [ ] StartVPP：VPP连接建立（需mock vpphelper）
- [ ] NewGRPCServer：临时目录创建
- [ ] NewFirewallEndpoint：chain构建逻辑
- [ ] NewRegistryClient：客户端创建

### 集成测试（现有Docker测试）
- [ ] 完整启动流程（6个阶段）
- [ ] VPP连接正常
- [ ] gRPC服务器可访问
- [ ] 端点注册成功

---

## 总结

### 接口设计原则
1. **简单性**: 每个模块仅提供必要的导出函数，无复杂接口抽象
2. **一致性**: 保持与main.go的行为完全等价
3. **可测试性**: 通过依赖注入支持测试（如传入Config对象）
4. **生命周期清晰**: 所有对象创建一次，生命周期与应用一致

### 与main.go的等价性
- ✅ 所有函数逻辑与main.go完全一致
- ✅ 所有数据结构与main.go完全一致
- ✅ 所有错误处理与main.go完全一致
- ✅ 仅改变代码组织方式，不改变行为

### 符合宪章
- ✅ 中文优先：所有注释使用中文
- ✅ 简洁架构：无复杂抽象层
- ✅ 功能完整：接口覆盖main.go的所有功能
- ✅ 一致性：继承现有设计模式
