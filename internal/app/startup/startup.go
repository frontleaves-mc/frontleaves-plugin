package startup

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xEmail "github.com/bamboo-services/bamboo-base-go/plugins/email"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type reg struct {
	ctx context.Context
}

func newInit() *reg {
	return &reg{ctx: context.Background()}
}

func Init() (context.Context, []xRegNode.RegNodeList) {
	businessReg := newInit()
	regNode := []xRegNode.RegNodeList{
		{Key: xCtx.DatabaseKey, Node: businessReg.databaseInit},
		{Key: xCtx.RedisClientKey, Node: businessReg.nosqlInit},
		{Key: xCtx.EmailClientKey, Node: xEmail.InitClient},
		{Key: bConst.CtxAuthClientKey, Node: businessReg.grpcAuthClientInit},
		{Key: xCtx.Exec, Node: businessReg.businessDataPrepare},
	}

	return businessReg.ctx, regNode
}
