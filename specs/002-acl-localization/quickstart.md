# Quick Start: ACL 模块本地化

**Feature**: 002-acl-localization | **Date**: 2025-11-13 (更新)

本指南提供 govpp ACL binapi 模块本地化的详细操作步骤，确保每次本地化一个模块，严格版本控制，最小化修改。

**关键更新**: 根据 2025-11-13 的目录结构澄清，binapi 模块直接与 `internal/acl/` 并列放置，统一使用 `binapi_` 前缀命名。

---

## 前置条件

### 环境检查
```bash
# 1. 验证 Go 版本
go version  # 应为 go1.23.8 或更高

# 2. 验证 Docker 可用
docker --version

# 3. 验证 Git 状态
git status  # 应为干净工作目录

# 4. 验证基线镜像存在
docker pull ifzzh520/vpp-acl-firewall:v1.0.0
```

### 依赖确认
```bash
# 确认 go.sum 中的 govpp 版本
grep "github.com/networkservicemesh/govpp" go.sum
# 期望输出包含: v0.0.0-20240328101142-8a444680fbba h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
```

---

## 迭代 1: 本地化 binapi/acl_types

### 步骤 1: 下载 govpp 模块到缓存

```bash
# 确保 govpp 模块已下载
go mod download github.com/networkservicemesh/govpp

# 定位缓存目录
GOVPP_CACHE=$(go env GOPATH)/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba

# 验证缓存目录存在
ls -la $GOVPP_CACHE/binapi/acl_types/
# 期望看到: acl_types.ba.go
```

### 步骤 2: 复制模块到本地（并列目录）

```bash
# 创建本地模块目录（与 internal/acl/ 并列）
mkdir -p internal/binapi_acl_types

# 复制源码
cp -r $GOVPP_CACHE/binapi/acl_types/* internal/binapi_acl_types/

# 取消只读权限
chmod -R u+w internal/binapi_acl_types/

# 验证文件已复制
ls -la internal/binapi_acl_types/
# 期望看到: acl_types.ba.go
```

### 步骤 3: 创建 go.mod 文件

在 `internal/binapi_acl_types/` 目录下创建 `go.mod`:

```go
module github.com/ifzzh/cmd-nse-template/internal/binapi_acl_types

go 1.23

require (
    go.fd.io/govpp v0.11.0
)
```

```bash
# 执行 go mod tidy 自动补全依赖
cd internal/binapi_acl_types/
go mod tidy
cd ../..
```

### 步骤 4: 添加包级别中文注释（可选）

编辑 `internal/binapi_acl_types/acl_types.ba.go`，在文件开头添加：

```go
// Package acl_types 提供 VPP ACL 类型定义的 Go 语言绑定。
//
// 本模块由 GoVPP binapi-generator 自动生成，对应 VPP API 版本。
// 原始仓库: github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba
// 本地化日期: 2025-11-13
// 版本哈希: h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
//
// 注意: 本代码为自动生成，请勿手动修改数据结构和接口定义。
```

### 步骤 5: 创建 README.md

在 `internal/binapi_acl_types/` 目录下创建 `README.md`:

```markdown
# govpp binapi/acl_types 本地化模块

## 来源信息
- **原始仓库**: github.com/networkservicemesh/govpp
- **版本**: v0.0.0-20240328101142-8a444680fbba
- **Commit Hash**: 8a444680fbba
- **go.sum Hash**: h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
- **本地化日期**: 2025-11-13

## 模块说明
本模块提供 VPP ACL（访问控制列表）类型定义的 Go 语言绑定，由 GoVPP binapi-generator 自动生成。

## 文件清单
- `acl_types.ba.go`: ACL 类型定义（数据结构、常量、枚举）

## 依赖关系
- `go.fd.io/govpp`: VPP Go 绑定基础库

## 修改说明
- ✅ 已添加: 包级别中文文档注释
- ✅ 已添加: go.mod 文件声明模块依赖
- ❌ 未修改: 数据结构和接口定义（保持与上游完全一致）

## 目录位置
本模块位于 `internal/binapi_acl_types/`，与 `internal/acl/` (sdk-vpp ACL) 并列放置。

## 升级指南
如需升级到新版本 govpp，请：
1. 查询新版本的 commit hash 和 go.sum hash
2. 重新下载对应版本的 binapi/acl_types 模块
3. 对比差异，确认无破坏性变更
4. 更新本 README.md 中的版本信息
```

### 步骤 6: 修改项目 go.mod

在项目根目录的 `go.mod` 文件末尾添加 replace 指令：

```bash
# 备份 go.mod
cp go.mod go.mod.backup

# 手动编辑 go.mod，在文件末尾添加:
# replace github.com/networkservicemesh/govpp/binapi/acl_types => ./internal/binapi_acl_types
```

