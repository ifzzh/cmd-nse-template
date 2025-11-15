# NSE 防火墙测试指南 / Firewall Testing Guide

本文档说明如何使用 `nsectl.sh` 和 `firewall-test.sh` 进行 VPP ACL 防火墙功能测试。

## 快速开始 / Quick Start

### 方法 1: 使用 nsectl.sh 测试命令 (推荐)

```bash
cd samenode-firewall

# 单独运行测试
./nsectl.sh test

# 完整流程 (部署 -> 等待就绪 -> 测试 -> 收集日志 -> 推送)
./nsectl.sh full
```

### 方法 2: 直接运行测试脚本

```bash
cd samenode-firewall

# 使用默认命名空间
./firewall-test.sh

# 指定命名空间
./firewall-test.sh my-custom-namespace
```

## 测试覆盖 / Test Coverage

`firewall-test.sh` 包含以下测试用例:

| 测试 | 描述 | 验证内容 |
|------|------|---------|
| **测试 1** | Pod 就绪检查 | NSC, NSE, 防火墙 Pod 都已就绪 |
| **测试 2** | ICMP 连通性测试 | NSC ↔ NSE ping 双向连通 |
| **测试 3** | VPP ACL 规则验证 | VPP ACL 规则已加载 |
| **测试 4** | 防火墙规则测试 | TCP 端口过滤 (80 阻止, 8080 允许) |
| **测试 5** | iperf3 性能测试 | TCP 5201 通过防火墙,性能正常 |
| **测试 6** | SPIRE 身份验证 | SPIRE Agent socket 已挂载 |

## nsectl.sh 命令说明 / Command Reference

### 基本命令

```bash
# 部署 NSE
./nsectl.sh apply

# 查看 Pod 状态
./nsectl.sh get

# 实时监控 Pod (需要 watch 命令)
./nsectl.sh watch

# 运行功能测试
./nsectl.sh test

# 收集日志
./nsectl.sh logs

# 查看 Pod 详情
./nsectl.sh describe

# 删除命名空间
./nsectl.sh delete

# 完整流程 (推荐用于远程环境测试)
./nsectl.sh full
```

### 高级选项

```bash
# 指定命名空间
./nsectl.sh -n my-namespace test

# 指定 kustomize 目录
./nsectl.sh -k ../other-deployment apply

# 指定 app 标签 (用于多 NSE 环境)
./nsectl.sh -a nse-firewall-vpp test

# 组合使用
./nsectl.sh -n test-ns -a my-firewall test
```

## 测试输出示例 / Test Output Example

```bash
$ ./nsectl.sh test
Kube context: kubernetes-admin@kubernetes
Namespace:    ns-nse-composition
App label:    app=nse-firewall-vpp
运行 NSE 特定测试脚本: /path/to/firewall-test.sh
测试命名空间: ns-nse-composition

========================================
VPP ACL 防火墙功能测试
命名空间: ns-nse-composition
时间: 2025-11-13 12:00:00
========================================

[TEST] 测试 1: 检查所有 Pod 就绪状态
[INFO] NSC Pod: alpine
[INFO] NSE Pod: nse-kernel-abc123
[INFO] 防火墙 Pod: nse-firewall-vpp-def456
[INFO] ✓ 测试 1 通过: 所有 Pod 已就绪

[TEST] 测试 2: ICMP 连通性测试 (ping)
[INFO] 测试 NSC -> NSE (172.16.1.100)
[INFO] ✓ NSC -> NSE ping 成功
[INFO] 测试 NSE -> NSC (172.16.1.101)
[INFO] ✓ NSE -> NSC ping 成功
[INFO] ✓ 测试 2 通过: ICMP 连通性正常

[TEST] 测试 3: VPP ACL 规则验证
[INFO] 查询 VPP ACL 规则
[INFO] ✓ VPP ACL 规则已配置
[INFO] ✓ 测试 3 通过: VPP ACL 规则已加载

[TEST] 测试 4: 防火墙规则测试 - TCP 端口过滤
[INFO] 在 NSE 上启动测试 HTTP 服务器 (后台运行)
[INFO] 测试 TCP 端口 80 (应该被防火墙阻止)
[INFO] ✓ 端口 80 被阻止 (防火墙规则生效)
[INFO] 测试 TCP 端口 8080 (应该被允许)
[INFO] ✓ 端口 8080 可访问 (防火墙规则生效)
[INFO] ✓ 测试 4 通过: 防火墙 TCP 端口过滤正常

[TEST] 测试 5: iperf3 性能测试 (TCP 5201 - 允许通过)
[INFO] 检查 iperf3 是否已安装
[INFO] 在 NSE 上启动 iperf3 服务器 (端口 5201)
[INFO] 运行 iperf3 客户端测试 (10秒)
[INFO] ✓ iperf3 性能测试成功 (TCP 5201 通过防火墙)
[INFO] ✓ 测试 5 通过: iperf3 性能测试正常

[TEST] 测试 6: SPIRE 身份验证检查
[INFO] 检查 SPIRE Agent socket 挂载
[INFO] ✓ SPIRE Agent socket 已挂载
[INFO] ✓ 测试 6 通过: SPIRE 配置检查完成

========================================
测试结果汇总
========================================
[INFO] ✓ 所有测试通过!
```

