#!/bin/bash

# 多環境部署腳本
# 支持 Kubernetes 和測試環境部署

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默認配置
ENVIRONMENT=${1:-"k8s"}
NAMESPACE=${2:-"default"}
IMAGE_TAG=${3:-"latest"}

echo -e "${GREEN}開始部署到環境: ${ENVIRONMENT}${NC}"

# 函數：構建 Docker 鏡像
build_images() {
    echo -e "${YELLOW}構建 Docker 鏡像...${NC}"
    
    # 構建服務端鏡像
    docker build -t harbor.trevi-dev.cc/cpp_run/rocketmq-server:${IMAGE_TAG} -f Dockerfile.server .
    
    # 構建客戶端鏡像
    docker build -t harbor.trevi-dev.cc/cpp_run/rocketmq-client:${IMAGE_TAG} -f Dockerfile.client .
    
    echo -e "${GREEN}鏡像構建完成${NC}"
}

# 函數：推送 Docker 鏡像
push_images() {
    echo -e "${YELLOW}推送 Docker 鏡像...${NC}"
    
    docker push harbor.trevi-dev.cc/cpp_run/rocketmq-server:${IMAGE_TAG}
    docker push harbor.trevi-dev.cc/cpp_run/rocketmq-client:${IMAGE_TAG}
    
    echo -e "${GREEN}鏡像推送完成${NC}"
}

# 函數：部署到 Kubernetes
deploy_k8s() {
    echo -e "${YELLOW}部署到 Kubernetes...${NC}"
    
    # 應用 ConfigMap
    kubectl apply -f k8s/rocketmq-config.yaml
    
    # 更新部署配置中的鏡像標籤
    sed -i.bak "s|image: harbor.trevi-dev.cc/cpp_run/rocketmq-server:.*|image: harbor.trevi-dev.cc/cpp_run/rocketmq-server:${IMAGE_TAG}|g" k8s/server-deployment.yaml
    sed -i.bak "s|image: harbor.trevi-dev.cc/cpp_run/rocketmq-client:.*|image: harbor.trevi-dev.cc/cpp_run/rocketmq-client:${IMAGE_TAG}|g" k8s/client-deployment.yaml
    
    # 應用部署
    kubectl apply -f k8s/server-deployment.yaml
    kubectl apply -f k8s/client-deployment.yaml
    
    # 清理備份文件
    rm -f k8s/*.bak
    
    echo -e "${GREEN}Kubernetes 部署完成${NC}"
}

# 函數：部署到測試環境
deploy_test() {
    echo -e "${YELLOW}部署到測試環境...${NC}"
    
    # 應用測試環境 ConfigMap
    kubectl apply -f k8s/rocketmq-config-test.yaml
    
    # 更新部署配置使用測試環境配置
    sed -i.bak "s|name: rocketmq-config|name: rocketmq-config-test|g" k8s/server-deployment.yaml
    sed -i.bak "s|name: rocketmq-config|name: rocketmq-config-test|g" k8s/client-deployment.yaml
    
    # 應用部署
    kubectl apply -f k8s/server-deployment.yaml
    kubectl apply -f k8s/client-deployment.yaml
    
    # 恢復原始配置
    sed -i.bak "s|name: rocketmq-config-test|name: rocketmq-config|g" k8s/server-deployment.yaml
    sed -i.bak "s|name: rocketmq-config-test|name: rocketmq-config|g" k8s/client-deployment.yaml
    
    # 清理備份文件
    rm -f k8s/*.bak
    
    echo -e "${GREEN}測試環境部署完成${NC}"
}

# 函數：檢查部署狀態
check_status() {
    echo -e "${YELLOW}檢查部署狀態...${NC}"
    
    # 等待 Pod 啟動
    sleep 10
    
    # 檢查 Pod 狀態
    echo "服務端 Pod 狀態:"
    kubectl get pods -l app=rocketmq-server
    
    echo "客戶端 Pod 狀態:"
    kubectl get pods -l app=rocketmq-client
    
    # 檢查服務狀態
    echo "服務狀態:"
    kubectl get services -l app=rocketmq-server
    kubectl get services -l app=rocketmq-client
}

# 函數：顯示日誌
show_logs() {
    echo -e "${YELLOW}顯示服務日誌...${NC}"
    
    echo "服務端日誌:"
    kubectl logs -l app=rocketmq-server --tail=10
    
    echo "客戶端日誌:"
    kubectl logs -l app=rocketmq-client --tail=10
}

# 主函數
main() {
    case $ENVIRONMENT in
        "k8s")
            build_images
            push_images
            deploy_k8s
            check_status
            show_logs
            ;;
        "test")
            build_images
            push_images
            deploy_test
            check_status
            show_logs
            ;;
        *)
            echo -e "${RED}錯誤: 不支持的環境 '${ENVIRONMENT}'${NC}"
            echo "用法: $0 [k8s|test] [namespace] [image_tag]"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}部署完成！${NC}"
}

# 顯示幫助信息
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo "用法: $0 [環境] [命名空間] [鏡像標籤]"
    echo ""
    echo "環境選項:"
    echo "  k8s   - 部署到 Kubernetes 環境"
    echo "  test  - 部署到測試環境"
    echo ""
    echo "示例:"
    echo "  $0 k8s default v1.0.0"
    echo "  $0 test default latest"
    exit 0
fi

# 執行主函數
main 