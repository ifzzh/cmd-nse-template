# binapi_acl - VPP ACL 插件 Go 语言绑定（本地化模块）

## 模块来源

本模块从以下位置本地化而来：

- **原始仓库**: `github.com/networkservicemesh/govpp`
- **版本哈希**: `v0.0.0-20240328101142-8a444680fbba`
- **相对路径**: `binapi/acl/`
- **本地化日期**: 2025-11-13
- **VPP API 版本**: 23.10-rc0~170-g6f1548434
- **binapi-generator 版本**: v0.10.0-dev

## 文件清单

| 文件名 | 大小 | 说明 |
|--------|------|------|
| `acl.ba.go` | ~70KB | ACL API 消息定义（42个消息） |
| `acl_rpc.ba.go` | ~12KB | ACL RPC 方法定义 |
| `go.mod` | ~400B | 模块依赖声明 |
| `go.sum` | ~1.4KB | 依赖校验和 |
| `README.md` | 本文件 | 模块文档 |

## 依赖关系

本模块依赖以下包：

### 直接依赖
- `go.fd.io/govpp v0.11.0` - GoVPP 核心库
- `github.com/networkservicemesh/govpp/binapi/acl_types` - ACL 类型定义（本地化模块 `../binapi_acl_types`）
- `github.com/networkservicemesh/govpp/binapi/ethernet_types` - 以太网类型
- `github.com/networkservicemesh/govpp/binapi/interface_types` - 接口类型
- `github.com/networkservicemesh/govpp/binapi/ip_types` - IP 类型

### 间接依赖
- `github.com/lunixbochs/struc v0.0.0-20200521075829-a4cb8d33dbbe` - 结构体序列化库

### Replace 指令
```go
replace github.com/networkservicemesh/govpp/binapi/acl_types => ../binapi_acl_types
```

**说明**: 本模块通过 replace 指令引用已本地化的 `acl_types` 模块，确保类型一致性并避免网络依赖。

## 代码修改说明

本地化过程中的修改：

### 1. 添加 go.mod 文件
- 声明模块路径: `github.com/ifzzh/cmd-nse-template/internal/binapi_acl`
- 指定 Go 版本: `1.23.8`
- 添加 replace 指令指向本地 acl_types 模块

### 2. 添加包级别中文注释
在 `acl.ba.go` 文件开头添加了详细的中文文档注释，包括：
- 模块功能说明
- 原始来源信息（仓库、版本哈希、相对路径、本地化日期）
- 注意事项（代码生成、依赖关系、升级说明）

### 3. 创建本 README 文档
记录模块来源、文件清单、依赖关系、修改说明和升级指南。

### 4. 保留原始代码
所有生成的 Go 代码文件（`acl.ba.go`、`acl_rpc.ba.go`）保持原样，未修改任何函数签名或逻辑。

## 升级指南

当上游 `github.com/networkservicemesh/govpp` 模块更新时，按以下步骤升级本模块：

### 1. 下载新版本到缓存
```bash
# 假设新版本为 v0.0.0-20250401000000-abcdef123456
go mod download github.com/networkservicemesh/govpp@v0.0.0-20250401000000-abcdef123456
```

### 2. 定位缓存路径
```bash
GOVPP_CACHE=$(go env GOPATH)/pkg/mod/github.com/networkservicemesh/govpp@v0.0.0-20250401000000-abcdef123456
ls -la $GOVPP_CACHE/binapi/acl/
```

### 3. 备份当前模块
```bash
cp -r internal/binapi_acl internal/binapi_acl.bak
```

### 4. 复制新版本代码
```bash
cp -r $GOVPP_CACHE/binapi/acl/acl.ba.go internal/binapi_acl/
cp -r $GOVPP_CACHE/binapi/acl/acl_rpc.ba.go internal/binapi_acl/
chmod -R u+w internal/binapi_acl/
```

### 5. 更新本地修改
- 重新添加包级别中文注释到 `acl.ba.go`（参考当前注释内容）
- 更新本 README.md 中的版本信息

### 6. 验证编译和测试
```bash
# 在 binapi_acl 目录中
cd internal/binapi_acl
go mod tidy
go build .

# 在项目根目录
cd ../..
go build ./...
go test ./...
```

### 7. 提交更改
```bash
git add internal/binapi_acl/
git commit -m "chore(binapi_acl): 升级到 govpp v0.0.0-20250401000000-abcdef123456

- 更新 ACL API bindings 到新版本
- VPP API 版本: [新的 VPP 版本]
- binapi-generator 版本: [新的生成器版本]
"
```

## 集成到项目

本模块已通过项目根目录的 `go.mod` 文件集成：

```go
// 项目 go.mod
replace github.com/networkservicemesh/govpp/binapi/acl => ./internal/binapi_acl
```

业务代码中保持原始导入路径：
```go
import (
    "github.com/networkservicemesh/govpp/binapi/acl"
)
```

编译时会自动重定向到本地模块 `./internal/binapi_acl`。

## 注意事项

1. **不要手动编辑 `acl.ba.go` 和 `acl_rpc.ba.go`**: 这些文件由代码生成器创建，手动修改会在升级时丢失
2. **依赖版本一致性**: 确保 `binapi_acl_types` 模块与本模块来自同一个 govpp 版本
3. **同步升级**: 升级本模块时，如果 acl_types 也有变更，需要同步升级 `binapi_acl_types` 模块
4. **类型兼容性**: 本模块依赖 `acl_types`，升级时需确保类型定义兼容

## 相关文档

- [internal/binapi_acl_types/README.md](../binapi_acl_types/README.md) - ACL 类型模块文档
- [项目 go.mod](../../go.mod) - 查看项目级别的 replace 指令
- [主 README](../../README.md) - 项目整体文档
