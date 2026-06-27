#!/bin/bash

# 集成测试脚本 - ACGWarehouse Backend API

set -o pipefail

PORT=7070
BASE_URL="http://localhost:${PORT}/api/v1"
PROJECT_DIR="/opt/acgwarehouse"
BUGS_FILE="${PROJECT_DIR}/.trellis/tasks/06-27-integration-verification/bugs.md"
BACKEND_PID=""

TEST_USER="testuser_$(date +%s)"
TEST_PASS="TestPass123"
ADMIN_USER="testadmin"
ADMIN_PASS="AdminPass123"

USER_TOKEN=""
ADMIN_TOKEN=""

TOTAL=0
PASS=0
FAIL=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASS++)); }
log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAIL++))
    echo "- $1" >> "$BUGS_FILE"
    echo "  原因: $2" >> "$BUGS_FILE"
    echo "" >> "$BUGS_FILE"
}

start_server() {
    log_info "启动后端服务 (端口: $PORT)..."
    cd "$PROJECT_DIR"
    export PORT=$PORT
    export ADMIN_USERNAME=$ADMIN_USER
    export ADMIN_PASSWORD=$ADMIN_PASS
    export JWT_SECRET="test-integration-secret"
    export LOG_LEVEL="warn"
    go run ./cmd/web &
    BACKEND_PID=$!
    log_info "等待服务就绪..."
    for i in {1..30}; do
        if curl -s "http://localhost:${PORT}/api/v1/ping" > /dev/null 2>&1; then
            log_info "服务已就绪 (PID: $BACKEND_PID)"
            return 0
        fi
        sleep 1
    done
    log_fail "服务启动失败" "服务在 30 秒内未就绪"
    return 1
}

stop_server() {
    if [[ -n "$BACKEND_PID" ]]; then
        log_info "停止后端服务 (PID: $BACKEND_PID)..."
        kill "$BACKEND_PID" 2>/dev/null || true
        wait "$BACKEND_PID" 2>/dev/null || true
        log_info "服务已停止"
    fi
}

run_test() {
    local name="$1"
    local method="$2"
    local path="$3"
    local expected_code="$4"
    local data="$5"
    local token="$6"

    ((TOTAL++))

    local url="${BASE_URL}${path}"
    local args=(-s -X "$method" -w "\n%{http_code}")

    [[ -n "$token" ]] && args+=(-H "Authorization: Bearer $token")
    [[ -n "$data" ]] && args+=(-H "Content-Type: application/json" -d "$data")

    local resp=$(curl "${args[@]}" "$url" 2>/dev/null)
    local code=$(echo "$resp" | tail -1)
    local body=$(echo "$resp" | sed '$d')

    if [[ "$code" == "$expected_code" ]]; then
        log_pass "$name"
        echo "$body"
        return 0
    else
        log_fail "$name" "HTTP $code (期望 $expected_code), 响应: $body"
        return 1
    fi
}

echo "# Bug 记录" > "$BUGS_FILE"
echo "" >> "$BUGS_FILE"

trap stop_server EXIT

start_server || exit 1

log_info "========== 用户认证测试 =========="

# T01: 健康检查
run_test "T01: 健康检查" GET "/ping" "200"

# T02: 用户注册
run_test "T02: 用户注册 - 正常" POST "/users/register" "200" "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASS\"}"

# T03: 重复注册
run_test "T03: 用户注册 - 重复" POST "/users/register" "400" "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASS\"}"

# T04: 非法参数
run_test "T04: 用户注册 - 非法参数" POST "/users/register" "400" '{"username":"ab","password":"123456"}'

# T05: 登录
body=$(run_test "T05: 用户登录 - 正常" POST "/users/login" "200" "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASS\"}")
USER_TOKEN=$(echo "$body" | grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*:.*"\([^"]*\)"/\1/')

# T06: 错误密码
run_test "T06: 用户登录 - 错误密码" POST "/users/login" "401" "{\"username\":\"$TEST_USER\",\"password\":\"WrongPass123\"}"

