#!/bin/bash

# Test script for the tx gRPC server with Jaeger integration

# --- Configuration ---
SERVER_BINARY="./tx"       # 指向编译后的服务器可执行文件
GRPC_ADDRESS="localhost:50051"   # 替换为您的 gRPC 服务器地址 (来自 cfg.GRPC.Address)

# System Service configuration
SYSTEM_PROTO_FILE_PATH="./proto/system.proto" # 指向您的 system.proto 定义文件
SYSTEM_SERVICE_NAME="system.SystemService"   # package_name.ServiceName for System service
SYSTEM_METHOD_NAME="SendFile"

# User Service configuration
USER_PROTO_FILE_PATH="./proto/user.proto"     # 指向您的 user.proto 定义文件
USER_SERVICE_NAME="user.UserService"       # package_name.ServiceName for User service
REGISTER_METHOD_NAME="Register"
LOGIN_METHOD_NAME="Login"
TEST_USERNAME="test"
TEST_PASSWORD="test"
TEST_LIKES="coding, grpc, testing"

PROTO_IMPORT_PATH="./proto"       # grpcurl 的 -import-path，通常是包含 .proto 文件的目录

TEMP_FILE_DIR=$(mktemp -d) # 创建一个临时目录
TEST_FILE_PATH="$TEMP_FILE_DIR/test_send_file_from_script.txt"
TEST_FILE_CONTENT="Hello from gRPC client! This is a test file for SendFile."
AUTH_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdCIsImV4cCI6MTc0ODEzMzExOCwibmJmIjoxNzQ4MDQ2NzE4LCJpYXQiOjE3NDgwNDY3MTh9.EEpHGXKEYy9XMFNu1Tct1AtidFDg8Kyl8HZI-SoFzG0" # 将在登录成功后填充
REGISTRATION_SUCCESSFUL=true # 标记注册是否成功

# --- Functions ---
cleanup() {
  echo "INFO: Cleaning up..."
  if [[ -n "$SERVER_PID" ]] && ps -p "$SERVER_PID" > /dev/null; then
    echo "INFO: Stopping server (PID $SERVER_PID)..."
    kill "$SERVER_PID"
    wait "$SERVER_PID" 2>/dev/null
    echo "INFO: Server stopped."
  fi
  if [ -d "$TEMP_FILE_DIR" ]; then
    echo "INFO: Removing temporary directory $TEMP_FILE_DIR"
    rm -rf "$TEMP_FILE_DIR"
  fi
}

# Trap EXIT signal to ensure cleanup runs
trap cleanup EXIT

# 4. Server binary
if [ ! -f "$SERVER_BINARY" ]; then
    echo "INFO: Server binary '$SERVER_BINARY' not found. Attempting to build from main.go..."
    if go build -o "$SERVER_BINARY" main.go; then
        echo "INFO: Server built successfully: $SERVER_BINARY"
    else
        echo "ERROR: Failed to build server from main.go. Please build it manually."
        exit 1
    fi
else
    echo "      OK: Server binary found: $SERVER_BINARY"
fi

# 5. Proto files
if [ ! -f "$SYSTEM_PROTO_FILE_PATH" ]; then
    echo "ERROR: System Proto file '$SYSTEM_PROTO_FILE_PATH' not found."
    exit 1
else
    echo "      OK: System Proto file found: $SYSTEM_PROTO_FILE_PATH"
fi
if [ ! -f "$USER_PROTO_FILE_PATH" ]; then
    echo "ERROR: User Proto file '$USER_PROTO_FILE_PATH' not found."
    exit 1
else
    echo "      OK: User Proto file found: $USER_PROTO_FILE_PATH"
fi
echo "INFO: --- Prerequisites Check Complete ---"
echo ""

# --- Test Steps ---
echo "INFO: --- Test Execution ---"

# 1. Create a temporary file to send
echo "[1] Creating temporary test file at $TEST_FILE_PATH"
echo "$TEST_FILE_CONTENT" > "$TEST_FILE_PATH"
if [ ! -f "$TEST_FILE_PATH" ]; then
    echo "ERROR: Failed to create temporary test file."
    exit 1
fi
echo "    Content: \"$TEST_FILE_CONTENT\""

# 2. Start the server in the background
echo "[2] Starting server '$SERVER_BINARY' in the background..."
"$SERVER_BINARY" &
SERVER_PID=$!
echo "    Server started with PID $SERVER_PID."
echo "    Waiting for server to initialize (5 seconds)..."
sleep 5 # Give the server a moment to start up

# Check if server is still running
if ! ps -p "$SERVER_PID" > /dev/null; then
    echo "ERROR: Server with PID $SERVER_PID failed to start or exited prematurely. Check server logs."
    SERVER_PID="" # Clear PID to prevent issues in cleanup
    exit 1
fi
echo "    Server appears to be running."

# 3. Register User
echo ""
echo "[3] Registering user '$TEST_USERNAME'..."
GRPcurl_REGISTER_ARGS=(
    -plaintext
    -import-path "$PROTO_IMPORT_PATH"
    -proto "$(basename "$USER_PROTO_FILE_PATH")"
    -d "{\"username\": \"$TEST_USERNAME\", \"password\": \"$TEST_PASSWORD\", \"likes\": \"$TEST_LIKES\"}"
    "$GRPC_ADDRESS"
    "$USER_SERVICE_NAME/$REGISTER_METHOD_NAME"
)

