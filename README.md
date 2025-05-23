# tx
这是一个典型的构建微服务的实践任务。

## 整体任务分解：

### 环境准备： 安装或使用 Docker/Docker Compose 搭建好 Redis, PostgreSQL (带向量插件), Jaeger 服务。

项目骨架搭建： 使用 uber-go/fx 搭建 Go 项目基础结构，集成 gRPC Server。

数据库设计与初始化： 创建 user.sql 文件，定义 user 表结构。编写代码连接 PostgreSQL 并执行 migrations (如果需要) 或确保表结构存在。

### Proto 文件定义：

创建 user.proto，定义 User 消息，UserService 和其三个 RPC 方法 (Register, Login, GetUserInfo) 的请求和响应消息。

创建 system.proto，定义 SystemService 和 SendFile RPC 方法（注意流式传输）。

使用 protoc 生成 Go 代码。

### 业务逻辑实现：
#### UserService：

实现 Register (幂等、密码哈希、embedding生成)。

实现 Login (密码验证、JWT生成)。

实现 GetUserInfo (JWT验证、获取用户ID、数据库查询)。

#### SystemService：

实现 SendFile (读取文件、以流式传输)。

### 中间件集成与应用：

PostgreSQL： 实现数据库连接，用户数据的 CURD 操作。考虑如何存储和查询向量。

Redis：作为缓存加速

Jaeger： 集成 Jaeger client，通过 gRPC Interceptors 实现请求链路追踪，并在 Span Tag 中记录用户 ID。

### 安全性：
实现密码哈希存储和验证 (bcrypt 等)。

实现 JWT 的生成、签名和验证。通过 gRPC Interceptors 强制需要登录的接口进行身份验证。

### 可观测性：
集成 net/http/pprof 或类似的库，提供性能监控端点。

### 单元测试：

为 SendFile 编写单元测试，模拟 gRPC 流，验证发送内容。

代码组织与提交： 遵循 Go 项目规范，使用 .gitignore 排除敏感信息，提交到 Git/Gitee 仓库。

(选作) 部署脚本： 编写一个脚本自动化部署过程 (构建、启动等)。

### 操作细节：
#### 中间件安装 
使用 Docker Compose ，定义 postgres (查找带 pgvector 扩展的镜像)、redis 和 jaeger 服务。

pgvector:  pgvector/pgvector 。在连接数据库后执行 CREATE EXTENSION vector;。

Jaeger: 使用 jaegertracing/all-in-one 镜像。

Uber Fx: Fx 是一个依赖注入框架。你需要学习如何定义 fx.Provide 函数来提供你的依赖（如数据库连接、Redis client、Jaeger tracer、各个 Service 的实现），然后使用 fx.Invoke 来启动你的 gRPC Server。

用户分布式 ID (user_id): 题目没有指定生成方式。简单的可以使用 UUID (github.com/gofrs/uuid 或 github.com/google/uuid)。在实际系统中，可能会使用像 Snowflake 算法等分布式 ID 生成器，但在面试中说明你理解其概念，用 UUID 实现即可。

注册幂等性： 实现幂等性的常见方法有：

数据库唯一约束： 在 user_id 列上设置唯一约束。这是最简单有效的方式。在尝试插入前先查询，如果存在则返回已存在错误；或者直接尝试插入，捕获唯一约束冲突错误。

使用 Redis 锁或状态： 在注册请求处理开始时，为该 user_id 在 Redis 中设置一个锁或状态标记。处理完成后释放。这种方式更复杂，对于简单的注册流程，数据库唯一约束通常足够。考虑到是实习面试，使用数据库唯一约束可能更符合预期。

密码处理： 切记不要明文存储密码。使用 bcrypt (golang.org/x/crypto/bcrypt) 等库进行密码加盐哈希存储和验证。

JWT： 使用 github.com/golang-jwt/jwt 或类似的库。生成 JWT 时，payload 中至少要包含 user_id。JWT 签名密钥需要保存在安全的地方（例如配置文件或环境变量），不要泄露。验证 JWT 并提取 user_id 的逻辑应该放在 gRPC Server Interceptor 中。

Embedding 生成： 题目说可以调用任意平台接口。对于面试而言，你不一定需要真的去调用一个付费或需要注册的 API。你可以：
调用一个免费/简单的公共 embedding API (如果能找到且方便)。
最简单的方式： 在代码中写一个模拟函数，接收喜好字符串，返回一个硬编码或随机生成的向量。在代码或文档中说明这里原本是调用外部 API 的地方。这能展示你理解了需要调用外部服务，且避免了不必要的复杂性和成本。

PostgreSQL 向量字段： 在 user.sql 中，like_embedding 字段类型应为 vector。插入时，将 Go 中的 []float32 或 []float64 转换为数据库支持的向量格式字符串（如 [1,2,3]）或使用支持向量类型的数据库驱动/ORM。查询时，pgvector 支持各种向量相似度计算操作符 (如 <=> for cosine distance)。虽然题目只要求存储和获取，但了解这些有助于展示你对向量数据库的理解。

gRPC Interceptors： 这是实现横切关注点（如认证、链路追踪）的标准方式。你需要为 UserService 和 SystemService (或者全局) 配置 Server Interceptors。

Tracing Interceptor: 在请求开始时创建 Span，结束时结束 Span。从 context.Context 获取或设置 Trace ID。

Auth Interceptor: 在需要认证的方法前执行。从请求 metadata 中获取 JWT，验证，将用户 ID 放入 context.Context 中，供后续业务逻辑使用。

链路追踪 (user_id): 在认证拦截器中验证 JWT 成功后，获取到 user_id。将这个 user_id 作为 Tag 添加到当前请求的 Span 中。这样在 Jaeger UI 中就可以通过 user_id 搜索相关请求的链路。

性能监控 (pprof): 在你的 fx 应用中，可以添加一个独立的 goroutine 或 module 来启动一个 HTTP Server，并在特定的路径 (/debug/pprof/) 注册 net/http/pprof 提供的 handler。

SendFile 测试： 使用 bufconn 可以创建一个内存中的 gRPC 连接，模拟网络通信，这样你就可以在单元测试中启动你的 gRPC Server，通过 bufconn 连接并调用 SendFile 方法，然后读取流式响应并与原始文件内容进行比对。

配置管理： 将数据库连接信息、Redis 连接信息、JWT 密钥、文件存放路径、Jaeger endpoint 等配置项集中管理起来，可以从文件 (如 YAML, JSON) 或环境变量中读取。

错误处理： 在 Go 中，错误处理非常重要。确保你的代码有良好的错误处理机制，特别是数据库操作、文件读取、网络通信等可能失败的地方。在 gRPC 中，使用 google.golang.org/grpc/status 和 google.golang.org/grpc/codes 返回标准的 gRPC 错误码和信息。