或使用脚本自动添加：

```bash
cat >> go.mod <<'EOF'

// ACL 模块本地化 replace 指令
replace (
    github.com/networkservicemesh/govpp/binapi/acl_types => ./internal/binapi_acl_types
)
EOF
```

### 步骤 7: 验证编译

```bash
# 清理缓存
go clean -modcache

# 验证 replace 生效
go list -m github.com/networkservicemesh/govpp/binapi/acl_types
# 期望输出: github.com/networkservicemesh/govpp/binapi/acl_types => ./internal/binapi_acl_types

# 编译项目
go build ./...
# 期望: 编译成功，无错误

# 运行测试（如果有）
go test ./...
```

### 步骤 8: Git 提交

```bash
# 添加所有修改文件
git add internal/binapi_acl_types/
git add go.mod go.sum

# 提交
git commit -m "feat: 本地化 govpp binapi/acl_types 模块

- 来源: github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba
- 版本哈希: h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
- 本地路径: internal/binapi_acl_types
- 修改内容: 添加中文注释和 README.md，保持代码不变
- 目标镜像版本: v1.0.1"
```

### 步骤 9: 构建 Docker 镜像

```bash
# 创建 VERSION 文件
echo "v1.0.1" > VERSION
git add VERSION
git commit --amend --no-edit  # 将 VERSION 文件并入上一次提交

# 构建镜像
VERSION=$(cat VERSION)
docker build -t ifzzh520/vpp-acl-firewall:${VERSION} -t ifzzh520/vpp-acl-firewall:latest .

# 验证镜像构建成功
docker images | grep vpp-acl-firewall
# 期望看到: ifzzh520/vpp-acl-firewall   v1.0.1   ...
```

### 步骤 10: 推送镜像供用户测试

```bash
# 推送到镜像仓库
docker push ifzzh520/vpp-acl-firewall:${VERSION}
docker push ifzzh520/vpp-acl-firewall:latest

# 打 Git tag
git tag ${VERSION} -m "ACL 模块本地化迭代 1: binapi/acl_types"
git push origin ${VERSION}
git push origin HEAD

echo "✅ 迭代 1 完成，镜像版本: ${VERSION}"
echo "请用户测试镜像功能，确认无问题后继续迭代 2"
```

---

## 迭代 2: 本地化 binapi/acl

**前置条件**: 迭代 1 用户测试通过

### 步骤 1: 下载并复制 binapi/acl

```bash
# govpp 模块已在迭代 1 中下载，直接使用缓存
GOVPP_CACHE=$(go env GOPATH)/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba

# 创建本地模块目录（与 internal/acl/ 并列）
mkdir -p internal/binapi_acl

# 复制源码
cp -r $GOVPP_CACHE/binapi/acl/* internal/binapi_acl/

# 取消只读权限
chmod -R u+w internal/binapi_acl/

# 验证文件已复制
ls -la internal/binapi_acl/
# 期望看到: acl.ba.go, acl_rpc.ba.go
```

### 步骤 2: 创建 go.mod 文件

在 `internal/binapi_acl/` 目录下创建 `go.mod`:

```go
module github.com/ifzzh/cmd-nse-template/internal/binapi_acl

go 1.23

require (
    go.fd.io/govpp v0.11.0
    github.com/networkservicemesh/govpp/binapi/acl_types v0.0.0-20240328101142-8a444680fbba
)
```

**重要**: 由于 `binapi/acl` 依赖 `binapi/acl_types`，需要在 go.mod 中声明依赖。

```bash
# 执行 go mod tidy
cd internal/binapi_acl/
go mod tidy
cd ../..
```

### 步骤 3: 添加包级别中文注释

编辑 `internal/binapi_acl/acl.ba.go`，在文件开头添加：

```go
// Package acl 提供 VPP ACL（访问控制列表）API 的 Go 语言绑定。
//
// 本模块由 GoVPP binapi-generator 自动生成，对应 VPP API 版本。
// 原始仓库: github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba
// 本地化日期: 2025-11-13
// 版本哈希: h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
//
// 注意: 本代码为自动生成，请勿手动修改数据结构和接口定义。
```

### 步骤 4: 创建 README.md

在 `internal/binapi_acl/` 目录下创建 `README.md`:

