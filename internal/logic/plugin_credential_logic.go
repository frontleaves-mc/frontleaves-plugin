package logic

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type pluginCredentialRepo struct {
	pluginCredential *repository.PluginCredentialRepo
}

type PluginCredentialLogic struct {
	logic
	repo pluginCredentialRepo
}

func NewPluginCredentialLogic(ctx context.Context) *PluginCredentialLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &PluginCredentialLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "PluginCredentialLogic"),
		},
		repo: pluginCredentialRepo{
			pluginCredential: repository.NewPluginCredentialRepo(db),
		},
	}
}

// Authenticate 验证插件身份
//
// 按插件名称查询凭证，检查是否启用，比对密钥。
func (l *PluginCredentialLogic) Authenticate(ctx context.Context, pluginName, secretKey string) (*entity.PluginCredential, *xError.Error) {
	l.log.Info(ctx, "Authenticate - 验证插件身份")

	cred, xErr := l.repo.pluginCredential.GetByName(ctx, pluginName)
	if xErr != nil {
		return nil, xError.NewError(ctx, xError.Unauthorized, "插件凭证无效", true)
	}

	if !cred.IsActive {
		return nil, xError.NewError(ctx, xError.PermissionDenied, "插件已被禁用", true)
	}

	if cred.SecretKey != secretKey {
		return nil, xError.NewError(ctx, xError.Unauthorized, "插件密钥不匹配", true)
	}

	return cred, nil
}
