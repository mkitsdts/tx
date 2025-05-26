#!/bin/bash

# --- 配置 ---
GRPC_SERVER_ADDR="localhost:50051" # 你的gRPC服务器地址和端口
USER_PROTO_PATH="proto/user.proto"
SYSTEM_PROTO_PATH="proto/system.proto"

LOGIN_USERNAME="test"
LOGIN_PASSWORD="test"

# 这是客户端请求服务器发送的文件路径，服务器需要能访问此路径下的文件
FILE_PATH_ON_SERVER="test.txt" # 请修改为服务器上实际存在的文件路径

# ghz 参数
CONCURRENT_REQUESTS=10 # ghz 并发请求数
TOTAL_REQUESTS=100     # ghz 总请求数

# --- 脚本开始 ---

# 为了测试，我们可以在服务器端创建一个示例文件 (如果 FILE_PATH_ON_SERVER 指向的不是已有文件)
# 注意：在生产测试中，你应该确保 FILE_PATH_ON_SERVER 指向一个有意义的、已存在的文件。
# 这里仅为演示，如果你的服务器需要一个真实文件，请确保它存在。
# echo "This is a test file for streaming." > "$FILE_PATH_ON_SERVER" # 取消注释并在服务器端有权限的地方创建

# 用户登录并获取Token (使用 grpcurl)
echo "正在登录用户 '$LOGIN_USERNAME'..."
LOGIN_RESPONSE=$(grpcurl -plaintext \
    -proto "${USER_PROTO_PATH}" \
    -d "{\"username\": \"${LOGIN_USERNAME}\", \"password\": \"${LOGIN_PASSWORD}\"}" \
    "${GRPC_SERVER_ADDR}" \
    user.UserService.Login 2>/dev/null) # 2>/dev/null 避免grpcurl自身的错误信息干扰jq

if [ -z "$LOGIN_RESPONSE" ]; then
    echo "登录失败: 未收到服务器响应或grpcurl执行错误。"
    exit 1
fi

TOKEN=$(echo "${LOGIN_RESPONSE}" | jq -r '.token // empty')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo "登录失败: 未能从响应中提取Token。"
    echo "服务器响应: $LOGIN_RESPONSE"
    exit 1
fi
echo "登录成功! Token: $TOKEN"

# 使用 grpcurl 发送文件请求并保存响应到临时目录
grpcurl -plaintext \
  -H "Authorization: $TOKEN" \
  -proto "${SYSTEM_PROTO_PATH}" \
  -d "{\"file_path\": \"${FILE_PATH_ON_SERVER}\"}" \
  "${GRPC_SERVER_ADDR}" \
  system.SystemService.SendFile > "temp/test.txt"
echo "使用grpcurl接收的文件保存在: temp/test.txt"

# 使用获取到的Token调用 SendFile (使用 ghz)
echo "开始使用ghz对 SendFile 进行压力测试..."
echo "请求服务器发送文件: $FILE_PATH_ON_SERVER"

# 使用 ghz 进行压力测试
ghz --insecure \
    --proto "${SYSTEM_PROTO_PATH}" \
    --call system.SystemService.SendFile \
    -m "{\"authorization\": \"${TOKEN}\"}" \
    -d "{\"file_path\": \"${FILE_PATH_ON_SERVER}\"}" \
    -c "${CONCURRENT_REQUESTS}" \
    -n "${TOTAL_REQUESTS}" \
    "${GRPC_SERVER_ADDR}"

echo "压力测试脚本执行完毕。"
# 临时目录会在脚本退出时通过 trap 命令自动清理