package bConst

import xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"

const (
	CtxAuthClientKey xCtx.ContextKey = "grpc_auth_client"
	CtxAuthUserKey   xCtx.ContextKey = "auth_userinfo"
)
