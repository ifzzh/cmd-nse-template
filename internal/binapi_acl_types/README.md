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
- `go.mod`: 模块依赖声明
- `go.sum`: 依赖校验和

## 依赖关系
- `go.fd.io/govpp`: VPP Go 绑定基础库
- `github.com/networkservicemesh/govpp/binapi/ethernet_types`: 以太网类型（间接依赖）
- `github.com/networkservicemesh/govpp/binapi/ip_types`: IP 类型（间接依赖）

## 修改说明
- ✅ 已添加: 包级别中文文档注释
- ✅ 已添加: go.mod 文件声明模块依赖
- ✅ 已添加: README.md 文档说明
- ❌ 未修改: 数据结构和接口定义（保持与上游完全一致）

## 目录位置
本模块位于 `internal/binapi_acl_types/`，与 `internal/acl/` (sdk-vpp ACL) 并列放置。

## 升级指南
如需升级到新版本 govpp，请：
1. 查询新版本的 commit hash 和 go.sum hash
2. 重新下载对应版本的 binapi/acl_types 模块
3. 对比差异，确认无破坏性变更
4. 更新本 README.md 中的版本信息
5. 重新生成 go.mod 和 go.sum
