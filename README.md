# tx
这是一个典型的构建微服务的实践任务。

## 整体任务分解：

### 环境准备： 

安装或使用 Docker/Docker Compose 搭建好 Redis, PostgreSQL (带向量插件), Jaeger 服务。

一键配置环境 docker-compose up -d

### 单元测试：

为 SendFile 编写单元测试，模拟 gRPC 流，验证发送内容。

代码组织与提交： 遵循 Go 项目规范，使用 .gitignore 排除敏感信息，提交到 Git/Gitee 仓库。

### 部署脚本：

需要安装 grpcurl 测试环境，docker-compose部署好后启动

./test_server.sh