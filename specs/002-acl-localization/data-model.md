# Data Model: ACL 模块本地化

**Feature**: 002-acl-localization | **Date**: 2025-11-12

## 概述

本功能为 **代码库本地化任务**，不涉及业务数据模型设计。

本文档记录模块本地化过程中涉及的关键实体和文件结构。

---

## 关键实体

### 1. 本地化模块 (LocalizedModule)

**描述**: 从在线仓库迁移到本地代码库的 Go 模块

**属性**:
- `模块名称` (string): 如 "binapi/acl", "binapi/acl_types"
- `原始仓库地址` (string): "github.com/networkservicemesh/govpp"
- `版本标识` (string): "v0.0.0-20240328101142-8a444680fbba"
- `版本哈希` (string): "h1:7B6X6N7rwJNpnfsUlBavxuZdYqTx8nAKwxVS/AkuX1o="
- `Commit Hash` (string): "8a444680fbba"
- `本地路径` (string): "internal/local-deps/govpp-binapi-acl"
- `本地化日期` (date): "2025-11-12"
- `状态` (enum): "待本地化" | "本地化中" | "测试中" | "已完成"

**关系**:
- 依赖其他本地化模块（如 binapi/acl 依赖 binapi/acl_types）
- 被项目主代码引用（通过 import 语句）

**生命周期**:
1. 待本地化: 在 go.sum 中识别出需要本地化的模块
2. 本地化中: 从缓存复制代码、创建 go.mod、添加注释
3. 测试中: 编译验证、构建镜像、用户功能测试
4. 已完成: 测试通过，git 提交，镜像推送

---

### 2. 版本记录 (VersionRecord)

**描述**: 镜像版本和对应的本地化模块记录

**属性**:
- `镜像版本` (string): 如 "v1.0.1", "v1.0.2"
- `Git Commit Hash` (string): 对应的 git 提交哈希
- `Git Tag` (string): 版本标签，与镜像版本一致
- `本地化模块` (array): 本次版本包含的本地化模块列表
- `发布日期` (date): 镜像构建和推送日期
- `测试状态` (enum): "待测试" | "测试中" | "测试通过" | "测试失败"
- `测试报告` (text): 用户功能测试结果

**关系**:
- 一个版本记录对应一个或多个本地化模块（本项目每次仅一个）
- 版本记录按时间顺序递增

**示例**:
```yaml
v1.0.1:
  git_commit: abc123def456
  git_tag: v1.0.1
  modules:
    - binapi/acl_types
  release_date: 2025-11-12
  test_status: 测试通过
  test_report: "NSM 注册成功，VPP 连接正常，ACL 规则加载无误"

v1.0.2:
  git_commit: def456ghi789
  git_tag: v1.0.2
  modules:
    - binapi/acl
  release_date: 2025-11-12
  test_status: 待测试
  test_report: ""
```

---

### 3. 依赖关系 (DependencyGraph)

**描述**: 本地化模块之间的依赖关系图

**结构**:
```
binapi/acl_types (无外部依赖，仅依赖 go.fd.io/govpp)
    ↑
binapi/acl (依赖 binapi/acl_types)
    ↑
internal/acl (sdk-vpp ACL 模块，已本地化，依赖 binapi/acl)
    ↑
main.go (主程序，依赖 internal/acl)
```

**依赖规则**:
- 必须先本地化被依赖的模块（如先本地化 acl_types，再本地化 acl）
- 非 ACL 相关模块保持在线依赖（如 go.fd.io/govpp）
- 本地化模块使用 go.mod replace 指令重定向

---

## 文件结构映射

### 本地化模块目录结构（更新后）

```
internal/
├── acl/                         # sdk-vpp ACL（已本地化）
│   ├── common.go
│   └── server.go
├── binapi_acl_types/            # govpp binapi/acl_types（新增）
│   ├── acl_types.ba.go          # VPP ACL 类型定义（源码）
│   ├── go.mod                   # 模块依赖声明
│   └── README.md                # 模块来源和版本信息
├── binapi_acl/                  # govpp binapi/acl（新增）
│   ├── acl.ba.go                # VPP ACL API 绑定（源码）
│   ├── acl_rpc.ba.go            # VPP ACL RPC 客户端（源码）
│   ├── go.mod                   # 模块依赖声明
│   └── README.md                # 模块来源和版本信息
├── config.go
└── imports/
    └── imports_linux.go
```

