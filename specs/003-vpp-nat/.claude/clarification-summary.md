# VPP NAT44 NSE 澄清阶段总结

**日期**: 2025-01-13
**阶段**: Clarification 完成
**下一步**: 执行 `/speckit.plan` 生成详细实施计划

---

## 已完成的工作

### 1. 技术调研

**调研内容**：
- ✅ 分析现有 ACL 防火墙实现模式（`internal/acl/server.go`, `internal/acl/common.go`）
- ✅ 分析 NSE 双接口 SFC 架构（`main.go:210-248`）
- ✅ 研究 VPP NAT44 ED 官方文档和 API
- ✅ 研究 govpp binapi NAT 模块（`nat_types`, `nat44_ed`）
- ✅ 对比 ACL 与 NAT 的实现差异

**调研结论**：
- NAT 实现比 ACL 更简单（无需双向规则和 src/dst 交换）
- 使用 VPP NAT44 ED 官方实现（难度 2/5，代码量 ~50 行，1-2 周开发）
- 完全复用 ACL 的项目结构和开发流程

---

### 2. 规范文档更新

**更新文件**：
- ✅ [specs/003-vpp-nat/spec.md](../spec.md) - 添加 Clarifications 章节
- ✅ [specs/003-vpp-nat/implementation-approach.md](../implementation-approach.md) - 完整实施方案

**Clarifications 章节记录的关键决策**（7 个问题）：

1. **Q1**: 是否应该直接修改现有代码，还是创建新项目？
   - **A**: 直接修改现有 ACL 防火墙代码

2. **Q2**: VPP 数据面的规则/插件注入是双向的还是在某个接口发生？
   - **A**: 通过接口角色（inside/outside）实现，VPP 自动处理双向转换

3. **Q3**: 是否应该取消 ACL 中的双向规则模式？
   - **A**: 是的，NAT 不需要 src/dst 交换逻辑

4. **Q4**: NAT 插件本地化策略？
   - **A**: 遵循 ACL 模式，本地化 `nat_types` 和 `nat44_ed` 模块

5. **Q5**: 应该使用 VPP NAT44 官方实现还是自研？
   - **A**: 使用 VPP NAT44 ED 官方实现

6. **Q6**: NAT 接口角色配置？
   - **A**: Interface A = inside, Interface B = outside

7. **Q7**: SNAT 和 DNAT 的关系？
   - **A**: 两者可共存，功能不同（SNAT=P1, DNAT=P4）

---

### 3. API 接口确认

**已确认的 3 个关键 API**（来自 govpp binapi nat44_ed）：

1. **Nat44InterfaceAddDelFeature** - 配置接口角色（inside/outside）
2. **Nat44AddDelAddressRange** - 配置 NAT 地址池
3. **Nat44AddDelStaticMapping** - 静态端口映射（P4）

**API 版本信息**：
- govpp 版本：v0.0.0-20240328101142-8a444680fbba
- VPP 版本：23.10-rc0~170-g6f1548434
- API 版本：nat44_ed v5.5.0, nat_types v0.0.1

---

### 4. 参考文档整理

**VPP 官方文档**：
- NAT44 ED 插件文档：https://docs.fd.io/vpp/23.02/developer/plugins/nat44_ed_doc.html
- NAT44 ED CLI 参考：https://s3-docs.fd.io/vpp/23.02/cli-reference/clis/clicmd_src_plugins_nat_nat44-ed.html
- VPP NAT Wiki：https://wiki.fd.io/view/VPP/NAT

**项目内参考**：
- ACL 本地化规范：`specs/002-acl-localization/spec.md`
- ACL 实现代码：`internal/acl/server.go`, `internal/acl/common.go`
- NSE 架构：`main.go:210-248`

---

## 实施方案要点

### 架构决策
- **技术选型**：VPP NAT44 ED 官方实现
- **接口配置**：Interface A (server端) = inside, Interface B (client端) = outside
- **双向处理**：VPP 自动通过会话表处理反向转换（无需手动创建反向规则）

### 实施策略
- **框架复制**：完全参考 `internal/acl/` 的结构
- **微调部分**：API 调用改为 NAT44 接口，删除 src/dst 交换逻辑
- **大改部分**：无（NAT 比 ACL 更简单）

### 分阶段交付
- **P1**：基础 SNAT（最小可交付，1-2 周）
- **P2**：配置管理（运维灵活性）
- **P3**：模块本地化（长期维护）
- **P4**：静态端口映射（高级功能，可选）

---

## 验证标准

### 质量检查
- ✅ [specs/003-vpp-nat/checklists/requirements.md](../checklists/requirements.md) - 18/18 项通过
- ✅ 规范聚焦于功能需求，未涉及实现细节
- ✅ 所有功能需求都有明确的验收场景
- ✅ 成功标准可衡量（99% 成功率、1ms 延迟、1000 会话等）

### 技术验证
- ✅ ACL vs NAT 差异分析完成
- ✅ VPP NAT44 ED API 接口确认
- ✅ SFC 架构和接口角色确认
- ✅ 本地化策略明确

---

## 风险评估

### 已识别风险及缓解措施

1. **VPP NAT44 插件配置不当导致数据包丢失**
   - **缓解**：仔细研读 VPP 文档，测试环境充分验证，添加详细中文日志

2. **本地化模块导致依赖冲突**
   - **缓解**：严格遵循 ACL 本地化模式，使用 `go mod tidy` 验证

3. **静态映射与动态 SNAT 冲突**
   - **缓解**：配置验证检测端口冲突，VPP 自身已处理共存，添加测试用例

---

## 下一步行动

### 立即执行
1. **用户最终审核**：确认所有决策和实施方案
2. **执行 `/speckit.plan`**：生成详细的任务计划和工作分解

### 待确认的问题（来自实施方案）
1. ✅ 是否同意使用 VPP NAT44 ED 官方实现？
2. ✅ 是否同意删除 ACL 的双向规则模式？
3. ✅ 是否同意本地化策略（nat_types + nat44_ed）？
4. ✅ 是否同意分阶段实施（P1→P2→P3→P4）？
5. ✅ 是否有其他技术疑问或调整建议？

**用户已确认**："我同意你的文档，请修正spec。"

---

## 文档交付清单

- ✅ [spec.md](../spec.md) - 功能规范（已添加 Clarifications 章节）
- ✅ [implementation-approach.md](../implementation-approach.md) - 完整实施方案（8 章节）
- ✅ [checklists/requirements.md](../checklists/requirements.md) - 质量检查清单（18/18 通过）
- ✅ [.claude/clarification-summary.md](.claude/clarification-summary.md) - 本澄清总结（当前文件）

**状态**：✅ Clarification 阶段完成，准备进入 Planning 阶段

---

**生成时间**：2025-01-13
**下一阶段**：`/speckit.plan`
