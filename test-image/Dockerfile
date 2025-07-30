FROM --platform=linux/amd64 alpine:latest

# 创建一个简单的测试程序
RUN echo '#!/bin/sh' > /test.sh && \
    echo 'echo "Hello from amd64 container"' >> /test.sh && \
    chmod +x /test.sh

CMD ["/test.sh"] 