package bConst

import xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"

const (
	EnvGrpcSecretKey     xEnv.EnvKey = "YGGLEAF_GRPC_SECRET_KEY"      // gRPC 服务间调用的共享密钥
	EnvYggleafGrpcHost   xEnv.EnvKey = "YGGLEAF_GRPC_HOST"            // Yggleaf gRPC 服务地址
	EnvYggleafGrpcPort   xEnv.EnvKey = "YGGLEAF_GRPC_PORT"            // Yggleaf gRPC 服务端口
	EnvMatrixManageInterval xEnv.EnvKey = "MATRIX_MANAGE_INTERVAL"    // Matrix 管理任务执行间隔，默认值为 5s，格式为 Go duration string（如 "5s"、"2s"、"500ms"）
)
