#!/usr/bin/env bash
# test.sh — 手动运行本次数据竞争修复相关的所有单元测试
#
# 覆盖:
#   1. pkg/engine/headless/graph    — CrawlGraph 读写锁 (含并发 race 测试)
#   2. pkg/engine/headless/crawler  — uniqueActions 互斥锁 / 导航策略串行化
#
# 用法:
#   bash test.sh           # 运行全部测试 (含 -race)
#   bash test.sh graph     # 只跑 graph 包
#   bash test.sh crawler   # 只跑 crawler 包
set -euo pipefail
cd "$(dirname "$0")"

# 沙箱环境下默认 go-build 缓存可能不可写, 需要时取消下一行注释
# export GOCACHE="${GOCACHE:-/tmp/gocache}"

run_graph() {
    echo "=== [1/4] go vet: graph ==="
    go vet ./pkg/engine/headless/graph/

    echo "=== [2/4] go test -race: graph (CrawlGraph 并发) ==="
    go test -race -count=1 -v \
        -run 'TestAddPageState|TestCrawlGraph_ConcurrentAccess' \
        ./pkg/engine/headless/graph/
}

run_crawler() {
    echo "=== [3/4] go vet: crawler ==="
    go vet ./pkg/engine/headless/crawler/

    echo "=== [4/4] go test -race: crawler (uniqueActions 并发 + 既有单测) ==="
    go test -race -count=1 -v ./pkg/engine/headless/crawler/
}

case "${1:-all}" in
    graph)   run_graph ;;
    crawler) run_crawler ;;
    all)     run_graph; run_crawler ;;
    *) echo "unknown target: $1 (expected: graph|crawler|all)" >&2; exit 1 ;;
esac

echo "ALL TESTS PASSED"
