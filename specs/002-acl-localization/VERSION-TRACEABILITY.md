# 版本追溯文档 / Version Traceability Document

**功能**: ACL 模块本地化 (002-acl-localization)
**最后更新**: 2025-11-13
**维护者**: [@ifzzh](https://github.com/ifzzh)

本文档记录每个本地化模块的原始来源信息，用于版本追溯、升级和问题排查。

---

## 📦 本地化模块清单

### 1. binapi_acl_types - VPP ACL 类型定义

**本地路径**: `internal/binapi_acl_types/`

#### 原始来源
- **仓库地址**: `github.com/networkservicemesh/govpp`
- **Commit 哈希**: `v0.0.0-20240328101142-8a444680fbba`
- **相对路径**: `binapi/acl_types/`
- **VPP API 版本**: 23.10-rc0~170-g6f1548434
- **binapi-generator 版本**: v0.10.0-dev

#### Go 模块信息
- **go.sum 哈希**: `h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o=`
- **直接依赖**: `go.fd.io/govpp v0.11.0`
- **间接依赖**: `github.com/lunixbochs/struc v0.0.0-20200521075829-a4cb8d33dbbe`

#### 本地化信息
- **本地化日期**: 2025-11-13
- **Docker 镜像**: `ifzzh520/vpp-acl-firewall:v1.0.1`
- **Git 标签**: `v1.0.1`
- **提交哈希**: `a6dc570` (初始本地化) → `57aa56e` (v1.0.1 稳定版)

#### 文件清单
| 文件名 | 大小 | 说明 |
|--------|------|------|
| `acl_types.ba.go` | ~19KB | ACL 类型定义（自动生成） |
| `go.mod` | ~200B | 模块依赖声明 |
| `go.sum` | ~800B | 依赖校验和 |
| `README.md` | ~4KB | 模块文档和升级指南 |

#### 代码修改
- ✅ 添加包级别中文注释（9行）
- ✅ 创建 go.mod 文件
- ✅ 创建 README.md 文档
- ❌ 未修改任何生成的代码逻辑

---

### 2. binapi_acl - VPP ACL 插件 API 和 RPC

**本地路径**: `internal/binapi_acl/`

#### 原始来源
- **仓库地址**: `github.com/networkservicemesh/govpp`
- **Commit 哈希**: `v0.0.0-20240328101142-8a444680fbba`
- **相对路径**: `binapi/acl/`
- **VPP API 版本**: 23.10-rc0~170-g6f1548434
- **binapi-generator 版本**: v0.10.0-dev

#### Go 模块信息
- **go.sum 哈希**: (见 internal/binapi_acl/go.sum)
- **直接依赖**:
  - `go.fd.io/govpp v0.11.0`
  - `github.com/networkservicemesh/govpp/binapi/acl_types` (本地化为 `../binapi_acl_types`)
  - `github.com/networkservicemesh/govpp/binapi/ethernet_types`
  - `github.com/networkservicemesh/govpp/binapi/interface_types`
  - `github.com/networkservicemesh/govpp/binapi/ip_types`
- **间接依赖**: `github.com/lunixbochs/struc v0.0.0-20200521075829-a4cb8d33dbbe`

#### 本地化信息
- **本地化日期**: 2025-11-13
- **Docker 镜像**: `ifzzh520/vpp-acl-firewall:v1.0.2`
- **Git 标签**: `v1.0.2`
- **提交哈希**: `ba9deb6`

#### 文件清单
| 文件名 | 大小 | 说明 |
|--------|------|------|
| `acl.ba.go` | ~70KB | ACL API 消息定义（42个消息，自动生成） |
| `acl_rpc.ba.go` | ~12KB | ACL RPC 方法定义（自动生成） |
| `go.mod` | ~400B | 模块依赖声明 + replace 指令 |
| `go.sum` | ~1.4KB | 依赖校验和 |
| `README.md` | ~5KB | 模块文档和升级指南 |

#### 代码修改
- ✅ 添加包级别中文注释（14行）
- ✅ 创建 go.mod 文件（含 replace 指令）
- ✅ 创建 README.md 文档
- ❌ 未修改任何生成的代码逻辑

---

### 3. acl - ACL 防火墙核心逻辑（已存在）

**本地路径**: `internal/acl/`

#### 原始来源
- **仓库地址**: `github.com/networkservicemesh/sdk-vpp`
- **相对路径**: `pkg/networkservice/...`
- **本地化日期**: 2025-01-12 (v1.0.0)

#### 说明
本模块是自研的 ACL 防火墙业务逻辑，不是 binapi 生成代码。已在 v1.0.0 中本地化。

#### 文件清单
| 文件名 | 说明 |
|--------|------|
| `common.go` | 公共函数（185 行代码 + 69 行注释） |
| `server.go` | 服务器实现（168 行代码 + 75 行注释） |

---

## 🔄 版本映射关系

| Docker 镜像版本 | Git 标签 | 本地化模块 | 提交哈希 | 发布日期 |
|-----------------|----------|------------|----------|----------|
| `v1.0.2` | `v1.0.2` | `binapi_acl_types` + `binapi_acl` | `ba9deb6` | 2025-11-13 |
| `v1.0.1` | `v1.0.1` | `binapi_acl_types` | `57aa56e` | 2025-11-13 |
| `v1.0.0` | `v1.0.0` | `acl` (业务逻辑) | - | 2025-01-12 |

---

## 📍 模块依赖图

```
项目 (github.com/ifzzh/cmd-nse-template)
│
├── internal/acl/                      # 业务逻辑 (v1.0.0)
│   ├── 依赖: govpp/binapi/acl         → 指向 internal/binapi_acl/ (replace)
│   └── 依赖: govpp/binapi/acl_types   → 指向 internal/binapi_acl_types/ (replace)
│
├── internal/binapi_acl_types/         # ACL 类型定义 (v1.0.1)
│   └── 依赖: go.fd.io/govpp v0.11.0
│
└── internal/binapi_acl/               # ACL API 和 RPC (v1.0.2)
    ├── 依赖: go.fd.io/govpp v0.11.0
    └── 依赖: govpp/binapi/acl_types   → 指向 ../binapi_acl_types (replace)
```

---

## 🔍 快速定位指南

### 场景 1: 升级上游 govpp 模块

1. 查找本文档对应模块的 **Commit 哈希** 和 **相对路径**
2. 下载新版本: `go mod download github.com/networkservicemesh/govpp@<new-version>`
3. 定位缓存路径: `$(go env GOPATH)/pkg/mod/github.com/networkservicemesh/govpp@<new-version>/<相对路径>`
4. 按照模块 README.md 中的升级指南执行

### 场景 2: 排查类型不匹配错误

1. 验证所有模块来自同一个上游版本: `v0.0.0-20240328101142-8a444680fbba`
2. 检查项目 go.mod 中的 replace 指令是否完整
3. 运行 `go mod verify` 验证模块完整性

### 场景 3: 回滚到特定版本

1. 查找目标版本的 **Git 标签** 和 **Docker 镜像版本**
2. 回滚代码: `git checkout <Git 标签>`
3. 重新构建镜像: `docker build -t ifzzh520/vpp-acl-firewall:<版本号> .`

---

## 📖 相关文档

- [internal/binapi_acl_types/README.md](../../internal/binapi_acl_types/README.md) - ACL 类型模块详细文档
- [internal/binapi_acl/README.md](../../internal/binapi_acl/README.md) - ACL 插件模块详细文档
- [go.mod](../../go.mod) - 项目级别 replace 指令配置
- [README.md](../../README.md) - 项目主文档
- [quickstart.md](./quickstart.md) - 模块本地化操作手册

---

## ⚠️ 注意事项

1. **版本一致性**: 所有 binapi 模块必须来自同一个 govpp 版本，避免类型定义不一致
2. **replace 指令**: 确保项目 go.mod 和模块内部 go.mod 的 replace 指令正确配置
3. **只读代码**: binapi 生成的代码（`*.ba.go`）不应手动修改，升级时会被覆盖
4. **升级同步**: 升级时需同步更新 binapi_acl_types 和 binapi_acl，确保兼容性
5. **版本追溯**: 升级后更新本文档的版本信息和哈希值

---

**维护说明**: 每次本地化新模块或升级现有模块时，必须更新本文档。