```markdown
# govpp binapi/acl 本地化模块

## 来源信息
- **原始仓库**: github.com/networkservicemesh/govpp
- **版本**: v0.0.0-20240328101142-8a444680fbba
- **Commit Hash**: 8a444680fbba
- **go.sum Hash**: h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
- **本地化日期**: 2025-11-13

## 模块说明
本模块提供 VPP ACL（访问控制列表）API 的 Go 语言绑定，由 GoVPP binapi-generator 自动生成。

## 文件清单
- `acl.ba.go`: ACL API 绑定（数据结构、消息类型、序列化方法）
- `acl_rpc.ba.go`: ACL RPC 客户端（RPC 接口、异步调用方法）

## 依赖关系
- `go.fd.io/govpp`: VPP Go 绑定基础库
- `github.com/networkservicemesh/govpp/binapi/acl_types`: ACL 类型定义（已本地化到 internal/binapi_acl_types）

## 修改说明
- ✅ 已添加: 包级别中文文档注释
- ✅ 已添加: go.mod 文件声明模块依赖
- ❌ 未修改: 数据结构和接口定义（保持与上游完全一致）

## 目录位置
本模块位于 `internal/binapi_acl/`，与 `internal/acl/` (sdk-vpp ACL) 和 `internal/binapi_acl_types/` (govpp acl_types) 并列放置。

## 升级指南
如需升级到新版本 govpp，请：
1. 查询新版本的 commit hash 和 go.sum hash
2. 重新下载对应版本的 binapi/acl 模块
3. 对比差异，确认无破坏性变更
4. 同步升级 binapi/acl_types 依赖版本
5. 更新本 README.md 中的版本信息
```

### 步骤 5: 更新项目 go.mod

在项目根目录的 `go.mod` 文件中，更新 replace 指令部分：

```bash
# 编辑 go.mod，修改 replace 部分为:
replace (
    github.com/networkservicemesh/govpp/binapi/acl_types => ./internal/binapi_acl_types
    github.com/networkservicemesh/govpp/binapi/acl => ./internal/binapi_acl
)
```

### 步骤 6: 验证编译

```bash
# 清理缓存
go clean -modcache

# 验证 replace 生效
go list -m github.com/networkservicemesh/govpp/binapi/acl
# 期望输出: github.com/networkservicemesh/govpp/binapi/acl => ./internal/binapi_acl

# 编译项目
go build ./...
# 期望: 编译成功，无错误

# 运行测试
go test ./...
```

### 步骤 7: Git 提交

```bash
# 添加所有修改文件
git add internal/binapi_acl/
git add go.mod go.sum

# 提交
git commit -m "feat: 本地化 govpp binapi/acl 模块

- 来源: github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba
- 版本哈希: h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=
- 本地路径: internal/binapi_acl
- 修改内容: 添加中文注释和 README.md，保持代码不变
- 依赖: binapi/acl_types (已本地化)
- 目标镜像版本: v1.0.2"
```

### 步骤 8: 构建 Docker 镜像

```bash
# 更新 VERSION 文件
echo "v1.0.2" > VERSION
git add VERSION
git commit --amend --no-edit

# 构建镜像
VERSION=$(cat VERSION)
docker build -t ifzzh520/vpp-acl-firewall:${VERSION} -t ifzzh520/vpp-acl-firewall:latest .

# 验证镜像构建成功
docker images | grep vpp-acl-firewall
# 期望看到: ifzzh520/vpp-acl-firewall   v1.0.2   ...
```

### 步骤 9: 推送镜像供用户测试

```bash
# 推送到镜像仓库
docker push ifzzh520/vpp-acl-firewall:${VERSION}
docker push ifzzh520/vpp-acl-firewall:latest

# 打 Git tag
git tag ${VERSION} -m "ACL 模块本地化迭代 2: binapi/acl"
git push origin ${VERSION}
git push origin HEAD

echo "✅ 迭代 2 完成，镜像版本: ${VERSION}"
echo "请用户测试镜像功能，确认无问题后本地化工作完成"
```

---

## 用户功能测试清单

每次镜像构建后，用户需要执行以下测试：

### 1. 部署测试
```bash
# 拉取新镜像
docker pull ifzzh520/vpp-acl-firewall:v1.0.X

# 部署到 Kubernetes 测试环境
kubectl apply -f deployment.yaml

# 检查 Pod 状态
kubectl get pods -l app=vpp-acl-firewall
# 期望: Pod 状态为 Running
```

### 2. NSM 注册验证
```bash
# 查看日志，确认 NSM 注册成功
kubectl logs <pod-name> | grep "NSM"
# 期望看到: "NSM 注册成功" 或类似日志（中文）
```

### 3. VPP 连接验证
```bash
# 进入 Pod 执行 VPP 命令
kubectl exec -it <pod-name> -- vppctl show version
# 期望: 输出 VPP 版本信息
```

### 4. ACL 规则验证
```bash
# 查看 ACL 规则
kubectl exec -it <pod-name> -- vppctl show acl-plugin acl
# 期望: 输出配置的 ACL 规则
```

### 5. 网络流量测试
```bash
# 测试允许的流量
curl <允许的目标IP>
# 期望: 连接成功

# 测试拒绝的流量
curl <拒绝的目标IP>
# 期望: 连接超时或拒绝
```

