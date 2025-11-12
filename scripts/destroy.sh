#!/bin/bash
set -e

echo "🗑️  Destroying Taskboard deployment..."

helmfile destroy

echo "✅ Destroyed!"
