#!/bin/bash
set -e

echo "start build app"
go build ./cmd/app/
echo "finished successfully build app"

./app.exe