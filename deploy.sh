#!/bin/bash

# 设置变量
NAMESPACE="default"

echo "开始部署 RocketMQ 示例应用到 Kubernetes..."

# 创建命名空间（如果不存在）
echo "检查命名空间..."
kubectl get namespace ${NAMESPACE} >/dev/null 2>&1 || kubectl create namespace ${NAMESPACE}

# 部署服务端
echo "部署服务端..."
kubectl apply -f k8s/server-deployment.yaml

if [ $? -eq 0 ]; then
    echo "服务端部署成功"
else
    echo "服务端部署失败"
    exit 1
fi

# 等待服务端启动
echo "等待服务端启动..."
kubectl wait --for=condition=available --timeout=300s deployment/rocketmq-server

if [ $? -eq 0 ]; then
    echo "服务端启动成功"
else
    echo "服务端启动超时"
    exit 1
fi

# 部署客户端
echo "部署客户端..."
kubectl apply -f k8s/client-deployment.yaml

if [ $? -eq 0 ]; then
    echo "客户端部署成功"
else
    echo "客户端部署失败"
    exit 1
fi

# 等待客户端启动
echo "等待客户端启动..."
kubectl wait --for=condition=available --timeout=300s deployment/rocketmq-client

if [ $? -eq 0 ]; then
    echo "客户端启动成功"
else
    echo "客户端启动超时"
    exit 1
fi

# 显示部署状态
echo "部署完成！显示状态："
kubectl get pods -l app=rocketmq-server
kubectl get pods -l app=rocketmq-client
kubectl get services -l app=rocketmq-server
kubectl get services -l app=rocketmq-client

echo "查看日志："
echo "服务端日志："
kubectl logs -l app=rocketmq-server --tail=50
echo ""
echo "客户端日志："
kubectl logs -l app=rocketmq-client --tail=50 