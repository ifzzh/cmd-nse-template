<!--
SYNC IMPACT REPORT
==================
版本变更: 1.0.0 → 1.1.0
变更类型: MINOR（新增根目录文件清单章节，扩展简洁架构原则）

修改的原则:
- 扩展: II. 简洁架构原则 - 新增"根目录文件清单"子章节，逐个分析所有文件/目录并标注保留/删除决策

新增章节:
- 根目录文件清单：详细列出18个文件/目录的用途、关系和保留/删除建议

需要同步更新的模板文件:
✅ .specify/templates/plan-template.md - 需确保"宪章合规性检查"与7条原则一致
✅ .specify/templates/spec-template.md - 需增加"中文优先"和"简洁架构"约束
✅ .specify/templates/tasks-template.md - 需增加"代码一致性验证"任务类型
⚠ .specify/templates/commands/*.md - 待检查是否有硬编码的原则引用

执行建议:
- 立即删除: .github/, .license/, .templateignore, .yamllint.yml（高优先级）
- 稍后考虑删除: staticcheck.conf（低优先级）
- 必须保留: main.go, go.mod, go.sum, Dockerfile, internal/, README.md等核心文件
-->

# cmd-nse-firewall-vpp 项目宪章

本项目是基于 Network Service Mesh (NSM) 框架和 VPP（Vector Packet Processing）技术的防火墙网络服务端点实现，提供高性能的网络流量ACL控制能力。

---

## 核心原则

### I. 中文优先原则（强制遵守）

**范围**: 除代码标识符外，所有可读文本必须使用简体中文

- ✅ **强制中文**: 代码注释、日志输出、错误信息、文档、Git提交信息
- ✅ **保持英文**: 变量名、函数名、类型名、包名（遵循Go语言规范）
- ⚠️ **违规处理**: 违反此原则的提交将被拒绝

**理由**: 确保项目协作者能够快速理解代码意图和业务逻辑，降低沟通成本，提高维护效率。

**示例**:
```go
// ✅ 正确
// 初始化防火墙配置，加载ACL规则
func initFirewallConfig() error {
    log.Info("开始加载ACL规则")
    return nil
}

// ❌ 错误
// Initialize firewall configuration
func initFirewallConfig() error {
    log.Info("Start loading ACL rules")
    return nil
}
```

---

### II. 简洁架构原则（强制遵守）

**目标**: 维持扁平化、可理解的目录结构

**规则**:
- 禁止深度嵌套（最大深度≤3层）
- 每个目录必须有明确的单一职责
- 删除与核心功能无关的代码，包括但不限于:
  - `.github/workflows/` (CI/CD配置)
  - 未使用的vendor目录
  - 临时文件和构建产物

**允许保留的目录**:
- `internal/` - 内部实现
- `.specify/` - SpecKit配置
- `.claude/` - Claude Code工作文件

**理由**: 简洁的结构降低认知负担，使新贡献者能够快速定位代码位置，减少维护复杂度。

**标准目录结构**:
```
cmd-nse-firewall-vpp/
├── main.go              # 主程序入口
├── internal/            # 内部实现（按需使用）
├── .specify/            # SpecKit配置
├── .claude/             # AI工作目录
├── Dockerfile           # 容器化配置
├── go.mod, go.sum      # Go依赖
└── README.md            # 项目文档
```

#### 根目录文件清单（强制遵守）

以下为项目根目录所有文件和目录的详细分析，每项标注是否保留/删除及理由。

| # | 文件/目录 | 类型 | 用途说明 | 与项目关系 | 决策 | 理由 | 删除优先级 |
|---|----------|------|---------|-----------|------|------|-----------|
| 1 | **main.go** | 文件 | 项目主程序入口（380行），实现完整的NSM防火墙端点功能 | 核心代码 | ✅ **必须保留** | 项目核心业务逻辑，删除将导致功能完全丧失 | N/A |
| 2 | **go.mod** | 文件 | Go模块依赖定义文件 | 依赖管理 | ✅ **必须保留** | 依赖稳定性原则要求，构建系统的基础 | N/A |
| 3 | **go.sum** | 文件 | Go模块依赖校验和文件 | 依赖管理 | ✅ **必须保留** | 确保依赖完整性和可重复构建 | N/A |
| 4 | **Dockerfile** | 文件 | 容器化构建配置，支持test/debug/release多阶段构建 | 构建工具 | ✅ **必须保留** | 项目唯一的容器化构建方式，测试依赖此文件 | N/A |
| 5 | **internal/** | 目录 | 内部实现目录，包含imports子目录 | 核心代码 | ✅ **必须保留** | Go模块内部实现，符合标准Go项目结构 | N/A |
| 6 | **README.md** | 文件 | 项目文档，说明构建、使用、测试、调试方法 | 项目文档 | ✅ **必须保留** | 项目的使用说明文档，需后续改为中文 | N/A |
| 7 | **.gitignore** | 文件 | Git忽略规则配置 | 版本控制 | ✅ **必须保留** | 版本控制必需文件，防止临时文件提交 | N/A |
| 8 | **.git/** | 目录 | Git版本控制元数据 | 版本控制 | ✅ **必须保留** | 版本控制系统核心目录，不可删除 | N/A |
| 9 | **LICENSE** | 文件 | Apache 2.0开源许可证 | 法律文档 | ✅ **必须保留** | 开源项目法律保护文件，删除将导致许可证不明 | N/A |
| 10 | **SECURITY.md** | 文件 | 安全漏洞报告指南 | 项目文档 | ✅ **保留** | 提供安全问题反馈渠道，标准开源项目文档 | N/A |
| 11 | **.golangci.yml** | 文件 | golangci-lint静态检查配置 | 代码质量工具 | ✅ **保留** | 代码质量标准要求的静态检查工具配置 | N/A |
| 12 | **.specify/** | 目录 | SpecKit配置目录，包含宪章、模板、脚本 | 项目管理 | ✅ **必须保留** | 宪章文件所在目录，项目治理核心 | N/A |
| 13 | **.claude/** | 目录 | Claude Code工作目录，存储操作日志和上下文 | AI工作文件 | ✅ **必须保留** | AI辅助开发工作文件，记录决策和操作历史 | N/A |
| 14 | **.github/** | 目录 | GitHub CI/CD配置（workflows子目录） | CI/CD配置 | ❌ **必须删除** | 违反"简洁架构原则"，CI/CD配置与核心功能无关 | **高优先级** |
| 15 | **.license/** | 目录 | 许可证模板目录（README.md, template.txt） | 模板文件 | ❌ **必须删除** | 仅用于模板项目，本项目已有LICENSE文件，冗余 | **高优先级** |
| 16 | **.templateignore** | 文件 | 模板项目忽略规则（内容：README.md） | 模板文件 | ❌ **必须删除** | 仅用于模板项目初始化，本项目已实例化，无用 | **高优先级** |
| 17 | **.yamllint.yml** | 文件 | YAML格式检查工具配置 | 代码质量工具 | ❌ **建议删除** | 项目无YAML配置文件，此工具配置无实际用途 | **高优先级** |
| 18 | **staticcheck.conf** | 文件 | staticcheck静态分析工具配置 | 代码质量工具 | ⚠️ **可选删除** | .golangci.yml已包含静态检查，此文件可能冗余 | **低优先级** |

**决策汇总**:
- ✅ **必须保留**（13项）: main.go, go.mod, go.sum, Dockerfile, internal/, README.md, .gitignore, .git/, LICENSE, SECURITY.md, .golangci.yml, .specify/, .claude/
- ❌ **必须删除**（4项）: .github/, .license/, .templateignore, .yamllint.yml
- ⚠️ **可选删除**（1项）: staticcheck.conf

**执行建议**:
1. **立即执行**（高优先级删除）:
   ```bash
   rm -rf .github/
   rm -rf .license/
   rm .templateignore
   rm .yamllint.yml
   ```

2. **评估后执行**（低优先级）:
   - 检查staticcheck.conf是否被实际使用
   - 如未使用则删除: `rm staticcheck.conf`

3. **后续改进**:
   - 将README.md内容改为中文（符合"中文优先原则"）
   - 验证internal/imports/目录的必要性

---

### III. 依赖稳定性原则（强制遵守）

**规则**: 模块路径和版本变更需严格控制

**模块路径变更流程**:
1. 必须经过文档化审批
2. 需更新所有引用位置
3. 确认不破坏现有功能

**版本升级流程**:
1. 优先查询本地项目是否使用fork版本
2. 查阅官方文档确认API变更
3. 执行本地测试验证兼容性
4. 记录升级理由和风险评估

**知识获取顺序**:
- 第一步: 检查本地代码实际使用的模块
- 第二步: 查询对应版本的官方文档
- 第三步: 联网搜索社区最佳实践

**理由**: 依赖不稳定是导致构建失败和运行时错误的主要原因，严格控制变更可避免意外破坏。

**禁止事项**:
- ❌ 未经验证随意升级依赖版本
- ❌ 修改go.mod中的模块路径而不更新代码引用
- ❌ 假设API兼容性而不查阅文档

---

### IV. 功能完整性原则（强制遵守）

**目标**: 严格保持项目的完整功能不被改变

**禁止破坏性变更**:
- 不允许删除现有功能模块
- 不允许改变核心业务流程
- 不允许移除已有的API端点

**新增功能限制**:
- 新功能必须经过充分论证
- 禁止引入复杂的抽象层和框架
- 新增代码不得破坏现有功能

**核心功能保护清单**:
- VPP连接和管理
- gRPC服务端点
- ACL规则加载和应用
- NSM注册和生命周期管理
- SPIFFE/SPIRE身份认证

**理由**: 项目已经过充分验证和生产环境测试，任何功能变更都可能引入未知风险，稳定性优先于新特性。

---

### V. 一致性优先原则（强制遵守）

**规则**: 新代码必须继承原项目的设计模式和命名规范

**编码前必须研究**:
- 查找至少3个相似实现作为参考
- 识别项目中的命名模式
- 理解现有的错误处理方式
- 复用现有的工具函数和类型

**命名规范继承**:
- 变量名: 保持驼峰命名，遵循Go惯例
- 函数名: 动词+名词结构（如`retrieveACLRules`）
- 类型名: 使用领域术语（如`Config`, `NetworkServiceEndpoint`）

**设计模式复用**:
- 配置管理: 使用`envconfig`库和`Config`结构体
- 日志记录: 使用`logrus`和嵌套格式化
- 错误处理: 使用`pkg/errors`包装
- 上下文传递: 统一使用`context.Context`

**理由**: 一致性降低代码库的学习成本，使代码更易于理解和维护，避免"每个人都有自己的风格"导致的混乱。

---

### VI. 本地优先原则（建议遵守）

**操作方式**: 所有开发操作直接在工作目录进行

- ✅ **推荐**: 就地编辑现有文件
- ✅ **推荐**: 增量修改和小步提交
- ❌ **禁止**: 创建项目副本或临时目录重建
- ❌ **禁止**: 通过复制粘贴移植代码

**理由**: 直接在工作目录操作可避免文件同步问题，减少不必要的IO开销，确保变更立即可见可测试。

---

### VII. 仓库独立性原则（项目战略）

**目标**: 从上游fork独立演化，保持技术兼容

- **主仓库**: git@github.com:ifzzh/cmd-nse-template.git
- **上游参考**: github.com/networkservicemesh/cmd-nse-firewall-vpp

**演化策略**:
- 独立迭代新功能
- 选择性同步上游关键修复
- 保持NSM协议兼容性
- 维护独立的发布节奏

**模块路径策略**（强制遵守）:

为便于区分代码所有权和维护范围，项目采用以下导入路径规范：

1. **外部依赖保持不变**（来自上游 networkservicemesh）:
   ```go
   // ✅ 正确 - 保持原有路径
   "github.com/networkservicemesh/sdk/pkg/..."
   "github.com/networkservicemesh/sdk-vpp/pkg/..."
   "github.com/networkservicemesh/api/pkg/..."
   "github.com/networkservicemesh/vpphelper"
   ```

2. **本地自定义模块使用独立路径**（项目内部代码）:
   ```go
   // ✅ 正确 - 使用 ifzzh 路径
   "github.com/ifzzh/cmd-nse-template/internal"

   // ❌ 错误 - 不要使用上游路径
   "github.com/networkservicemesh/cmd-template/internal"
   ```

3. **go.mod 模块声明**:
   ```go
   module github.com/ifzzh/cmd-nse-template
   ```

**理由**:
- 项目需要根据特定需求进行定制化开发，完全依赖上游会限制灵活性，独立演化更符合长期战略
- 清晰的路径区分有助于理解代码所有权：
  - `github.com/networkservicemesh/...` = 外部依赖（上游维护）
  - `github.com/ifzzh/...` = 本地代码（自行维护）
- 避免与上游模块路径混淆，便于代码审查和依赖管理

---

## 开发规范

### 代码质量标准

**编码规范**:
- 遵循Go语言官方代码风格
- 使用`gofmt`和`golangci-lint`工具
- 保持函数复杂度≤15（圈复杂度）

**注释要求**:
- 所有导出函数必须有中文文档注释
- 复杂逻辑必须添加中文行内注释
- 注释描述"为什么"而非"是什么"

**测试标准**:
- 核心功能需提供单元测试
- 集成测试覆盖主要业务流程
- 使用Docker容器进行测试隔离

### 依赖管理

- 使用Go Modules管理依赖
- 定期审查并更新安全漏洞依赖
- 主要依赖库:
  - `github.com/networkservicemesh/sdk` - NSM核心SDK
  - `github.com/networkservicemesh/sdk-vpp` - VPP集成
  - `github.com/networkservicemesh/api` - NSM API定义
  - `go.fd.io/govpp` - VPP绑定

### 文档要求

- README.md: 使用中文描述项目用途和使用方法
- 代码注释: 使用中文解释业务逻辑
- 提交信息: 使用中文描述变更内容（格式：`类型: 简短描述`）

---

## 质量保证

### 验证流程

**本地验证**:
1. 构建: `go build ./...`
2. 测试: `docker run --privileged --rm $(docker build -q --target test .)`
3. 静态检查: `golangci-lint run`

**功能验证**:
- VPP连接正常
- gRPC服务可访问
- ACL规则正确加载
- NSM注册成功

### 审查标准

- 代码变更需符合所有核心原则
- 新增代码需提供测试覆盖
- 文档和注释使用中文
- 提交信息清晰描述变更意图

---

## 治理规则

### 宪章权威性

- 本宪章是项目的最高开发准则
- 所有代码审查必须验证宪章遵守情况
- 宪章修订需经过充分讨论和文档化

### 例外处理

- 必须提供书面理由和风险评估
- 记录在`.claude/operations-log.md`
- 限定影响范围和时间

### 持续改进

- 定期审查宪章的有效性
- 根据实践经验调整条款
- 保持宪章的简洁和可执行性

---

**版本**: 1.1.0 | **批准日期**: 2025-11-11 | **最后修订**: 2025-11-11