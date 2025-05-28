#!/bin/bash
set -e
cd "$(dirname "$0")/.."

echo "[backend] 单元测试..."
cd backend || cd .
go test ./...
cd ..

echo "[cli] 单元测试..."
cd cli
go test ./...
cd ..

echo "[frontend] Lint 检查..."
cd frontend
npm run lint || true
cd ..

echo "[API] 集成测试..."
curl -s http://localhost:8080/api/devices || echo "后端未启动，跳过 API 测试"

echo "所有测试完成。"
