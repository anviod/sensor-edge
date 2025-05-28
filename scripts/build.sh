#!/bin/bash
set -e
cd "$(dirname "$0")/.."

echo "[backend] building..."
cd backend || cd .
GOOS=linux GOARCH=amd64 go build -o sensor-edge-server ./cmd/main.go
cd ..

echo "[frontend] building..."
cd frontend
npm install && npm run build
cd ..

echo "[cli] building..."
cd cli
GOOS=linux GOARCH=amd64 go build -o sensor-edge ./cmd/main.go
cd ..

echo "Build finished."
