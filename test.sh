#!/usr/bin/env bash
# test.sh — 手动运行所有单元测试
#
# 用法:
#   ./test.sh           运行全部单元测试(含 race 检测)
#   ./test.sh quick     只运行本次修复相关的 race 测试
#
# 说明:
#   - 需要本机可写的 GOCACHE(沙箱环境下默认指向 /private/tmp/gocache)
#   - integration_tests 需要真实浏览器/网络,不在本脚本范围内

set -euo pipefail
cd "$(dirname "$0")"

export GOCACHE="${GOCACHE:-/private/tmp/gocache}"
mkdir -p "$GOCACHE"

echo "==> [1/4] 编译检查 (go build ./...)"
go build ./...

echo "==> [2/4] 静态检查 (go vet)"
go vet ./pkg/engine/headless/... ./pkg/utils/... ./pkg/types/...

echo "==> [3/4] 数据竞争修复的 race 测试"
echo "    - graph: 并发 AddPageState/AddEdge/GetPageState/ShortestPath"
go test -race -count=1 -v -run 'TestCrawlGraph' ./pkg/engine/headless/graph/
echo "    - crawler: 并发 uniqueActions 去重 (markActionUnique)"
go test -race -count=1 -v -run 'TestMarkActionUnique' ./pkg/engine/headless/crawler/

if [[ "${1:-}" == "quick" ]]; then
	echo "==> quick 模式, 跳过全量单元测试"
	exit 0
fi

echo "==> [4/4] 全量单元测试 (pkg/... + internal/..., 含 race)"
# 注意: pkg/engine/headless/browser 与 captcha 的部分测试需要
# 本地网络/浏览器环境, 在受限沙箱中可能失败, 与本次修复无关。
go test -race -count=1 -timeout 5m ./pkg/... ./internal/...

echo "==> 全部通过 ✅"