### 6. 日志检查
```bash
# 查看所有日志
kubectl logs <pod-name>
# 期望: 所有日志为中文，无错误信息
```

---

## 回滚指南

如果测试失败，执行以下回滚操作：

### 回滚镜像
```bash
# 使用上一个稳定版本
kubectl set image deployment/vpp-acl-firewall app=ifzzh520/vpp-acl-firewall:v1.0.X-1

# 或回滚到基线版本
kubectl set image deployment/vpp-acl-firewall app=ifzzh520/vpp-acl-firewall:v1.0.0
```

### 回滚代码
```bash
# 回滚到上一次提交
git reset --hard HEAD~1

# 删除失败的 tag
VERSION="v1.0.X"  # 失败的版本
git tag -d ${VERSION}
git push origin :refs/tags/${VERSION}

# 恢复 go.mod
git checkout HEAD go.mod go.sum

# 删除本地化模块目录（如果需要）
rm -rf internal/binapi_acl  # 或 binapi_acl_types
```

### 定位问题
```bash
# 检查编译错误
go build -v ./...

# 检查依赖冲突
go mod graph | grep acl

# 检查 replace 是否生效
go list -m all | grep acl

# 对比代码差异
git diff HEAD~1 internal/binapi_acl_types/
git diff HEAD~1 internal/binapi_acl/
```

---

## 常见问题 (FAQ)

### Q1: 编译时提示 "module not found"
**原因**: replace 路径错误或本地模块缺少 go.mod

**解决**:
```bash
# 验证路径
ls -la internal/binapi_acl/go.mod

# 如果缺少，创建 go.mod
cd internal/binapi_acl/
go mod init github.com/ifzzh/cmd-nse-template/internal/binapi_acl
go mod tidy
```

### Q2: Docker 构建失败
**原因**: Dockerfile 无法访问本地模块或依赖解析失败

**解决**:
```bash
# 确保 Dockerfile 中包含 COPY go.mod go.sum 和 COPY internal/ 步骤
# 检查 Dockerfile 中的工作目录设置
```

### Q3: 用户测试时 ACL 功能异常
**原因**: 可能修改了 binapi 接口定义或数据结构

**解决**:
```bash
# 对比本地代码与原始缓存代码
diff -r internal/binapi_acl/ \
       $(go env GOPATH)/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20240328101142-8a444680fbba/binapi/acl/

# 如果有非预期的差异，重新复制原始代码
```

### Q4: go.sum 哈希不匹配
**原因**: 本地修改导致模块哈希变化

**解决**:
```bash
# 执行 go mod tidy 重新计算哈希
go mod tidy

# 如果仍然失败，清理缓存
go clean -modcache
go mod download
```

### Q5: 目录命名冲突（binapi/acl vs internal/acl）
**原因**: 直接使用 acl 命名与现有目录冲突

**解决**:
- ✅ 使用 `binapi_acl/` 和 `binapi_acl_types/` 统一命名（推荐）
- ✅ 与 `internal/acl/` (sdk-vpp ACL) 区分清晰
- ❌ 避免直接使用 `acl/` 或 `acl_types/` 导致混淆

---

## 完成标志

当以下条件全部满足时，ACL 模块本地化工作完成：

- [x] binapi/acl_types 已本地化到 `internal/binapi_acl_types/`
- [x] binapi/acl 已本地化到 `internal/binapi_acl/`
- [x] 项目 go.mod 包含正确的 replace 指令
- [x] 所有本地化模块包含 go.mod、README.md 和中文注释
- [x] 项目编译通过: `go build ./...`
- [x] Docker 镜像构建成功
- [x] 用户功能测试全部通过
- [x] Git 提交和 tag 已推送: v1.0.1, v1.0.2
- [x] 镜像已推送到仓库: ifzzh520/vpp-acl-firewall:v1.0.1, v1.0.2

---

## 目录结构总览（最终状态）

```
internal/
├── acl/                         # sdk-vpp ACL（已本地化）
│   ├── common.go
│   └── server.go
├── binapi_acl_types/            # govpp binapi/acl_types（迭代1）
│   ├── acl_types.ba.go
│   ├── go.mod
│   └── README.md
├── binapi_acl/                  # govpp binapi/acl（迭代2）
│   ├── acl.ba.go
│   ├── acl_rpc.ba.go
│   ├── go.mod
│   └── README.md
├── config.go
└── imports/
    └── imports_linux.go
```

**命名说明**:
- `binapi_acl_types/` = govpp binapi/acl_types（统一 binapi 前缀）
- `binapi_acl/` = govpp binapi/acl（与 acl/ 区分）
- 所有模块与 `acl/` 并列，不使用嵌套子目录

---

**指南版本**: 2.0 | **最后更新**: 2025-11-13
