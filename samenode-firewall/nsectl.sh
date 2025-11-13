#!/usr/bin/env bash
set -euo pipefail

# NSE 通用控制脚本 / NSE Generic Control Script
# 通过 NF_TYPE 变量自动推断 NSE 类型并生成相关配置
#
# Usage examples:
#   ./nsectl.sh apply
#   ./nsectl.sh watch
#   ./nsectl.sh test
#   ./nsectl.sh full
#
# Options:
#   -n|--namespace <ns>    Override namespace (default: ns-nse-composition)
#   -k|--kustomize <dir>   Kustomize dir for apply (default: .)
#   -w|--watch-interval N  watch interval seconds (default: 2)
#   -t|--nf-type <type>    Network Function type (auto-detected from directory name)
#   -h|--help              Show help

# ============================================================================
# 核心配置: NF_TYPE (网络功能类型)
# 从目录名自动推断 (例如: samenode-firewall -> firewall)
# 可通过 -t|--nf-type 参数覆盖
# ============================================================================
script_dir_init() {
  cd "$(dirname "$0")" && pwd
}

NF_TYPE="${NF_TYPE:-}"  # 允许环境变量覆盖
if [[ -z "$NF_TYPE" ]]; then
  # 从目录名自动推断 NF 类型 (提取最后一个 - 后的部分)
  local_dir=$(basename "$(script_dir_init)")
  NF_TYPE="${local_dir##*-}"
fi

# ============================================================================
# 基于 NF_TYPE 自动生成的派生变量
# ============================================================================
NAMESPACE="ns-nse-composition"
KUSTOMIZE_DIR="."
WATCH_INTERVAL=2

# APP_LABEL: nse-{NF_TYPE}-vpp (例如: nse-firewall-vpp, nse-nat-vpp)
APP_LABEL="nse-${NF_TYPE}-vpp"

# LOG_FILE_PREFIX: 日志文件前缀 (例如: nse-firewall-vpp, nse-nat-vpp)
LOG_FILE_PREFIX="nse-${NF_TYPE}-vpp"

# TEST_SCRIPT_NAME: 测试脚本名称 (例如: firewall-test.sh, nat-test.sh)
TEST_SCRIPT_NAME="${NF_TYPE}-test.sh"

# POD_DISPLAY_NAME: Pod 显示名称 (例如: 防火墙, NAT)
case "$NF_TYPE" in
  firewall) POD_DISPLAY_NAME="防火墙 / Firewall" ;;
  nat)      POD_DISPLAY_NAME="NAT" ;;
  vpn)      POD_DISPLAY_NAME="VPN" ;;
  proxy)    POD_DISPLAY_NAME="代理 / Proxy" ;;
  *)        POD_DISPLAY_NAME="$NF_TYPE" ;;
esac

# Parse flags before subcommand
ACTION="help"
ARGS=("$@")
while [[ ${#ARGS[@]} -gt 0 ]]; do
  case "${ARGS[0]}" in
    -n|--namespace)
      NAMESPACE="${ARGS[1]:-}"; ARGS=(${ARGS[@]:2}) ;;
    -k|--kustomize)
      KUSTOMIZE_DIR="${ARGS[1]:-}"; ARGS=(${ARGS[@]:2}) ;;
    -w|--watch-interval)
      WATCH_INTERVAL="${ARGS[1]:-}"; ARGS=(${ARGS[@]:2}) ;;
    -t|--nf-type)
      # 覆盖 NF_TYPE 并重新生成派生变量
      NF_TYPE="${ARGS[1]:-}"; ARGS=(${ARGS[@]:2})
      APP_LABEL="nse-${NF_TYPE}-vpp"
      LOG_FILE_PREFIX="nse-${NF_TYPE}-vpp"
      TEST_SCRIPT_NAME="${NF_TYPE}-test.sh"
      case "$NF_TYPE" in
        firewall) POD_DISPLAY_NAME="防火墙 / Firewall" ;;
        nat)      POD_DISPLAY_NAME="NAT" ;;
        vpn)      POD_DISPLAY_NAME="VPN" ;;
        proxy)    POD_DISPLAY_NAME="代理 / Proxy" ;;
        *)        POD_DISPLAY_NAME="$NF_TYPE" ;;
      esac
      ;;
    -a|--app-label)
      # 保留此选项用于完全自定义 APP_LABEL (覆盖自动生成)
      APP_LABEL="${ARGS[1]:-}"; ARGS=(${ARGS[@]:2}) ;;
    -h|--help)
      ACTION="help"; ARGS=(${ARGS[@]:1}); break ;;
    apply|get|watch|logs|describe|delete|test|full|help)
      ACTION="${ARGS[0]}"; ARGS=(${ARGS[@]:1}); break ;;
    *)
      echo "未知选项或命令: ${ARGS[0]} / Unknown option or action: ${ARGS[0]}" >&2
      ACTION="help"; break ;;
  esac
