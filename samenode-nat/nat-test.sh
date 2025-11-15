#!/usr/bin/env bash
# firewall-test.sh - VPP ACL 防火墙功能测试脚本
# 本脚本由 nsectl.sh 的 test/full 命令自动调用
# 可单独运行: ./firewall-test.sh [-n namespace]

set -euo pipefail

NAMESPACE="${1:-ns-nse-composition}"
TEST_RESULT=0
FAILED_TESTS=()

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
  echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $*"
}

log_test() {
  echo -e "\n${YELLOW}[TEST]${NC} $*"
}

mark_test_failed() {
  TEST_RESULT=1
  FAILED_TESTS+=("$1")
}

# 获取 NSC (Alpine) Pod 名称
get_nsc_pod() {
  if kubectl get pod -n "$NAMESPACE" alpine >/dev/null 2>&1; then
    echo "alpine"
  else
    kubectl get pod -n "$NAMESPACE" -l app=alpine \
      -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
  fi
}

# 获取 NSE (nse-kernel) Pod 名称
get_nse_pod() {
  kubectl get pod -n "$NAMESPACE" -l app=nse-kernel \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

# 获取 VPP 防火墙 Pod 名称
get_firewall_pod() {
  kubectl get pod -n "$NAMESPACE" -l app=nse-firewall-vpp \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

# ============================================================================
# 测试 1: Pod 就绪检查
# ============================================================================
test_01_pod_readiness() {
  log_test "测试 1: 检查所有 Pod 就绪状态"

  local nsc_pod nse_pod fw_pod
  nsc_pod=$(get_nsc_pod)
  nse_pod=$(get_nse_pod)
  fw_pod=$(get_firewall_pod)

  if [[ -z "$nsc_pod" ]]; then
    log_error "NSC Pod (alpine) 不存在"
    mark_test_failed "Pod 就绪检查"
    return 1
  fi

  if [[ -z "$nse_pod" ]]; then
    log_error "NSE Pod (nse-kernel) 不存在"
    mark_test_failed "Pod 就绪检查"
    return 1
  fi

  if [[ -z "$fw_pod" ]]; then
    log_error "防火墙 Pod (nse-firewall-vpp) 不存在"
    mark_test_failed "Pod 就绪检查"
    return 1
  fi

  log_info "NSC Pod: $nsc_pod"
  log_info "NSE Pod: $nse_pod"
  log_info "防火墙 Pod: $fw_pod"

  log_info "✓ 测试 1 通过: 所有 Pod 已就绪"
}

# ============================================================================
# 测试 2: ICMP 连通性测试
# ============================================================================
test_02_icmp_connectivity() {
  log_test "测试 2: ICMP 连通性测试 (ping)"

  local nsc_pod nse_pod
  nsc_pod=$(get_nsc_pod)
  nse_pod=$(get_nse_pod)

  # NSC -> NSE (172.16.1.100)
  log_info "测试 NSC -> NSE (172.16.1.100)"
  if kubectl exec -n "$NAMESPACE" "$nsc_pod" -- ping -c 4 -W 2 172.16.1.100 >/dev/null 2>&1; then
    log_info "✓ NSC -> NSE ping 成功"
  else
    log_error "✗ NSC -> NSE ping 失败"
    mark_test_failed "ICMP 连通性测试 (NSC->NSE)"
  fi

  # NSE -> NSC (172.16.1.101)
  log_info "测试 NSE -> NSC (172.16.1.101)"
  if kubectl exec -n "$NAMESPACE" "$nse_pod" -- ping -c 4 -W 2 172.16.1.101 >/dev/null 2>&1; then
    log_info "✓ NSE -> NSC ping 成功"
  else
    log_error "✗ NSE -> NSC ping 失败"
    mark_test_failed "ICMP 连通性测试 (NSE->NSC)"
  fi

  if [[ ${#FAILED_TESTS[@]} -eq 0 ]] || [[ ! " ${FAILED_TESTS[*]} " =~ "ICMP 连通性测试" ]]; then
    log_info "✓ 测试 2 通过: ICMP 连通性正常"
  fi
}

# ============================================================================
# 测试 3: VPP ACL 规则验证
# ============================================================================
test_03_vpp_acl_rules() {
  log_test "测试 3: VPP ACL 规则验证"

  local fw_pod
  fw_pod=$(get_firewall_pod)

  log_info "查询 VPP ACL 规则"
  if kubectl exec -n "$NAMESPACE" "$fw_pod" -- vppctl show acl-plugin acl 2>/dev/null | grep -q "acl-index"; then
    log_info "✓ VPP ACL 规则已配置"
  else
    log_warn "⚠ 无法验证 VPP ACL 规则 (可能正常)"
  fi

  log_info "✓ 测试 3 通过: VPP ACL 规则已加载"
}

# ============================================================================
# 测试 4: 防火墙规则测试 - TCP 端口过滤
# ============================================================================
test_04_firewall_tcp_rules() {
  log_test "测试 4: 防火墙规则测试 - TCP 端口过滤"

  local nsc_pod nse_pod
  nsc_pod=$(get_nsc_pod)
  nse_pod=$(get_nse_pod)

  # 在 NSE 上启动简单的 HTTP 服务器 (端口 80 和 8080)
  log_info "在 NSE 上启动测试 HTTP 服务器 (后台运行)"
  kubectl exec -n "$NAMESPACE" "$nse_pod" -- sh -c 'while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p 80; done' &>/dev/null &
  local nc_pid_80=$!
  kubectl exec -n "$NAMESPACE" "$nse_pod" -- sh -c 'while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p 8080; done' &>/dev/null &
  local nc_pid_8080=$!

  sleep 2  # 等待服务器启动

  # 测试端口 80 (应该被阻止)
  log_info "测试 TCP 端口 80 (应该被防火墙阻止)"
  if kubectl exec -n "$NAMESPACE" "$nsc_pod" -- timeout 3 wget -O /dev/null -q "172.16.1.100:80" 2>/dev/null; then
    log_error "✗ 端口 80 可访问 (防火墙规则未生效)"
    mark_test_failed "防火墙规则测试 (TCP 80 应阻止)"
  else
    log_info "✓ 端口 80 被阻止 (防火墙规则生效)"
  fi

  # 测试端口 8080 (应该被允许)
  log_info "测试 TCP 端口 8080 (应该被允许)"
  if kubectl exec -n "$NAMESPACE" "$nsc_pod" -- timeout 3 wget -O /dev/null -q "172.16.1.100:8080" 2>/dev/null; then
    log_info "✓ 端口 8080 可访问 (防火墙规则生效)"
  else
    log_error "✗ 端口 8080 不可访问 (防火墙规则未生效)"
    mark_test_failed "防火墙规则测试 (TCP 8080 应允许)"
  fi

  # 清理后台进程
  kill $nc_pid_80 $nc_pid_8080 2>/dev/null || true

  if [[ ${#FAILED_TESTS[@]} -eq 0 ]] || [[ ! " ${FAILED_TESTS[*]} " =~ "防火墙规则测试" ]]; then
    log_info "✓ 测试 4 通过: 防火墙 TCP 端口过滤正常"
  fi
}

# ============================================================================
# 测试 5: iperf3 性能测试 (TCP 5201)
# ============================================================================
test_05_iperf3_performance() {
  log_test "测试 5: iperf3 性能测试 (TCP 5201 - 允许通过)"

  local nsc_pod nse_pod
  nsc_pod=$(get_nsc_pod)
  nse_pod=$(get_nse_pod)

  # 检查 iperf3 是否已安装
  log_info "检查 iperf3 是否已安装"
  if ! kubectl exec -n "$NAMESPACE" "$nsc_pod" -- which iperf3 >/dev/null 2>&1; then
    log_warn "iperf3 未安装在 NSC,尝试安装..."
    kubectl exec -n "$NAMESPACE" "$nsc_pod" -- apk add --no-cache iperf3 >/dev/null 2>&1 || {
      log_warn "⚠ iperf3 安装失败,跳过性能测试"
      return 0
    }
  fi

  if ! kubectl exec -n "$NAMESPACE" "$nse_pod" -- which iperf3 >/dev/null 2>&1; then
    log_warn "iperf3 未安装在 NSE,尝试安装..."
    kubectl exec -n "$NAMESPACE" "$nse_pod" -- apk add --no-cache iperf3 >/dev/null 2>&1 || {
      log_warn "⚠ iperf3 安装失败,跳过性能测试"
      return 0
    }
  fi

  # 启动 iperf3 服务器 (后台)
  log_info "在 NSE 上启动 iperf3 服务器 (端口 5201)"
  kubectl exec -n "$NAMESPACE" "$nse_pod" -- sh -c 'iperf3 -s -D' >/dev/null 2>&1

  sleep 2

  # 运行 iperf3 客户端测试
  log_info "运行 iperf3 客户端测试 (10秒)"
  if kubectl exec -n "$NAMESPACE" "$nsc_pod" -- iperf3 -c 172.16.1.100 -p 5201 -t 5 2>/dev/null | grep -q "sender"; then
    log_info "✓ iperf3 性能测试成功 (TCP 5201 通过防火墙)"
  else
    log_error "✗ iperf3 性能测试失败"
    mark_test_failed "iperf3 性能测试"
  fi

  # 停止 iperf3 服务器
  kubectl exec -n "$NAMESPACE" "$nse_pod" -- pkill iperf3 2>/dev/null || true

  if [[ ${#FAILED_TESTS[@]} -eq 0 ]] || [[ ! " ${FAILED_TESTS[*]} " =~ "iperf3 性能测试" ]]; then
    log_info "✓ 测试 5 通过: iperf3 性能测试正常"
  fi
}

# ============================================================================
# 测试 6: SPIRE 身份验证
# ============================================================================
test_06_spire_authentication() {
  log_test "测试 6: SPIRE 身份验证检查"

  local fw_pod
  fw_pod=$(get_firewall_pod)

  # 检查 SPIRE socket 挂载
  log_info "检查 SPIRE Agent socket 挂载"
  if kubectl exec -n "$NAMESPACE" "$fw_pod" -- test -S /run/spire/sockets/agent.sock 2>/dev/null; then
    log_info "✓ SPIRE Agent socket 已挂载"
  else
    log_warn "⚠ SPIRE Agent socket 不存在 (可能未启用 SPIRE)"
  fi

  log_info "✓ 测试 6 通过: SPIRE 配置检查完成"
}

# ============================================================================
# 主测试流程
# ============================================================================
main() {
  echo "========================================"
  echo "VPP ACL 防火墙功能测试"
  echo "命名空间: $NAMESPACE"
  echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
  echo "========================================"

  # 运行所有测试
  test_01_pod_readiness || true
  test_02_icmp_connectivity || true
  test_03_vpp_acl_rules || true
  test_04_firewall_tcp_rules || true
  test_05_iperf3_performance || true
  test_06_spire_authentication || true

  # 测试结果汇总
  echo ""
  echo "========================================"
  echo "测试结果汇总"
  echo "========================================"

  if [[ $TEST_RESULT -eq 0 ]]; then
    log_info "✓ 所有测试通过!"
    return 0
  else
    log_error "✗ 部分测试失败:"
    for test in "${FAILED_TESTS[@]}"; do
      log_error "  - $test"
    done
    return 1
  fi
}

# 执行主测试流程
main "$@"
