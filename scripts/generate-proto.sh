# Generate Go code from protobuf.
# Run this from root of the project.

#!/bin/bash

set -e # Exit on error.

# Output coloring.
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No color.

echo -e "${BLUE}Generating protobuf code...${NC}"

# Create output dir.
mkdir -p pkg/api/task/v1

MODULE_NAME=$(grep "^module" go.mod|awk '{print $2}')

protoc \
  --go_out=. \
  --go_opt=paths=import \
  --go_opt=module=${MODULE_NAME} \
  --go-grpc_out=. \
  --go-grpc_opt=paths=import \
  --go-grpc_opt=module=${MODULE_NAME} \
  --proto_path=. \
  proto/task/v1/task.proto

echo -e "${GREEN}✓ Generated Go code in pkg/api/task/v1/${NC}"
echo -e "${BLUE}Files created:${NC}"
ls -lh pkg/api/task/v1/