# T07: 不存在用户
run_test "T07: 用户登录 - 不存在用户" POST "/users/login" "401" '{"username":"nonexistent_xyz","password":"SomePass123"}'

# T08: 管理员登录
body=$(run_test "T08: 管理员登录" POST "/users/login" "200" "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")
ADMIN_TOKEN=$(echo "$body" | grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*:.*"\([^"]*\)"/\1/')

log_info "========== 权限控制测试 =========="

# T09: 获取当前用户 - 已认证
run_test "T09: 获取当前用户 - 已认证" GET "/users/me" "200" "" "$USER_TOKEN"

# T10: 获取当前用户 - 未认证
run_test "T10: 获取当前用户 - 未认证" GET "/users/me" "401"

log_info "========== 图片管理测试 =========="

# T11: 图片列表
run_test "T11: 图片列表" GET "/images?page=1&size=10" "200"

# T12: 图片详情 - 不存在
run_test "T12: 图片详情 - 不存在" GET "/images/999999999" "404"

# T13: 图片搜索
run_test "T13: 图片搜索" GET "/search?q=test&page=1&size=10" "200"

log_info "========== 标签管理测试 =========="

# T14: 标签列表
run_test "T14: 标签列表" GET "/tags" "200"

# T15: 标签建议
run_test "T15: 标签建议" GET "/tags/suggest?q=test&limit=10" "200"

# T16: 创建标签
run_test "T16: 创建标签" POST "/tags" "200" '{"name":"test_tag_integration"}' "$USER_TOKEN"

log_info "========== 评分测试 =========="

# T17: 评分 - 正常
run_test "T17: 图片评分 - 正常 (0)" PUT "/images/1/rating" "200" '{"score":0}' "$USER_TOKEN"

# T18: 评分 - 边界
run_test "T18: 图片评分 - 边界 (100)" PUT "/images/1/rating" "200" '{"score":100}' "$USER_TOKEN"

# T19: 评分 - 超范围
run_test "T19: 图片评分 - 超范围 (101)" PUT "/images/1/rating" "400" '{"score":101}' "$USER_TOKEN"

# T20: 评分 - 未认证
run_test "T20: 图片评分 - 未认证" PUT "/images/1/rating" "401" '{"score":50}'

log_info "========== 收藏夹测试 =========="

# T21: 创建收藏夹
run_test "T21: 创建收藏夹" POST "/collections" "200" '{"name":"测试收藏夹","visibility":"public"}' "$USER_TOKEN"

# T22: 收藏夹列表
run_test "T22: 收藏夹列表" GET "/collections" "200" "" "$USER_TOKEN"

# T23: 未认证访问收藏夹
run_test "T23: 收藏夹 - 未认证" GET "/collections" "401"

log_info "========== 热榜测试 =========="

# T24: 日榜
run_test "T24: 热榜 - 日榜" GET "/rankings?period=day&page=1&size=10" "200"

# T25: 周榜
run_test "T25: 热榜 - 周榜" GET "/rankings?period=week&page=1&size=10" "200"

# T26: 月榜
run_test "T26: 热榜 - 月榜" GET "/rankings?period=month&page=1&size=10" "200"

# T27: 非法周期
run_test "T27: 热榜 - 非法周期" GET "/rankings?period=invalid&page=1&size=10" "400"

log_info "========== 管理员操作测试 =========="

# T28: 管理员更新标签
run_test "T28: 管理员更新标签" PUT "/tags/1" "200" '{"name":"updated_tag"}' "$ADMIN_TOKEN"

# T29: 普通用户访问管理接口
run_test "T29: 普通用户访问管理接口" PUT "/tags/1" "403" '{"name":"hack"}' "$USER_TOKEN"

echo ""
echo "==================== 测试总结 ===================="
echo "总数: $TOTAL"
echo "通过: $PASS"
echo "失败: $FAIL"
echo "=================================================="

if [[ $FAIL -eq 0 ]]; then
    log_info "所有测试通过!"
    exit 0
else
    log_info "部分测试失败，详见 $BUGS_FILE"
    exit 1
fi
