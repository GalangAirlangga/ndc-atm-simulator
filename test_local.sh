#!/usr/bin/env bash
set -e

HOST="${1:-127.0.0.1}"
PORT="${2:-12355}"

echo "[test] build simulator"
go build -o ndcatm .

echo "[test] start mock server on ${HOST}:${PORT}"
python3 mock_server.py "${HOST}" "${PORT}" &
SERVER_PID=$!
sleep 0.5

cleanup() {
    echo "[test] stop mock server"
    kill "${SERVER_PID}" 2>/dev/null || true
}
trap cleanup EXIT

echo "[test] send NDC ASCII message"
./ndcatm -host "${HOST}:${PORT}" -class 1 -type 11 -subtype 1 -luno 001 -ascii 'Hello' -timeout 3s

echo "[test] done"
