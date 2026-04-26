package grpc

import (
	"context"
	"fmt"
	"log/slog"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/constant"
	authpb "github.com/frontleaves-mc/frontleaves-yggleaf/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type AuthClient struct {
	client   authpb.AuthServiceClient
	conn     *grpc.ClientConn
	accessID string
	secretKey string
	log      *xLog.LogNamedLogger
}

func NewAuthClient(ctx context.Context) (*AuthClient, error) {
	log := xLog.WithName(xLog.NamedINIT, "AuthClient")

	host := xEnv.GetEnvString("YGGLEAF_GRPC_HOST", "localhost")
	port := xEnv.GetEnvString("YGGLEAF_GRPC_PORT", "1119")
	accessID := xEnv.GetEnvString("PLUGIN_APP_ACCESS_ID", "")
	secretKey := xEnv.GetEnvString("PLUGIN_APP_SECRET_KEY", "")

	addr := fmt.Sprintf("%s:%s", host, port)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("连接 yggleaf gRPC 失败: %w", err)
	}

	client := authpb.NewAuthServiceClient(conn)
	log.Info(ctx, "认证服务 gRPC 客户端初始化成功", slog.String("addr", addr))

	return &AuthClient{
		client:    client,
		conn:      conn,
		accessID:  accessID,
		secretKey: secretKey,
		log:       log,
	}, nil
}

func (c *AuthClient) ValidateToken(ctx context.Context, accessToken string) (*authpb.ValidateTokenResponse, error) {
	md := metadata.Pairs(
		xGrpcConst.MetadataAppAccessID.String(), c.accessID,
		xGrpcConst.MetadataAppSecretKey.String(), c.secretKey,
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{
		AccessToken: accessToken,
	})
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *AuthClient) GetUserInfo(ctx context.Context, userID string) (*authpb.GetUserInfoResponse, error) {
	md := metadata.Pairs(
		xGrpcConst.MetadataAppAccessID.String(), c.accessID,
		xGrpcConst.MetadataAppSecretKey.String(), c.secretKey,
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.client.GetUserInfo(ctx, &authpb.GetUserInfoRequest{
		UserId: userID,
	})
}
