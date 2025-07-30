#!/bin/bash

# 设置变量
HARBOR_REGISTRY="harbor.trevi-dev.cc"
PROJECT="cpp_run"
VERSION="latest"

# 设置镜像名称
SERVER_IMAGE="${HARBOR_REGISTRY}/${PROJECT}/rocketmq-server:${VERSION}"
CLIENT_IMAGE="${HARBOR_REGISTRY}/${PROJECT}/rocketmq-client:${VERSION}"

echo "开始构建和推送 Docker 镜像..."

# 构建服务端镜像
echo "构建服务端镜像: ${SERVER_IMAGE}"
docker buildx build --platform linux/amd64 --load -f Dockerfile.server -t ${SERVER_IMAGE} .

if [ $? -eq 0 ]; then
    echo "服务端镜像构建成功"
else
    echo "服务端镜像构建失败"
    exit 1
fi

# 构建客户端镜像
echo "构建客户端镜像: ${CLIENT_IMAGE}"
docker buildx build --platform linux/amd64 --load -f Dockerfile.client -t ${CLIENT_IMAGE} .

if [ $? -eq 0 ]; then
    echo "客户端镜像构建成功"
else
    echo "客户端镜像构建失败"
    exit 1
fi

# 推送服务端镜像
echo "推送服务端镜像到 Harbor..."
docker push ${SERVER_IMAGE}

if [ $? -eq 0 ]; then
    echo "服务端镜像推送成功"
else
    echo "服务端镜像推送失败"
    exit 1
fi

# 推送客户端镜像
echo "推送客户端镜像到 Harbor..."
docker push ${CLIENT_IMAGE}

if [ $? -eq 0 ]; then
    echo "客户端镜像推送成功"
else
    echo "客户端镜像推送失败"
    exit 1
fi

echo "所有镜像构建和推送完成！"
echo "服务端镜像: ${SERVER_IMAGE}"
echo "客户端镜像: ${CLIENT_IMAGE}" 