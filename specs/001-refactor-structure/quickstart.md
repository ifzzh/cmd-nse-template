# 快速开始：项目结构重构

**日期**: 2025-11-11
**目标**: 为开发者提供重构实施的快速参考指南

---

## 概述

本指南面向即将执行重构的开发者，提供分步实施指引。重构将380行main.go拆分为5个模块，采用渐进式提取策略，确保每一步都可验证和回滚。

**预计时间**: 8-12小时（包括编码、测试、文档）

---

## 前置条件

### 环境要求
- [x] Go 1.23.8已安装
- [x] Docker已安装（用于测试）
- [x] golangci-lint已安装（用于代码检查）
- [x] 工作目录为仓库根目录

### 工具验证
```bash
# 验证Go版本
go version  # 应显示go1.23.8

# 验证Docker
docker --version

# 验证golangci-lint
golangci-lint --version

# 验证当前目录
pwd  # 应为 /home/ifzzh/Project/NSE-Frame/cmd-nse-firewall-vpp
```

### 分支切换
```bash
# 确认当前在正确的分支
git branch --show-current  # 应显示 001-refactor-structure

# 如果不在正确的分支，切换到该分支
git checkout 001-refactor-structure
```

### 代码备份
```bash
# 备份main.go（用于对比）
cp main.go main.go.backup

# 查看当前main.go行数
wc -l main.go  # 应显示 380 main.go
```

---

## 重构流程总览（6个阶段）

```
阶段1: 提取config模块 (1-2小时)
  → internal/config.go (60行)

阶段2: 提取vppconn模块 (1-2小时)
  → internal/vppconn.go (50行)

阶段3: 提取server模块 (1-2小时)
  → internal/server.go (60行)

阶段4: 提取endpoint模块 (1-2小时)
  → internal/endpoint.go (80行)

阶段5: 提取registry模块 (1-2小时)
  → internal/registry.go (50行)

阶段6: 重构main.go (2小时)
  → main.go (100行以内)

总计: 约8-12小时
```

---

## 阶段1：提取config模块

### 目标
将配置管理逻辑（Config结构体、环境变量解析、ACL规则加载）提取到`internal/config.go`。

### 实施步骤

#### 1.1 创建配置模块文件
```bash
# 创建文件
touch internal/config.go
```

#### 1.2 编写config.go内容
打开`internal/config.go`，编写以下内容：