echo "    Executing: grpcurl ${GRPcurl_REGISTER_ARGS[*]}"
REGISTER_RESPONSE_JSON=$(grpcurl "${GRPcurl_REGISTER_ARGS[@]}" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$REGISTER_RESPONSE_JSON" ]; then
    REG_SUCCESS=$(echo "$REGISTER_RESPONSE_JSON" | jq -r .success)
    USER_ID=$(echo "$REGISTER_RESPONSE_JSON" | jq -r .user_id) # 假设响应中有 user_id
    if [ "$REG_SUCCESS" == "true" ]; then
        echo "    INFO: Registration successful for user '$TEST_USERNAME'. User ID: $USER_ID"
        REGISTRATION_SUCCESSFUL=true
    else
        ERROR_MSG=$(echo "$REGISTER_RESPONSE_JSON" | jq -r .error_message)
        echo "    ERROR: Registration call succeeded but registration failed by server."
        echo "    Response: $REGISTER_RESPONSE_JSON"
        echo "    Server Error: $ERROR_MSG"
        echo "    Skipping Login and SendFile tests."
    fi
else
    echo "    ERROR: Registration gRPC call failed."
    echo "    Attempted command: grpcurl ${GRPcurl_REGISTER_ARGS[*]}"
    echo "    Response (if any): $REGISTER_RESPONSE_JSON"
    echo "    Check server logs and grpcurl output if you run the command manually."
    echo "    Skipping Login and SendFile tests."
fi


# 4. Login to User Service to get Auth Token (only if registration was successful)
if [ "$REGISTRATION_SUCCESSFUL" != "true" ]; then
    echo ""
    echo "[4] SKIPPED: Login because registration failed or was skipped."
else
    echo ""
    echo "[4] Logging in to $USER_SERVICE_NAME/$LOGIN_METHOD_NAME to retrieve auth token..."
    GRPcurl_LOGIN_ARGS=(
        -plaintext
        -import-path "$PROTO_IMPORT_PATH"
        -proto "$(basename "$USER_PROTO_FILE_PATH")"
        -d "{\"username\": \"$TEST_USERNAME\", \"password\": \"$TEST_PASSWORD\"}"
        "$GRPC_ADDRESS"
        "$USER_SERVICE_NAME/$LOGIN_METHOD_NAME"
    )

    echo "    Executing: grpcurl ${GRPcurl_LOGIN_ARGS[*]}"
    LOGIN_RESPONSE_JSON=$(grpcurl "${GRPcurl_LOGIN_ARGS[@]}" 2>/dev/null)

    if [ $? -eq 0 ] && [ -n "$LOGIN_RESPONSE_JSON" ]; then
        AUTH_TOKEN=$(echo "$LOGIN_RESPONSE_JSON" | jq -r .token)
        if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
            echo "    INFO: Login successful. Token retrieved: $AUTH_TOKEN"
        else
            echo "    ERROR: Login call succeeded but token not found in response or is null."
            echo "    Response: $LOGIN_RESPONSE_JSON"
            echo "    Skipping SendFile test."
        fi
    else
        echo "    ERROR: Login gRPC call failed."
        echo "    Attempted command: grpcurl ${GRPcurl_LOGIN_ARGS[*]}"
        echo "    Response (if any): $LOGIN_RESPONSE_JSON"
        echo "    Check server logs and grpcurl output if you run the command manually."
        echo "    Skipping SendFile test."
    fi
fi

# 5. Make a gRPC call to SystemService/SendFile using the retrieved token (if available and registration was successful)
if [ "$REGISTRATION_SUCCESSFUL" != "true" ] || [ -z "$AUTH_TOKEN" ]; then
    echo ""
    echo "[5] SKIPPED: SendFile gRPC call because registration failed, was skipped, or auth token was not retrieved."
else
    echo ""
    echo "[5] Making gRPC call to $SYSTEM_SERVICE_NAME/$SYSTEM_METHOD_NAME on $GRPC_ADDRESS"
    echo "    Request: SendFile with file_path: $TEST_FILE_PATH"
    echo "    Using Auth Token: $AUTH_TOKEN"

    GRPcurl_SENDFILE_ARGS=(
        -plaintext
        -import-path "$PROTO_IMPORT_PATH"
        -proto "$(basename "$SYSTEM_PROTO_FILE_PATH")"
        -d "{\"file_path\": \"$TEST_FILE_PATH\"}"
        -H "Authorization:$AUTH_TOKEN"
        "$GRPC_ADDRESS"
        "$SYSTEM_SERVICE_NAME/$SYSTEM_METHOD_NAME"
    )

    echo "    Executing: grpcurl ${GRPcurl_SENDFILE_ARGS[*]}"
    if grpcurl "${GRPcurl_SENDFILE_ARGS[@]}"; then
        echo "    INFO: SendFile gRPC call initiated successfully. Server streamed response (if any) above."
    else
        echo "    ERROR: SendFile gRPC call failed. Check server logs and grpcurl output above."
        echo "    WARNING: SendFile gRPC call failed, Jaeger traces might not be generated as expected."
    fi
fi

# 6. Check Jaeger
echo ""
echo "[6] ACTION REQUIRED: Check Jaeger for Traces"
echo "    Please open your Jaeger UI (e.g., http://localhost:16686)."
echo "    - Select your service (e.g., 'tx-service', or as configured in cfg.Jaeger.ServiceName / tracer.InitJaeger)."
echo "    - Look for operations named '$REGISTER_METHOD_NAME', '$LOGIN_METHOD_NAME', and '$SYSTEM_METHOD_NAME'."
echo "    - Verify that the trace details (spans, tags, logs) look correct."
echo "    - If you don't see traces, check your server logs for any Jaeger initialization errors or other issues."
echo ""
echo "    Press [Enter] to continue and stop the server..."
read -r

echo "INFO: --- Test Execution Complete ---"
# Cleanup will be handled by the trap

exit 0
```