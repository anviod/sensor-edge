#!/bin/bash
set -e
cd "$(dirname "$0")/.."
cd frontend
npm install && npm run build
cp -r dist ../backend/public
cd ..
echo "[frontend] 部署产物已复制到 backend/public 目录。"