done

script_dir() {
  cd "$(dirname "$0")" && pwd
}

ensure_logs_dir() {
  mkdir -p "$(script_dir)/logs"
}

# Print and execute a command safely (shell-escaped preview)
run() {
  printf '+ '
  printf '%q ' "$@"
  echo
  "$@"
}

get_vpp_pod() {
  kubectl get pod -n "$NAMESPACE" -l app="$APP_LABEL" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

get_alpine_pod() {
  # The Alpine pod name is static in manifests; fall back to label if changed later
  if kubectl get pod -n "$NAMESPACE" alpine >/dev/null 2>&1; then
    echo "alpine"
  else
    kubectl get pod -n "$NAMESPACE" -l app=alpine \
      -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
  fi
}

print_context() {
  echo "=========================================="
  echo "网络功能类型 / NF Type:          $NF_TYPE ($POD_DISPLAY_NAME)"
  echo "Kubernetes 上下文 / Kube context: $(kubectl config current-context 2>/dev/null || echo unknown)"
  echo "命名空间 / Namespace:            $NAMESPACE"
  echo "应用标签 / App label:            app=$APP_LABEL"
  echo "=========================================="
}

cmd_apply() {
  print_context
  echo "从 '$KUSTOMIZE_DIR' 应用 kustomize 配置 / Applying kustomize from '$KUSTOMIZE_DIR'..."
  run kubectl apply -k "$KUSTOMIZE_DIR"
}

cmd_get() {
  print_context
  run kubectl get pod -n "$NAMESPACE" -o wide
}

cmd_watch() {
  print_context
  if command -v watch >/dev/null 2>&1; then
    echo "+ watch -n $WATCH_INTERVAL kubectl get pod -n $NAMESPACE -o wide"
    watch -n "$WATCH_INTERVAL" "kubectl get pod -n $NAMESPACE -o wide"
  else
    echo "'watch' not found; falling back to kubectl --watch (Ctrl+C to stop)" >&2
    echo "+ kubectl get pod -n $NAMESPACE -o wide --watch"
    kubectl get pod -n "$NAMESPACE" -o wide --watch
  fi
}

cmd_logs() {
  print_context
  ensure_logs_dir
  # 将日志写入固定文件名（不带时间戳）

  local alpine_pod
  alpine_pod=$(get_alpine_pod)
  if [[ -n "$alpine_pod" ]]; then
    echo "保存 $alpine_pod 的日志 (容器: cmd-nsc-init) / Saving logs for $alpine_pod (container: cmd-nsc-init)"
    run kubectl logs -n "$NAMESPACE" "$alpine_pod" -c cmd-nsc-init \
      > "$(script_dir)/logs/cmd-nsc-init.log" || true
  else
    echo "警告: 未找到 Alpine pod,跳过 cmd-nsc-init 日志 / WARN: Alpine pod not found; skipping cmd-nsc-init logs" >&2
  fi

  local vpp_pod
  vpp_pod=$(get_vpp_pod)
  if [[ -n "$vpp_pod" ]]; then
    echo "保存 $vpp_pod 的日志 / Saving logs for $vpp_pod"
    run kubectl logs -n "$NAMESPACE" "$vpp_pod" \
      > "$(script_dir)/logs/${LOG_FILE_PREFIX}.log" || true
  else
    echo "警告: 未找到 $APP_LABEL pod,跳过 ${LOG_FILE_PREFIX} 日志 / WARN: $APP_LABEL pod not found; skipping ${LOG_FILE_PREFIX} logs" >&2
  fi

  echo "日志已写入 / Logs written to $(script_dir)/logs/"
}

# 等待带有 app 标签的 Pod 出现(最多 60 秒),然后等待就绪(最多 180 秒)
wait_for_app_pods_ready() {
  local exist_timeout=60
  local start_ts=$(date +%s)
  echo "等待带有标签 app=$APP_LABEL 的 Pod 出现 (超时: ${exist_timeout}秒) / Waiting for pods with label app=$APP_LABEL to appear (timeout: ${exist_timeout}s)..."
  while true; do
    local count
    count=$(kubectl get pod -n "$NAMESPACE" -l app="$APP_LABEL" --no-headers 2>/dev/null | wc -l || true)
    if [[ "$count" -gt 0 ]]; then
      echo "找到 $count 个 Pod (app=$APP_LABEL),等待就绪 (超时: 180秒) / Found $count pod(s) with app=$APP_LABEL. Waiting for Ready (timeout: 180s)..."
      break
    fi
    if (( $(date +%s) - start_ts >= exist_timeout )); then
      echo "警告: ${exist_timeout}秒内未发现 app=$APP_LABEL 的 Pod / WARN: No pods with app=$APP_LABEL appeared within ${exist_timeout}s" >&2
      break
    fi
    sleep 2
  done

  # 如果 Pod 存在,等待就绪
  if kubectl get pod -n "$NAMESPACE" -l app="$APP_LABEL" --no-headers >/dev/null 2>&1; then
    kubectl wait --for=condition=Ready pod -l app="$APP_LABEL" -n "$NAMESPACE" --timeout=180s || {
      echo "警告: kubectl wait 未成功完成 (app=$APP_LABEL) / WARN: kubectl wait did not complete successfully for app=$APP_LABEL" >&2
    }
  fi
}

# 动态调用 NSE 特定的测试脚本
# 基于 NF_TYPE 自动查找测试脚本 (例如: firewall-test.sh, nat-test.sh)
cmd_test() {
  print_context
  local SD
  SD="$(script_dir)"

  # 构建测试脚本路径 (使用 TEST_SCRIPT_NAME 变量)
  local test_script="$SD/${TEST_SCRIPT_NAME}"

  if [[ -f "$test_script" && -x "$test_script" ]]; then
    echo "运行 ${POD_DISPLAY_NAME} 测试脚本 / Running ${POD_DISPLAY_NAME} test script: $test_script"
    echo "测试命名空间 / Test namespace: $NAMESPACE"
    echo ""
    "$test_script" "$NAMESPACE"
  else
    echo "未找到测试脚本 / Test script not found: $test_script" >&2
    echo "请创建可执行的测试脚本 / Please create an executable test script:" >&2
    echo "  touch $test_script" >&2
    echo "  chmod +x $test_script" >&2
    echo ""
    echo "提示 / Hint: 测试脚本命名规则 / Test script naming: ${NF_TYPE}-test.sh" >&2
    exit 1
  fi
}

cmd_full() {
  ensure_logs_dir
  local SD
  SD="$(script_dir)"
  local REPO_ROOT
  REPO_ROOT="$SD/.."
  local cmdline_log="$SD/logs/cmdline.log"
  # 将所有输出捕获到 cmdline.log 并同时显示在屏幕上
  {
    echo "=== 拉取最新代码 / GIT PULL (repo root) ==="
    (
      cd "$REPO_ROOT" && {
        echo "+ cd $REPO_ROOT && git pull"
        git pull || echo "警告: git pull 失败 (继续执行) / WARN: git pull failed (continuing)" >&2
      }
    )
    echo

    echo "=== 部署应用 / APPLY ==="
    print_context
    echo "从 '$KUSTOMIZE_DIR' 应用 kustomize 配置 / Applying kustomize from '$KUSTOMIZE_DIR'..."
    kubectl apply -k "$KUSTOMIZE_DIR"
    echo

    echo "=== 等待 Pod 就绪 / WAIT READY (app=$APP_LABEL) ==="
    wait_for_app_pods_ready
    echo

    echo "当前 Pod 状态 / Current pods:"
    kubectl get pod -n "$NAMESPACE" -o wide || true
    echo

    echo "=== 等待 10 秒 / SLEEP 10s ==="
    sleep 10
    echo

    echo "当前 Pod 状态 / Current pods:"
    kubectl get pod -n "$NAMESPACE" -o wide || true
    echo

    echo "=== 运行功能测试 / RUN TESTS ==="
    cmd_test || echo "警告: 测试失败或不可用 / WARN: Tests failed or not available" >&2
    echo

    echo "=== 收集日志 / COLLECT LOGS ==="
    cmd_logs
    echo

    echo "=== 查看详情 / DESCRIBE (app=$APP_LABEL) ==="
    cmd_describe || true
    echo

    echo "=== 提交并推送日志 / GIT COMMIT & PUSH LOGS (repo root) ==="
    (
      cd "$REPO_ROOT" && {
        echo "+ cd $REPO_ROOT && git add ."
        git add .
        if ! git diff --cached --quiet; then
          echo "+ git commit -m '推送logs'"
          git commit -m "推送logs" || echo "警告: git commit 失败 / WARN: git commit failed" >&2
          echo "+ git push"
          git push || echo "警告: git push 失败 / WARN: git push failed" >&2
        else
          echo "没有变更需要提交 / No changes to commit."
        fi
      }
    )
    echo

    echo "=== 完成 / DONE ==="
  } > >(tee "$cmdline_log") 2>&1

  echo "完整流程执行完毕,命令行日志已保存到 $cmdline_log"
  echo "Full flow complete. Cmdline log saved to $cmdline_log"
}

cmd_describe() {
  print_context
  local vpp_pod
  vpp_pod=$(get_vpp_pod)
  if [[ -z "$vpp_pod" ]]; then
    echo "在命名空间 '$NAMESPACE' 中未找到 ${POD_DISPLAY_NAME} pod (app=$APP_LABEL) / ${POD_DISPLAY_NAME} pod not found in namespace '$NAMESPACE' (app=$APP_LABEL)" >&2
    exit 1
  fi
  run kubectl describe pod -n "$NAMESPACE" "$vpp_pod"
}

cmd_delete() {
  print_context
  read -r -p "删除命名空间 '$NAMESPACE'? / Delete namespace '$NAMESPACE'? [y/N] " ans
  if [[ "$ans" =~ ^[Yy]$ ]]; then
    run kubectl delete ns "$NAMESPACE"
    run clear
  else
    echo "已取消 / Aborted."
  fi
}

cmd_help() {
  cat <<EOF
========================================
NSE 通用控制脚本 / NSE Generic Control Script
当前 NF 类型 / Current NF Type: $NF_TYPE ($POD_DISPLAY_NAME)
========================================

使用方法 / Usage: nsectl.sh [options] <action>

命令 / Actions:
  apply        应用 kustomize 配置 / kubectl apply -k <dir> (default: .)
  get          查看命名空间中的 Pod / kubectl get pods in namespace
  watch        实时监控 Pod 状态 / watch kubectl get pods (requires 'watch')
  logs         收集日志到 ./logs/ / collect logs to ./logs/
  describe     查看 ${POD_DISPLAY_NAME} pod 详情 / describe the ${POD_DISPLAY_NAME} pod
  test         运行 ${POD_DISPLAY_NAME} 测试 / run ${POD_DISPLAY_NAME} tests (${TEST_SCRIPT_NAME})
  full         完整流程 / run: apply -> wait Ready -> sleep 10s -> test -> logs -> describe -> git push
  delete       删除命名空间 / delete the namespace
  help         显示此帮助信息 / show this message

选项 / Options:
  -n, --namespace <ns>      命名空间 / namespace (default: ns-nse-composition)
  -k, --kustomize <dir>     kustomize 目录 / kustomize dir for apply (default: .)
  -w, --watch-interval <n>  监控刷新间隔(秒) / watch refresh interval seconds (default: 2)
  -t, --nf-type <type>      网络功能类型 / Network Function type (default: auto-detect from directory)
  -a, --app-label <value>   应用标签 / app label (default: nse-\${NF_TYPE}-vpp, current: $APP_LABEL)
  -h, --help                显示帮助 / show help

示例 / Examples:
  ./nsectl.sh apply                        # 部署应用
  ./nsectl.sh watch                        # 实时监控
  ./nsectl.sh get                          # 查看 Pod
  ./nsectl.sh test                         # 运行测试
  ./nsectl.sh logs                         # 收集日志
  ./nsectl.sh describe                     # 查看详情
  ./nsectl.sh full                         # 完整流程(推荐)
  ./nsectl.sh delete                       # 删除资源
  ./nsectl.sh -n my-namespace test         # 指定命名空间测试
  ./nsectl.sh -t nat test                  # 覆盖 NF 类型为 NAT 并运行测试

========================================
NF_TYPE 变量配置 / NF_TYPE Variable Configuration:
========================================

NF_TYPE 是核心配置变量,自动推断或手动指定:
- 自动推断: 从目录名提取 (例如: samenode-firewall -> firewall)
- 手动指定: 使用 -t|--nf-type 选项
- 环境变量: export NF_TYPE=nat

基于 NF_TYPE 自动生成的派生变量:
- APP_LABEL        = nse-\${NF_TYPE}-vpp     (当前: $APP_LABEL)
- LOG_FILE_PREFIX  = nse-\${NF_TYPE}-vpp     (当前: $LOG_FILE_PREFIX)
- TEST_SCRIPT_NAME = \${NF_TYPE}-test.sh     (当前: $TEST_SCRIPT_NAME)
- POD_DISPLAY_NAME = 根据 NF_TYPE 映射      (当前: $POD_DISPLAY_NAME)

支持的 NF 类型 / Supported NF Types:
- firewall : 防火墙 / Firewall
- nat      : NAT
- vpn      : VPN
- proxy    : 代理 / Proxy
- 其他     : 使用 NF_TYPE 原值

========================================
测试脚本约定 / Test Script Convention:
========================================

测试脚本命名规则: \${NF_TYPE}-test.sh
示例:
- firewall-test.sh (当前目录: 防火墙测试)
- nat-test.sh      (NAT 测试)
- vpn-test.sh      (VPN 测试)

这允许不同的 NSE 类型拥有各自的测试脚本,实现通用控制脚本。
EOF
}

case "$ACTION" in
  apply)    cmd_apply ;;
  get)      cmd_get ;;
  watch)    cmd_watch ;;
  logs)     cmd_logs ;;
  describe) cmd_describe ;;
  test)     cmd_test ;;
  full)     cmd_full ;;
  delete)   cmd_delete ;;
  help|*)   cmd_help ;;
esac
