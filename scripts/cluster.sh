#!/usr/bin/env bash
# Starts a local 3-node toyblockchain cluster on ports 8001-8003.
# Usage:
#   ./scripts/cluster.sh start   # build + launch 3 nodes in the background
#   ./scripts/cluster.sh stop    # stop all nodes started by this script
#   ./scripts/cluster.sh status  # show whether each node is responding

set -euo pipefail

cd "$(dirname "$0")/.."

PORTS=(8001 8002 8003)
PIDFILE=".cluster_pids"
LOGDIR="logs"

peers_for() {
    local self=$1
    local list=()
    for p in "${PORTS[@]}"; do
        if [[ "$p" != "$self" ]]; then
            list+=("localhost:$p")
        fi
    done
    local IFS=,
    echo "${list[*]}"
}

start() {
    echo "Building..."
    go build -o bin/node ./cmd/node

    mkdir -p "$LOGDIR"
    : > "$PIDFILE"

    for port in "${PORTS[@]}"; do
        peers=$(peers_for "$port")
        echo "Starting node on :$port (peers: $peers)"

        ./bin/node \
            -addr "localhost:$port" \
            -peers "$peers" \
            > "$LOGDIR/node-$port.log" 2>&1 &

        echo $! >> "$PIDFILE"
    done

    echo "Cluster started. Logs in ./$LOGDIR/, PIDs in $PIDFILE."
    echo "Try: curl localhost:8001/status"
}

stop() {
    if [[ ! -f "$PIDFILE" ]]; then
        echo "No PID file found — nothing to stop."
        exit 0
    fi

    while read -r pid; do
        if kill "$pid" 2>/dev/null; then
            echo "Stopped PID $pid"
        fi
    done < "$PIDFILE"

    rm -f "$PIDFILE"
}

status() {
    for port in "${PORTS[@]}"; do
        if curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port/health" | grep -q 200; then
            echo "node :$port -> healthy"
        else
            echo "node :$port -> unreachable"
        fi
    done
}

case "${1:-}" in
    start)  start ;;
    stop)   stop ;;
    status) status ;;
    *)
        echo "Usage: $0 {start|stop|status}"
        exit 1
        ;;
esac
