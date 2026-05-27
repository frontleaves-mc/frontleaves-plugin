package bConst

import xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"

const (
	EnvGrpcSecretKey     xEnv.EnvKey = "YGGLEAF_GRPC_SECRET_KEY"      // gRPC 服务间调用的共享密钥
	EnvYggleafGrpcHost   xEnv.EnvKey = "YGGLEAF_GRPC_HOST"            // Yggleaf gRPC 服务地址
	EnvYggleafGrpcPort   xEnv.EnvKey = "YGGLEAF_GRPC_PORT"            // Yggleaf gRPC 服务端口
	EnvMatrixManageInterval xEnv.EnvKey = "MATRIX_MANAGE_INTERVAL"    // Matrix 管理任务执行间隔，默认值为 5s，格式为 Go duration string（如 "5s"、"2s"、"500ms"）

	// 邮件服务
	EnvEmailHost     xEnv.EnvKey = "EMAIL_HOST"                       // SMTP 服务器地址
	EnvEmailPort     xEnv.EnvKey = "EMAIL_PORT"                       // SMTP 服务器端口
	EnvEmailUser     xEnv.EnvKey = "EMAIL_USERNAME"                   // SMTP 登录用户名
	EnvEmailPass     xEnv.EnvKey = "EMAIL_PASSWORD"                   // SMTP 登录密码
	EnvEmailFromName xEnv.EnvKey = "EMAIL_FROM_NAME"                  // 邮件发件人名称
	EnvEmailFrom     xEnv.EnvKey = "EMAIL_FROM_ADDRESS"               // 邮件发件人地址
)