**提示**: 完整代码见[contracts/internal-modules.md](./contracts/internal-modules.md#模块1config配置管理)

**核心要点**:
- 包名使用`package internal`（不是`package config`）
- 复制main.go中的Config结构体（行87-103）
- 实现LoadConfig函数（封装行106-164逻辑）
- 实现retrieveACLRules函数（行356-379，保持私有）
- 所有注释改为中文

**代码结构预览**:
```go
package internal

import (
    "context"
    "net/url"
    "os"
    "path/filepath"
    "time"

    "github.com/kelseyhightower/envconfig"
    "github.com/networkservicemesh/govpp/binapi/acl_types"
    "github.com/networkservicemesh/sdk/pkg/tools/log"
    "github.com/pkg/errors"
    "gopkg.in/yaml.v2"
)

// Config 表示应用的完整配置
type Config struct {
    // ... 字段与main.go保持一致
}

// LoadConfig 从环境变量加载配置并解析ACL规则
func LoadConfig(ctx context.Context) (*Config, error) {
    // ... 实现逻辑
}

// retrieveACLRules 从配置文件加载ACL规则（私有函数）
func retrieveACLRules(ctx context.Context, c *Config) {
    // ... 实现逻辑
}
```

#### 1.3 修改main.go使用新模块
在main.go中：
1. 删除Config结构体定义（行87-114）
2. 删除(c *Config) Process()方法（行106-114）
3. 删除retrieveACLRules函数（行356-379）
4. 修改配置加载代码：

**修改前**（行148-164）:
```go
config := new(Config)
if err := config.Process(); err != nil {
    logrus.Fatal(err.Error())
}
l, err := logrus.ParseLevel(config.LogLevel)
// ...
config.retrieveACLRules(ctx)
```

**修改后**:
```go
config, err := internal.LoadConfig(ctx)
if err != nil {
    logrus.Fatal(err.Error())
}
l, err := logrus.ParseLevel(config.LogLevel)
// ... (其余保持不变)
```

#### 1.4 验证构建
```bash
# 构建项目
go build ./...

# 应成功构建，无错误
```

#### 1.5 验证功能（可选）
```bash
# 构建Docker镜像（debug目标）
docker build --target debug -t firewall-debug .

# 运行容器验证配置加载
docker run --rm --privileged firewall-debug

# 观察日志输出，确认"executing phase 1: get config from environment"正常
```

#### 1.6 提交变更
```bash
# 查看变更
git diff

# 添加文件
git add internal/config.go main.go

# 提交（使用中文commit message）
git commit -m "重构: 提取config模块到internal/config.go

- 将Config结构体从main.go迁移到internal/config.go
- 实现LoadConfig函数封装环境变量解析逻辑
- retrieveACLRules作为私有函数保留在config模块
- main.go精简约60行
- 功能等价性验证通过"
```

### 验收标准
- [x] internal/config.go文件创建，约60行
- [x] main.go中Config相关代码已删除
- [x] `go build ./...`构建成功
- [x] 配置加载功能正常（通过日志验证）
- [x] Git提交已完成

---

## 阶段2：提取vppconn模块

### 目标
将VPP连接管理逻辑封装到`internal/vppconn.go`，提供VPPManager用于生命周期管理。

### 实施步骤

#### 2.1 创建vppconn模块文件
```bash
touch internal/vppconn.go
```

#### 2.2 编写vppconn.go内容
**提示**: 完整代码见[contracts/internal-modules.md](./contracts/internal-modules.md#模块2vppconnvpp连接管理)

**核心要点**:
- 定义VPPManager结构体（封装conn、errCh、cancel）
- 实现StartVPP函数（封装vpphelper.StartAndDialContext）
- 实现GetConnection和GetErrorChannel方法
- 封装exitOnErr逻辑到内部监控goroutine

**代码结构预览**:
```go
package internal

import (
    "context"
    "github.com/networkservicemesh/vpphelper"
    "github.com/networkservicemesh/sdk/pkg/tools/log"
    "go.fd.io/govpp/api"
)

// VPPManager 管理VPP连接的生命周期
type VPPManager struct {
    conn   api.Connection
    errCh  <-chan error
    cancel context.CancelFunc
}

// StartVPP 启动VPP并建立连接
func StartVPP(ctx context.Context) (*VPPManager, error) {
    // ... 实现逻辑
}

// GetConnection 返回VPP连接对象
func (m *VPPManager) GetConnection() api.Connection {
    return m.conn
}

// GetErrorChannel 返回错误通道
func (m *VPPManager) GetErrorChannel() <-chan error {
    return m.errCh
}

// monitorErrors 监控VPP错误（私有方法）
func (m *VPPManager) monitorErrors(ctx context.Context) {
    // ... 实现逻辑
}
```

#### 2.3 修改main.go使用新模块
**修改前**（行229-230）:
```go
vppConn, vppErrCh := vpphelper.StartAndDialContext(ctx)
exitOnErr(ctx, cancel, vppErrCh)
```

**修改后**:
```go
vppManager, err := internal.StartVPP(ctx)
if err != nil {
    log.FromContext(ctx).Fatalf("VPP启动失败: %+v", err)
}
vppErrCh := vppManager.GetErrorChannel()
exitOnErr(ctx, cancel, vppErrCh)
```

**注意**: 需要将后续使用vppConn的地方改为`vppManager.GetConnection()`

#### 2.4 验证构建和功能
```bash
go build ./...
docker build --target test .
```

#### 2.5 提交变更
```bash
git add internal/vppconn.go main.go
git commit -m "重构: 提取vppconn模块到internal/vppconn.go

- 创建VPPManager封装VPP连接和错误通道
- 实现StartVPP函数启动并连接VPP
- 封装exitOnErr逻辑到VPPManager内部监控
- main.go精简约10行
- VPP连接功能验证通过"
```

### 验收标准
- [x] internal/vppconn.go文件创建，约50行
- [x] main.go使用VPPManager代替直接调用vpphelper
- [x] 构建和测试通过
- [x] Git提交已完成

---

## 阶段3：提取server模块

### 目标
将gRPC服务器创建、TLS配置、启动逻辑封装到`internal/server.go`。

### 实施步骤

#### 3.1 创建server模块文件
```bash
touch internal/server.go
```

#### 3.2 编写server.go内容
**提示**: 完整代码见[contracts/internal-modules.md](./contracts/internal-modules.md#模块3servergRPC服务器管理)

**核心要点**:
- 定义GRPCServer结构体（封装server、listenOn、tmpDir）
- 实现NewGRPCServer函数（创建临时目录和gRPC服务器）
- 实现GetServer、GetListenURL、ListenAndServe、Cleanup方法

#### 3.3 修改main.go使用新模块
**修改前**（行268-288）:
```go
server := grpc.NewServer(...)
firewallEndpoint.Register(server)
tmpDir, err := os.MkdirTemp("", config.Name)
// ...
listenOn := &(url.URL{...})
srvErrCh := grpcutils.ListenAndServe(ctx, listenOn, server)
exitOnErr(ctx, cancel, srvErrCh)
```

**修改后**:
```go
grpcServer, err := internal.NewGRPCServer(ctx, config, tlsServerConfig)
if err != nil {
    log.FromContext(ctx).Fatalf("gRPC服务器创建失败: %+v", err)
}
defer grpcServer.Cleanup()

firewallEndpoint.Register(grpcServer.GetServer())

srvErrCh := grpcServer.ListenAndServe(ctx)
exitOnErr(ctx, cancel, srvErrCh)
```

#### 3.4 验证和提交
```bash
go build ./...
git add internal/server.go main.go
git commit -m "重构: 提取server模块到internal/server.go"
```

### 验收标准
- [x] internal/server.go文件创建，约60行
- [x] main.go精简约20行
- [x] gRPC服务器启动功能正常
- [x] Git提交已完成

---

## 阶段4：提取endpoint模块

### 目标
将NSM端点创建逻辑（60+行的chain构建）封装到`internal/endpoint.go`。

### 实施步骤

#### 4.1 创建endpoint模块文件
```bash
touch internal/endpoint.go
```

#### 4.2 编写endpoint.go内容
**提示**: 完整代码见[contracts/internal-modules.md](./contracts/internal-modules.md#模块4endpoint端点构建)

**核心要点**:
- 定义FirewallEndpoint结构体（嵌入endpoint.Endpoint）
- 实现NewFirewallEndpoint函数（封装60+行的chain构建逻辑）
- 实现Register方法

#### 4.3 修改main.go使用新模块
**修改前**（行232-264）:
```go
firewallEndpoint := new(struct{ endpoint.Endpoint })
firewallEndpoint.Endpoint = endpoint.NewServer(ctx,
    spiffejwt.TokenGeneratorFunc(source, config.MaxTokenLifetime),
    endpoint.WithName(config.Name),
    // ... 60+行的chain配置
)
```

**修改后**:
```go
firewallEndpoint := internal.NewFirewallEndpoint(
    ctx,
    config,
    vppManager.GetConnection(),
    source,
    clientOptions,
)
```

#### 4.4 验证和提交
```bash
go build ./...
git add internal/endpoint.go main.go
git commit -m "重构: 提取endpoint模块到internal/endpoint.go"
```

### 验收标准
- [x] internal/endpoint.go文件创建，约80行
- [x] main.go精简约60行
- [x] 端点创建和注册功能正常
- [x] Git提交已完成

---

## 阶段5：提取registry模块

### 目标
将NSM注册客户端创建和端点注册逻辑封装到`internal/registry.go`。

### 实施步骤

#### 5.1 创建registry模块文件
```bash
touch internal/registry.go
```

#### 5.2 编写registry.go内容
**提示**: 完整代码见[contracts/internal-modules.md](./contracts/internal-modules.md#模块5registrynsm注册服务)

**核心要点**:
- 实现NewRegistryClient函数（创建注册客户端）
- 实现RegisterEndpoint函数（注册端点到NSM）

#### 5.3 修改main.go使用新模块
**修改前**（行295-319）:
```go
nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(...)
nse, err := nseRegistryClient.Register(ctx, &registryapi.NetworkServiceEndpoint{...})
```

**修改后**:
```go
nseRegistryClient := internal.NewRegistryClient(ctx, config, clientOptions)
nse, err := internal.RegisterEndpoint(ctx, nseRegistryClient, config, grpcServer.GetListenURL())
```

#### 5.4 验证和提交
```bash
go build ./...
git add internal/registry.go main.go
git commit -m "重构: 提取registry模块到internal/registry.go"
```

### 验收标准
- [x] internal/registry.go文件创建，约50行
- [x] main.go精简约20行
- [x] 端点注册功能正常
- [x] Git提交已完成

---

## 阶段6：重构main.go

### 目标
将main.go精简至100行以内，仅负责应用生命周期管理和模块协调。

### 实施步骤

#### 6.1 重组main.go逻辑
此时main.go应该已经非常简洁，主要结构如下：

```go
func main() {
    // Phase 1: 设置context和日志
    ctx, cancel := notifyContext()
    defer cancel()
    log.EnableTracing(true)
    logrus.SetFormatter(&nested.Formatter{})
    ctx = log.WithLog(ctx, logruslogger.New(ctx, map[string]interface{}{"cmd": os.Args[0]}))

    // Phase 2: 加载配置
    config, err := internal.LoadConfig(ctx)
    if err != nil {
        logrus.Fatal(err.Error())
    }
    // 设置日志级别...

    // Phase 3: 配置OpenTelemetry和pprof
    // ...

    // Phase 4: 获取SPIFFE SVID
    source, err := workloadapi.NewX509Source(ctx)
    // ...

    // Phase 5: 创建gRPC客户端选项
    clientOptions := append(...)

    // Phase 6: 启动VPP
    vppManager, _ := internal.StartVPP(ctx)
    vppErrCh := vppManager.GetErrorChannel()
    exitOnErr(ctx, cancel, vppErrCh)

    // Phase 7: 创建端点
    firewallEndpoint := internal.NewFirewallEndpoint(ctx, config, vppManager.GetConnection(), source, clientOptions)

    // Phase 8: 创建并启动gRPC服务器
    grpcServer, _ := internal.NewGRPCServer(ctx, config, tlsServerConfig)
    defer grpcServer.Cleanup()
    firewallEndpoint.Register(grpcServer.GetServer())
    srvErrCh := grpcServer.ListenAndServe(ctx)
    exitOnErr(ctx, cancel, srvErrCh)

    // Phase 9: 注册到NSM
    nseRegistryClient := internal.NewRegistryClient(ctx, config, clientOptions)
    nse, _ := internal.RegisterEndpoint(ctx, nseRegistryClient, config, grpcServer.GetListenURL())
    logrus.Infof("端点注册成功: %+v", nse)

    // Phase 10: 等待退出
    log.FromContext(ctx).Infof("启动完成，耗时 %v", time.Since(starttime))
    <-ctx.Done()
    <-vppErrCh
}

// 保留辅助函数
func exitOnErr(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) { ... }
func notifyContext() (context.Context, context.CancelFunc) { ... }
```

#### 6.2 验证行数
```bash
wc -l main.go
# 应显示 ≤100 main.go
```

#### 6.3 完整功能测试
```bash
# 构建测试
go build ./...

# Docker测试
docker build --target test .

# 对比重构前后日志输出
# 1. 运行重构前的备份
docker run --rm --privileged firewall-backup 2>&1 | tee log.before

# 2. 运行重构后的版本
docker run --rm --privileged firewall-latest 2>&1 | tee log.after

# 3. 对比日志（6个启动阶段应完全一致）
diff log.before log.after
```

#### 6.4 最终提交
```bash
git add main.go
git commit -m "重构: 重构main.go为简洁的应用协调层

- main.go精简至100行以内（从380行减少73%）
- 仅负责应用生命周期管理和模块协调
- 所有业务逻辑已委托到5个internal模块
- 6个启动阶段逻辑保持不变
- 完整功能测试通过"
```

### 验收标准
- [x] main.go行数≤100行
- [x] 6个启动阶段逻辑保持不变
- [x] 完整功能测试通过
- [x] 日志输出与重构前一致
- [x] Git提交已完成

---

## 验证清单

### 代码质量验证
```bash
# 静态检查
golangci-lint run

# 构建检查
go build ./...

# 循环依赖检查
go mod graph | grep "cmd-nse-firewall-vpp/internal"
# 应无循环依赖
```

### 功能验证
```bash
# Docker测试
docker build --target test .

# 运行容器
docker run --rm --privileged $(docker build -q --target debug .)

# 观察日志，验证6个阶段：
# ✅ Phase 1: get config from environment
# ✅ Phase 2: retrieve spiffe svid
# ✅ Phase 3: create grpc client options
# ✅ Phase 4: create firewall network service endpoint
# ✅ Phase 5: create grpc server and register firewall-server
# ✅ Phase 6: register nse with nsm
```

### 成功标准验证
基于[spec.md](./spec.md)中的成功标准：

- [ ] **SC-001**: main.go≤100行（对比380行）
  ```bash
  wc -l main.go  # 应≤100
  ```

- [ ] **SC-002**: 每个模块职责单一明确
  ```bash
  ls -l internal/*.go
  # 应有5个文件：config.go, vppconn.go, server.go, endpoint.go, registry.go
  ```

- [ ] **SC-003**: 通过所有测试
  ```bash
  docker build --target test .
  ```

- [ ] **SC-004**: 注释覆盖率≥80%
  ```bash
  # 检查每个文件的导出函数都有中文文档注释
  grep -E "^// [A-Z]" internal/*.go
  ```

- [ ] **SC-005**: 无循环依赖
  ```bash
  go mod graph | grep "cmd-nse-firewall-vpp/internal"
  ```

- [ ] **SC-006**: 构建时间增加<10%
  ```bash
  time go build  # 对比重构前的时间
  ```

- [ ] **SC-007**: 6个启动阶段逻辑不变
  ```bash
  # 对比日志输出
  diff log.before log.after
  ```

- [ ] **SC-008**: 每个模块理解时间<5分钟
  ```bash
  # 代码审查：每个文件≤100行，职责单一
  ```

---

## 常见问题排查

### 问题1：构建失败 - 找不到internal包
**现象**:
```
package internal is not in GOROOT
```

**解决**:
```bash
# 确认go.mod中module路径正确
grep "^module" go.mod
# 应显示: module github.com/networkservicemesh/cmd-nse-firewall-vpp

# 确认import路径正确
grep "import.*internal" main.go
# 应使用: github.com/networkservicemesh/cmd-nse-firewall-vpp/internal
```

### 问题2：VPP连接失败
**现象**:
```
error getting vppapi connection
```

**解决**:
- 确认Docker运行时使用`--privileged`标志
- 检查VPP是否正确启动（查看容器日志）

### 问题3：配置加载错误
**现象**:
```
cannot process envconfig nsm
```

**解决**:
- 检查环境变量前缀是否正确（NSM_）
- 确认Config结构体的envconfig标签正确

### 问题4：重构后行为不一致
**现象**:
日志输出与重构前不同

**解决**:
1. 对比main.go.backup和当前main.go
2. 检查是否遗漏了某些逻辑
3. 使用git diff查看详细变更
4. 如果问题严重，回滚到上一个提交：
   ```bash
   git reset --hard HEAD~1
   ```

---

## 回滚策略

### 回滚单个模块
```bash
# 查看提交历史
git log --oneline

# 回滚到指定提交（例如回滚endpoint模块）
git revert <commit-hash>

# 或者直接恢复main.go.backup中的逻辑
```

### 完全回滚
```bash
# 恢复到重构前
git reset --hard <重构前的commit>

# 恢复main.go备份
cp main.go.backup main.go
```

---

## 下一步

重构完成后，可以继续以下任务：

### 立即任务
1. **代码审查**: 邀请团队成员审查重构代码
2. **更新文档**: 将README.md改为中文（符合"中文优先原则"）
3. **清理备份**: 删除main.go.backup和log文件

### 后续改进（可选）
1. **添加单元测试**: 为各模块添加单元测试（超出本次重构范围）
2. **性能测试**: 验证重构后性能无退化
3. **文档完善**: 为每个模块编写使用示例

### 提交PR（如果需要）
```bash
# 推送到远程分支
git push origin 001-refactor-structure

# 使用GitHub CLI创建PR
gh pr create --title "重构: 项目结构调整 - 降低耦合度提升可维护性" \
  --body "$(cat specs/001-refactor-structure/spec.md)"
```

---

## 总结

### 重构成果
- ✅ main.go从380行精简至100行以内（73%减少）
- ✅ 5个独立模块，职责明确
- ✅ 功能完全保持不变
- ✅ 代码可读性大幅提升
- ✅ 符合所有宪章原则

### 关键成功因素
1. **渐进式提取**: 每次仅提取一个模块，降低风险
2. **立即验证**: 每个阶段完成后立即构建和测试
3. **小步提交**: 每个模块一个提交，便于问题定位
4. **功能等价性**: 保持与main.go完全一致的行为

### 经验教训
- 重构前先备份main.go
- 使用Git频繁提交，便于回滚
- 对比日志输出验证等价性
- 保持模块接口简单，避免过度抽象

---

**预祝重构成功！如有问题，请参考[research.md](./research.md)和[contracts/internal-modules.md](./contracts/internal-modules.md)获取详细信息。**
