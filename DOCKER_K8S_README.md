# RocketMQ Go 示例 - Docker & Kubernetes 部署指南

## 📋 概述

本项目演示了如何使用 RocketMQ 实现 Request-Response 和 Publish-Subscribe 模式，并提供了完整的 Docker 镜像构建和 Kubernetes 部署方案。

## 🏗️ 项目结构

```
godemo/
├── server/                 # 服务端代码
│   └── main.go
├── client/                 # 客户端代码
│   └── main.go
├── k8s/                   # Kubernetes 部署文件
│   ├── server-deployment.yaml
│   └── client-deployment.yaml
├── Dockerfile.server       # 服务端 Dockerfile
├── Dockerfile.client       # 客户端 Dockerfile
├── build-and-push.sh      # 构建和推送脚本
├── deploy.sh              # 部署脚本
└── go.mod                 # Go 模块文件
```

## 🐳 Docker 镜像构建

### 1. 构建和推送镜像

```bash
# 执行构建和推送脚本
./build-and-push.sh
```

这将：
- 构建服务端镜像：`harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest`
- 构建客户端镜像：`harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest`
- 推送镜像到 Harbor 仓库

### 2. 手动构建（可选）

```bash
# 构建服务端镜像
docker build -f Dockerfile.server -t harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest .

# 构建客户端镜像
docker build -f Dockerfile.client -t harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest .

# 推送镜像
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest
```

## ☸️ Kubernetes 部署

### 1. 自动部署

```bash
# 执行部署脚本
./deploy.sh
```

这将：
- 部署服务端到 Kubernetes
- 部署客户端到 Kubernetes
- 等待服务启动
- 显示部署状态和日志

### 2. 手动部署

```bash
# 部署服务端
kubectl apply -f k8s/server-deployment.yaml

# 部署客户端
kubectl apply -f k8s/client-deployment.yaml

# 查看部署状态
kubectl get pods -l app=rocketmq-server
kubectl get pods -l app=rocketmq-client

# 查看日志
kubectl logs -l app=rocketmq-server
kubectl logs -l app=rocketmq-client
```

## 🔧 配置说明

### 环境变量

- `ROCKETMQ_NAMESERVER`: RocketMQ NameServer 地址
  - 默认：`rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876`
  - 支持：Kubernetes 内部 DNS 解析

- `ROCKETMQ_GROUP`: RocketMQ 消费者组名称
  - 默认：`chat_server_group`

### Kubernetes 配置

#### 服务端部署
- **镜像**: `harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest`
- **端口**: 8080 (健康检查)
- **资源限制**: 128Mi 内存，100m CPU
- **健康检查**: `/health` 和 `/ready` 端点

#### 客户端部署
- **镜像**: `harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest`
- **资源限制**: 128Mi 内存，100m CPU
- **一次性任务**: 发送请求后退出

## 📊 监控和日志

### 查看 Pod 状态

```bash
# 查看所有相关 Pod
kubectl get pods -l app=rocketmq-server
kubectl get pods -l app=rocketmq-client

# 查看服务
kubectl get services -l app=rocketmq-server
kubectl get services -l app=rocketmq-client
```

### 查看日志

```bash
# 查看服务端日志
kubectl logs -l app=rocketmq-server --tail=100 -f

# 查看客户端日志
kubectl logs -l app=rocketmq-client --tail=100 -f
```

### 健康检查

```bash
# 检查服务端健康状态
kubectl exec -it $(kubectl get pods -l app=rocketmq-server -o jsonpath='{.items[0].metadata.name}') -- curl http://localhost:8080/health

# 检查服务端就绪状态
kubectl exec -it $(kubectl get pods -l app=rocketmq-server -o jsonpath='{.items[0].metadata.name}') -- curl http://localhost:8080/ready
```

## 🧹 清理

### 删除部署

```bash
# 删除所有相关资源
kubectl delete -f k8s/server-deployment.yaml
kubectl delete -f k8s/client-deployment.yaml

# 或者删除所有相关 Pod 和服务
kubectl delete pods,services -l app=rocketmq-server
kubectl delete pods,services -l app=rocketmq-client
```

### 删除镜像（可选）

```bash
# 从 Harbor 删除镜像（需要管理员权限）
docker rmi harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest
docker rmi harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest
```

## 🔍 故障排除

### 常见问题

1. **镜像拉取失败**
   ```bash
   # 检查 Harbor 登录状态
   docker login harbor.trevi-dev.cc
   ```

2. **RocketMQ 连接失败**
   ```bash
   # 检查 RocketMQ 服务状态
   kubectl get pods -n rocketmq-system
   
   # 检查 NameServer 服务
   kubectl get services -n rocketmq-system
   ```

3. **Pod 启动失败**
   ```bash
   # 查看 Pod 详细状态
   kubectl describe pod <pod-name>
   
   # 查看 Pod 事件
   kubectl get events --sort-by='.lastTimestamp'
   ```

4. **健康检查失败**
   ```bash
   # 检查服务端是否正常监听
   kubectl exec -it <server-pod-name> -- netstat -tlnp
   ```

### 调试命令

```bash
# 进入 Pod 调试
kubectl exec -it <pod-name> -- /bin/sh

# 查看环境变量
kubectl exec -it <pod-name> -- env

# 测试网络连接
kubectl exec -it <pod-name> -- nslookup rocketmq-nameserver.rocketmq-system.svc.cluster.local
```

## 📝 注意事项

1. **RocketMQ 依赖**: 确保 Kubernetes 集群中已部署 RocketMQ
2. **网络策略**: 确保 Pod 可以访问 RocketMQ 服务
3. **资源限制**: 根据实际需求调整 CPU 和内存限制
4. **镜像版本**: 建议使用特定版本标签而不是 `latest`
5. **日志轮转**: 生产环境建议配置日志轮转和持久化

## 🚀 扩展建议

1. **水平扩展**: 可以部署多个服务端实例
2. **配置管理**: 使用 ConfigMap 管理配置
3. **密钥管理**: 使用 Secret 管理敏感信息
4. **监控集成**: 集成 Prometheus 和 Grafana
5. **日志聚合**: 使用 ELK 或 Fluentd 进行日志聚合 