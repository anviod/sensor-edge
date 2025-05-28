#!/bin/bash
set -e
cd "$(dirname "$0")/.."

echo "[backend] 启动..."
./backend/sensor-edge-server &
BACKEND_PID=$!

sleep 2
echo "[frontend] 启动..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

sleep 2
echo "[cli] 可用，示例：./cli/sensor-edge device list"

echo "所有服务已启动。按 Ctrl+C 退出。"
wait $BACKEND_PID $FRONTEND_PID
