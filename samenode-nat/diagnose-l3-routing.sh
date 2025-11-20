#!/bin/bash
# L3 路由模式诊断脚本
# 用于诊断 v1.0.6 L3 路由配置和 ping 不通问题

set -e

echo "========================================"
echo "  L3 路由模式诊断（v1.0.6）"
echo "========================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取 Pod 名称
echo "查找相关 Pod..."
NAT_POD=$(kubectl get pods -n ns-nse-composition -l app=nse-nat-vpp -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
NSC_POD=$(kubectl get pods -n ns-nse-composition -l app=alpine-nsc -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
NSE_POD=$(kubectl get pods -n ns-nse-composition -l app=nse-kernel -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$NAT_POD" ]; then
    echo -e "${RED}❌ 错误: 找不到 NAT NSE Pod${NC}"
    exit 1
fi

echo -e "NAT POD: ${GREEN}$NAT_POD${NC}"
echo -e "NSC POD: ${GREEN}${NSC_POD:-未找到}${NC}"
echo -e "NSE POD: ${GREEN}${NSE_POD:-未找到}${NC}"
echo ""

# ====================================================================================
# 第1部分：VPP 接口配置检查
# ====================================================================================
echo "========================================"
echo "1. VPP 接口状态"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface
echo ""

# ====================================================================================
# 第2部分：接口地址配置检查（关键：L3 vs L2）
# ====================================================================================
echo "========================================"
echo "2. 接口地址配置（L3 vs L2 模式检查）"
echo "========================================"
INTERFACE_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface address)
echo "$INTERFACE_OUTPUT"
echo ""

if echo "$INTERFACE_OUTPUT" | grep -q "L2 xconnect"; then
    echo -e "${RED}❌ 检测到 L2 xconnect 模式！${NC}"
    echo -e "${YELLOW}   v1.0.6 应该是 L3 路由模式，请检查镜像版本${NC}"
elif echo "$INTERFACE_OUTPUT" | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/[0-9]+"; then
    echo -e "${GREEN}✓ 检测到 L3 IP 地址配置${NC}"
else
    echo -e "${YELLOW}⚠ 警告: 接口可能没有配置 IP 地址${NC}"
fi
echo ""

# ====================================================================================
# 第3部分：路由表检查（L3 模式关键）
# ====================================================================================
echo "========================================"
echo "3. IP 路由表（FIB）"
echo "========================================"
FIB_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show ip fib)
echo "$FIB_OUTPUT"
echo ""

# 检查是否有路由条目
if echo "$FIB_OUTPUT" | grep -qE "172\.16\.1\.(100|101)"; then
    echo -e "${GREEN}✓ 检测到 172.16.1.x 路由条目${NC}"
else
    echo -e "${RED}❌ 缺少 172.16.1.x 路由条目！${NC}"
    echo -e "${YELLOW}   这可能是 ping 不通的原因${NC}"
fi
echo ""

# ====================================================================================
# 第4部分：ARP 表检查
# ====================================================================================
echo "========================================"
echo "4. ARP 表"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show ip arp
echo ""

# ====================================================================================
# 第5部分：NAT 配置检查
# ====================================================================================
echo "========================================"
echo "5. NAT44 接口配置"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 interfaces
echo ""

echo "========================================"
echo "6. NAT44 地址池"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 addresses
echo ""

echo "========================================"
echo "7. NAT44 会话"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show nat44 sessions
echo ""

# ====================================================================================
# 第6部分：L2 xconnect 检查（应该为空）
# ====================================================================================
echo "========================================"
echo "8. L2 xconnect 配置（应该为空）"
echo "========================================"
XCONNECT_OUTPUT=$(kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show l2 xconnect)
echo "$XCONNECT_OUTPUT"

if echo "$XCONNECT_OUTPUT" | grep -q "memif"; then
    echo -e "${RED}❌ 检测到 L2 xconnect！v1.0.6 不应该有这个${NC}"
else
    echo -e "${GREEN}✓ 无 L2 xconnect（符合 v1.0.6 预期）${NC}"
fi
echo ""

# ====================================================================================
# 第7部分：数据包计数器检查
# ====================================================================================
echo "========================================"
echo "9. 接口数据包统计"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show interface | grep -E "(memif|rx packets|tx packets|drops)"
echo ""

# ====================================================================================
# 第8部分：VPP 图节点统计
# ====================================================================================
echo "========================================"
echo "10. VPP 数据包处理路径（graph nodes）"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show node counters
echo ""

# ====================================================================================
# 第9部分：错误统计
# ====================================================================================
echo "========================================"
echo "11. VPP 错误统计"
echo "========================================"
kubectl exec -n ns-nse-composition $NAT_POD -- vppctl show errors
echo ""

# ====================================================================================
# 第10部分：连通性测试
# ====================================================================================
if [ -n "$NSC_POD" ] && [ -n "$NSE_POD" ]; then
    echo "========================================"
    echo "12. 连通性测试"
    echo "========================================"

    echo "NSC (172.16.1.101) -> NSE (172.16.1.100)"
    kubectl exec -n ns-nse-composition $NSC_POD -- ping -c 3 -W 2 172.16.1.100 2>&1 || echo -e "${RED}✗ Ping 失败${NC}"
    echo ""

    echo "NSE (172.16.1.100) -> NSC (172.16.1.101)"
    kubectl exec -n ns-nse-composition $NSE_POD -c nse -- ping -c 3 -W 2 172.16.1.101 2>&1 || echo -e "${RED}✗ Ping 失败${NC}"
    echo ""
fi

# ====================================================================================
# 总结和建议
# ====================================================================================
echo "========================================"
echo "诊断总结"
echo "========================================"
echo ""

# 检查关键项
ISSUES=0

# 检查1: L3 vs L2 模式
if echo "$INTERFACE_OUTPUT" | grep -q "L2 xconnect"; then
    echo -e "${RED}[问题1] 仍在使用 L2 xconnect 模式${NC}"
    echo "  解决: 确认镜像版本是 v1.0.6"
    ISSUES=$((ISSUES+1))
elif ! echo "$INTERFACE_OUTPUT" | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/[0-9]+"; then
    echo -e "${YELLOW}[警告1] 接口未配置 IP 地址${NC}"
    echo "  可能原因: ipaddress 组件未正常工作"
    ISSUES=$((ISSUES+1))
else
    echo -e "${GREEN}[正常1] L3 IP 地址已配置${NC}"
fi

# 检查2: 路由表
if ! echo "$FIB_OUTPUT" | grep -qE "172\.16\.1\.(100|101)"; then
    echo -e "${RED}[问题2] 缺少必要的路由条目${NC}"
    echo "  解决: 检查 routes 组件是否正常工作"
    echo "  预期: 应该看到 172.16.1.100/32 和 172.16.1.101/32 的路由"
    ISSUES=$((ISSUES+1))
else
    echo -e "${GREEN}[正常2] 路由表已配置${NC}"
fi

# 检查3: xconnect 状态
if echo "$XCONNECT_OUTPUT" | grep -q "memif"; then
    echo -e "${RED}[问题3] L2 xconnect 仍然存在${NC}"
    echo "  解决: 代码未正确移除 xconnect"
    ISSUES=$((ISSUES+1))
else
    echo -e "${GREEN}[正常3] L2 xconnect 已移除${NC}"
fi

echo ""
if [ $ISSUES -eq 0 ]; then
    echo -e "${GREEN}✓ 所有检查通过，但 ping 仍不通${NC}"
    echo ""
    echo "可能的其他原因："
    echo "  1. 接口顺序问题（nat → ipaddress → routes vs ipaddress → routes → nat）"
    echo "  2. memif 接口未正确配置"
    echo "  3. 需要手动添加 IP neighbor（ARP）"
    echo "  4. VPP 数据包处理路径问题"
    echo ""
    echo "建议："
    echo "  1. 调整客户端链顺序：up → memif → ipaddress → routes → nat"
    echo "  2. 手动配置 ARP：vppctl set ip arp <interface> <ip> <mac>"
    echo "  3. 检查 VPP graph nodes 中的丢包原因"
else
    echo -e "${RED}✗ 发现 $ISSUES 个问题，请先解决这些问题${NC}"
fi

echo ""
echo "========================================"
echo "诊断完成"
echo "========================================"