## 测试脚本架构 / Test Script Architecture

### 设计原则

1. **通用性**: `nsectl.sh` 与 NSE 类型无关,可用于任何 NSE
2. **动态性**: 自动从目录名推断 NSE 类型 (例如: `samenode-firewall` → `firewall`)
3. **可扩展性**: 每个 NSE 目录提供自己的 `<nse-type>-test.sh` 脚本
4. **一致性**: 所有测试脚本遵循相同的命令行接口 (接受命名空间参数)

### 目录结构

```
samenode-firewall/
├── nsectl.sh              # 通用控制脚本
├── firewall-test.sh       # 防火墙特定测试 (可执行)
├── kustomization.yaml
├── client.yaml
├── server.yaml
└── logs/
    ├── cmd-nsc-init.log
    ├── nse-firewall-vpp.log
    └── cmdline.log
```

### 添加新的 NSE 测试脚本

如果你有其他类型的 NSE (例如: NAT, VPN, Proxy),只需:

1. 创建目录: `samenode-nat/`
2. 复制 `nsectl.sh` 到新目录
3. 创建测试脚本: `samenode-nat/nat-test.sh`
4. 赋予执行权限: `chmod +x nat-test.sh`
5. 运行测试: `./nsectl.sh test`

`nsectl.sh` 会自动识别并调用 `nat-test.sh`!

## 故障排查 / Troubleshooting

### 问题 1: 测试脚本未找到

**错误**:
```
未找到测试脚本: /path/to/firewall-test.sh
```

**解决方案**:
```bash
# 检查文件是否存在
ls -la samenode-firewall/firewall-test.sh

# 如果不存在,创建它
touch samenode-firewall/firewall-test.sh
chmod +x samenode-firewall/firewall-test.sh
```

### 问题 2: 测试失败 - Pod 未就绪

**错误**:
```
[ERROR] NSC Pod (alpine) 不存在
```

**解决方案**:
```bash
# 检查 Pod 状态
kubectl get pod -n ns-nse-composition

# 如果 Pod 不存在,先部署
./nsectl.sh apply

# 等待 Pod 就绪
kubectl wait --for=condition=Ready pod -l app=alpine -n ns-nse-composition --timeout=180s
```

### 问题 3: 测试失败 - 端口访问异常

**错误**:
```
[ERROR] ✗ 端口 80 可访问 (防火墙规则未生效)
```

**解决方案**:
```bash
# 1. 检查 VPP ACL 规则
kubectl exec -n ns-nse-composition deploy/nse-firewall-vpp -- vppctl show acl-plugin acl

# 2. 检查 ConfigMap 配置
kubectl get cm firewall-config-file -n ns-nse-composition -o yaml

# 3. 重新部署防火墙
kubectl delete pod -n ns-nse-composition -l app=nse-firewall-vpp
./nsectl.sh apply
```

### 问题 4: iperf3 未安装

**警告**:
```
[WARN] ⚠ iperf3 安装失败,跳过性能测试
```

**解决方案**:
```bash
# 手动安装 iperf3
kubectl exec -n ns-nse-composition alpine -- apk add --no-cache iperf3
kubectl exec -n ns-nse-composition deploy/nse-kernel -- apk add --no-cache iperf3

# 重新运行测试
./nsectl.sh test
```

## CI/CD 集成 / CI/CD Integration

### GitHub Actions 示例

```yaml
name: NSE Firewall Tests

on:
  push:
    branches: [ 002-acl-localization ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Kubernetes
        uses: engineerd/setup-kind@v0.5.0

      - name: Deploy NSE
        run: |
          cd samenode-firewall
          ./nsectl.sh apply

      - name: Run Tests
        run: |
          cd samenode-firewall
          ./nsectl.sh test

      - name: Collect Logs
        if: always()
        run: |
          cd samenode-firewall
          ./nsectl.sh logs

      - name: Upload Logs
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-logs
          path: samenode-firewall/logs/
```

### Jenkins Pipeline 示例

```groovy
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                sh 'cd samenode-firewall && ./nsectl.sh apply'
            }
        }
        stage('Test') {
            steps {
                sh 'cd samenode-firewall && ./nsectl.sh test'
            }
        }
        stage('Collect Logs') {
            steps {
                sh 'cd samenode-firewall && ./nsectl.sh logs'
                archiveArtifacts artifacts: 'samenode-firewall/logs/**', allowEmptyArchive: true
            }
        }
    }
    post {
        always {
            sh 'cd samenode-firewall && ./nsectl.sh delete || true'
        }
    }
}
```

## 最佳实践 / Best Practices

1. **自动化测试**: 在 CI/CD 流水线中使用 `./nsectl.sh test`
2. **测试隔离**: 为每个测试运行使用独立的命名空间
3. **日志保存**: 测试失败时使用 `./nsectl.sh logs` 收集诊断信息
4. **完整流程**: 在远程环境使用 `./nsectl.sh full` 一键部署和测试
5. **定期清理**: 测试完成后使用 `./nsectl.sh delete` 清理资源

## 相关文档 / Related Documentation

- [README.md](README.md) - NSE 防火墙部署指南
- [config-file.yaml](config-file.yaml) - ACL 防火墙规则配置
- [../../README.md](../../README.md) - 项目总览和技术栈
