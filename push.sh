#!/bin/sh

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 <version>"
	exit 1
fi

docker build -t mcp-proxy:latest . || exit 1
docker tag mcp-proxy:latest lukejoshuapark/mcp-proxy:latest || exit 1
docker tag mcp-proxy:latest lukejoshuapark/mcp-proxy:$1 || exit 1
docker push lukejoshuapark/mcp-proxy:latest || exit 1
docker push lukejoshuapark/mcp-proxy:$1 || exit 1
