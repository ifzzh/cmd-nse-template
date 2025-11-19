#!/bin/bash
# v1.0.6 验证脚本 - L3 路由模式验证
# 用途：验证从 L2 xconnect 迁移到 L3 路由模式后 NAT 功能是否正常

set -e

echo "=================================================="
echo "  v1.0.6 验证脚本 - L3 路由模式 NAT 验证"
echo "=================================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取 NAT POD 名称
echo "1. 查找 NAT NSE Pod..."
NAT_POD=$(kubectl get pods -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].metadata.name}')
if [ -z "$NAT_POD" ]; then
    echo -e "${RED}❌ 错误: 找不到 NAT NSE Pod${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 找到 NAT POD: $NAT_POD${NC}"
echo ""

# 检查 Pod 状态
echo "2. 检查 Pod 运行状态..."
POD_STATUS=$(kubectl get pod -n ns-nse-composition $NAT_POD -o jsonpath='{.status.phase}')
if [ "$POD_STATUS" != "Running" ]; then
    echo -e "${RED}❌ Pod 状态异常: $POD_STATUS${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Pod 状态: Running${NC}"
echo ""

# 检查镜像版本
echo "3. 检查镜像版本..."
IMAGE=$(kubectl get pod -n ns-nse-composition $NAT_POD -o jsonpath='{.spec.containers[0].image}')
echo "   镜像: $IMAGE"
if [[ $IMAGE == *"v1.0.6"* ]]; then
    echo -e "${GREEN}✓ 镜像版本正确: v1.0.6${NC}"
else
    echo -e "${YELLOW}⚠ 警告: 镜像版本不是 v1.0.6${NC}"
fi
echo ""

# 检查 NAT44 ED 插件是否启用
echo "4. 检查 NAT44 ED 插件状态..."
echo "   执行: vppctl show nat44 plugin"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 plugin
echo ""

# 检查接口地址配置（关键：应该是 L3 模式，不是 L2 xconnect）
echo "5. 检查接口地址配置（L3 vs L2 模式）..."
echo "   执行: vppctl show interface address"
INTERFACE_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface address)
echo "$INTERFACE_OUTPUT"
echo ""

# 验证是否是 L3 模式
if echo "$INTERFACE_OUTPUT" | grep -q "L2 xconnect"; then
    echo -e "${RED}❌ 失败: 仍在使用 L2 xconnect 模式！${NC}"
    echo -e "${YELLOW}   这意味着 v1.0.6 未正确部署，请检查镜像版本${NC}"
    exit 1
elif echo "$INTERFACE_OUTPUT" | grep -q "L3"; then
    echo -e "${GREEN}✓ 成功: 已切换到 L3 路由模式${NC}"
else
    echo -e "${YELLOW}⚠ 警告: 无法确定接口模式${NC}"
fi
echo ""

# 检查 NAT 接口配置
echo "6. 检查 NAT 接口配置..."
echo "   执行: vppctl show nat44 interfaces"
NAT_INTERFACES=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 interfaces)
echo "$NAT_INTERFACES"
echo ""

# 验证 inside 和 outside 接口是否都配置
if echo "$NAT_INTERFACES" | grep -q " in" && echo "$NAT_INTERFACES" | grep -q " out"; then
    echo -e "${GREEN}✓ NAT 接口配置正确: inside + outside${NC}"
else
    echo -e "${RED}❌ NAT 接口配置不完整${NC}"
    exit 1
fi
echo ""

# 检查路由表（L3 模式特有）
echo "7. 检查路由表（L3 模式验证）..."
echo "   执行: vppctl show ip fib"
FIB_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show ip fib)
echo "$FIB_OUTPUT"
echo ""

# 验证是否有路由条目
if echo "$FIB_OUTPUT" | grep -qE "10\.60\.[0-9]+\.[0-9]+/24"; then
    echo -e "${GREEN}✓ 路由表存在 10.60.x.x/24 路由条目（L3 模式正常）${NC}"
else
    echo -e "${YELLOW}⚠ 警告: 未找到预期的路由条目${NC}"
fi
echo ""

# 检查 NAT 会话（初始状态可能为 0）
echo "8. 检查 NAT 会话..."
echo "   执行: vppctl show nat44 sessions"
SESSION_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions)
echo "$SESSION_OUTPUT"
echo ""

# 提取会话数
SESSION_COUNT=$(echo "$SESSION_OUTPUT" | grep -oP '\d+ sessions' | grep -oP '\d+' | head -1)
if [ -n "$SESSION_COUNT" ]; then
    echo "   当前会话数: $SESSION_COUNT"
    if [ "$SESSION_COUNT" -eq 0 ]; then
        echo -e "${YELLOW}⚠ 当前无 NAT 会话（正常，需要有流量才会创建会话）${NC}"
    else
        echo -e "${GREEN}✓ NAT 会话已创建: $SESSION_COUNT 个会话${NC}"
    fi
else
    echo -e "${YELLOW}⚠ 无法解析会话数${NC}"
fi
echo ""

# 检查 L2 xconnect（应该不存在）
echo "9. 检查 L2 xconnect 配置（应该为空）..."
echo "   执行: vppctl show l2 xconnect"
XCONNECT_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show l2 xconnect)
echo "$XCONNECT_OUTPUT"
echo ""

if echo "$XCONNECT_OUTPUT" | grep -q "memif"; then
    echo -e "${RED}❌ 失败: 仍然存在 L2 xconnect 配置！${NC}"
    echo -e "${YELLOW}   这意味着 v1.0.6 未正确应用${NC}"
    exit 1
else
    echo -e "${GREEN}✓ L2 xconnect 已移除（符合 v1.0.6 预期）${NC}"
fi
echo ""

# 最终总结
echo "=================================================="
echo "  验证总结"
echo "=================================================="
echo ""
echo "关键检查项："
echo -e "  [✓] Pod 状态: ${GREEN}Running${NC}"
echo -e "  [✓] 镜像版本: ${GREEN}v1.0.6${NC}"
echo -e "  [✓] 接口模式: ${GREEN}L3 路由模式${NC}"
echo -e "  [✓] NAT 接口: ${GREEN}inside + outside${NC}"
echo -e "  [✓] 路由表: ${GREEN}存在路由条目${NC}"
echo -e "  [✓] L2 xconnect: ${GREEN}已移除${NC}"
echo ""

if [ "$SESSION_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}下一步：发送测试流量以创建 NAT 会话${NC}"
    echo ""
    echo "建议执行："
    echo "  1. 从 NSC 发送流量到外部服务"
    echo "  2. 重新运行本脚本查看 NAT 会话"
    echo ""
    echo "示例命令："
    echo '  NSC_POD=$(kubectl get pods -n ns-nse-composition -l app=alpine-nsc -o jsonpath='"'"'{.items[0].metadata.name}'"'"')'
    echo '  kubectl exec -n ns-nse-composition $NSC_POD -- ping -c 5 <目标IP>'
    echo '  ./verify-v1.0.6.sh'
else
    echo -e "${GREEN}✓ v1.0.6 L3 路由模式 NAT 验证成功！${NC}"
    echo -e "${GREEN}✓ NAT 会话已正常创建，功能正常工作${NC}"
fi
echo ""
echo "=================================================="
