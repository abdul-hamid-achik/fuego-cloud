#!/bin/bash
set -e

echo "Generating sqlc code..."
sqlc generate

echo "Done!"