**命名说明**（2025-11-13 更新）:
- `binapi_acl_types/` 对应 govpp binapi/acl_types（统一使用 binapi 前缀）
- `binapi_acl/` 对应 govpp binapi/acl（与现有 acl/ 区分）
- 所有本地化 binapi 模块与 `acl/` 并列放置，不使用嵌套子目录

### 版本控制文件

```
根目录/
├── VERSION                      # 当前镜像版本号（如 "v1.0.2"）
├── go.mod                       # 项目依赖管理（包含 replace 指令）
├── go.sum                       # 项目依赖校验和
└── .git/
    └── refs/tags/               # Git 版本标签（v1.0.0, v1.0.1, v1.0.2）
```

---

## 数据流

### 本地化流程数据流

```
1. 输入: go.sum (版本哈希)
   ↓
2. 查询: Go 模块缓存 ($GOPATH/pkg/mod/)
   ↓
3. 复制: 源码文件 → internal/local-deps/
   ↓
4. 生成: go.mod, README.md (元数据)
   ↓
5. 修改: 项目 go.mod (添加 replace 指令)
   ↓
6. 验证: go build, go test (编译检查)
   ↓
7. 输出: Git commit, Docker 镜像, VERSION 文件
```

### 版本迭代数据流

```
1. 输入: VERSION 文件 (当前版本)
   ↓
2. 递增: 补丁版本号 +1 (v1.0.1 → v1.0.2)
   ↓
3. 构建: Docker 镜像 (新版本标签)
   ↓
4. 推送: 镜像仓库 (ifzzh520/vpp-acl-firewall:vX.Y.Z)
   ↓
5. 测试: 用户功能验证
   ↓
6. 反馈: 测试状态 (通过/失败)
   ↓
7. 决策: 继续下一个模块 / 回滚版本
```

---

## 状态机

### 模块本地化状态机

```
[待本地化] ---> [本地化中] ---> [编译验证] ---> [镜像构建] ---> [用户测试] ---> [已完成]
      ↓             ↓              ↓               ↓              ↓
   (识别模块)    (复制代码)    (go build)     (docker build)   (部署测试)
                                                                   ↓
                                                            [测试失败] ---> [回滚]
```

**状态转换条件**:
- `待本地化 → 本地化中`: 用户启动本地化任务
- `本地化中 → 编译验证`: 代码复制和配置完成
- `编译验证 → 镜像构建`: go build 成功
- `镜像构建 → 用户测试`: docker build 成功，镜像推送完成
- `用户测试 → 已完成`: 所有功能测试通过
- `用户测试 → 回滚`: 测试失败，需回滚代码和镜像

---

## 约束和验证规则

### 版本一致性约束
- **规则**: 本地化模块代码必须与 go.sum 中记录的哈希完全匹配
- **验证**: 对比源码文件与缓存文件的 diff，确保仅有预期修改（注释、go.mod）
- **违反后果**: 编译失败或运行时行为不一致

### 依赖顺序约束
- **规则**: 必须按依赖关系顺序本地化（先 acl_types，后 acl）
- **验证**: 检查 go.mod 中的 require 语句，确认被依赖模块已本地化
- **违反后果**: 编译时找不到依赖模块

### 版本递增约束
- **规则**: 镜像版本号严格递增，补丁版本号 +1
- **验证**: 对比 VERSION 文件和 git tag，确认无版本跳跃或倒退
- **违反后果**: 版本管理混乱，难以追溯变更历史

### 接口不变约束
- **规则**: 不允许修改 binapi 模块的数据结构、函数签名、接口定义
- **验证**: 代码审查，确认仅修改注释和元数据
- **违反后果**: VPP API 通信失败，功能破坏

---

## 总结

虽然本功能不涉及传统的业务数据模型，但通过定义**本地化模块、版本记录、依赖关系**等实体，以及它们的属性、关系和状态机，可以清晰地理解模块本地化工作的结构和流程。

这种"元数据模型"帮助指导实施过程，确保版本严格控制、依赖关系正确、状态转换有序。

---

**文档版本**: 1.0 | **最后更新**: 2025-11-12
