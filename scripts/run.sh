#!/bin/bash
set -e

echo "start build"
go build .
echo "finished successfully build"

./messenger-backend.exe