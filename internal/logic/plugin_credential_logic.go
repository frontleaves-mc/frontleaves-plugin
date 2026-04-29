package logic

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xUtil "github.com/bamboo-services/bamboo-base-go/common/utility"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiPC "github.com/frontleaves-mc/frontleaves-plugin/api/plugin_credential"
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
// 按插件名称查询凭证，比对密钥。
func (l *PluginCredentialLogic) Authenticate(ctx context.Context, pluginName, secretKey string) (*entity.PluginCredential, *xError.Error) {
	l.log.Info(ctx, "Authenticate - 验证插件身份")

	cred, xErr := l.repo.pluginCredential.GetByName(ctx, pluginName)
	if xErr != nil {
		return nil, xError.NewError(ctx, xError.Unauthorized, "插件凭证无效", true)
	}

	if cred.SecretKey != secretKey {
		return nil, xError.NewError(ctx, xError.Unauthorized, "插件密钥不匹配", true)
	}

	return cred, nil
}

// Create 创建插件凭证
func (l *PluginCredentialLogic) Create(ctx context.Context, name, description string) (*apiPC.PluginCredentialResponse, *xError.Error) {
	l.log.Info(ctx, "Create - 创建插件凭证")

	secretKey := xUtil.Security().GenerateLongKey()

	cred := &entity.PluginCredential{
		Name:        name,
		SecretKey:   secretKey,
		Description: description,
	}

	if xErr := l.repo.pluginCredential.Create(ctx, cred); xErr != nil {
		return nil, xErr
	}

	return l.toResponse(cred, true), nil
}

// List 查询插件凭证列表
func (l *PluginCredentialLogic) List(ctx context.Context, page, pageSize int) ([]apiPC.PluginCredentialResponse, int64, *xError.Error) {
	l.log.Info(ctx, "List - 查询插件凭证列表")

	creds, total, xErr := l.repo.pluginCredential.List(ctx, page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiPC.PluginCredentialResponse
	for _, cred := range creds {
		resp = append(resp, *l.toResponse(&cred, false))
	}
	return resp, total, nil
}

// GetByID 按 ID 查询插件凭证
func (l *PluginCredentialLogic) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*apiPC.PluginCredentialResponse, *xError.Error) {
	l.log.Info(ctx, "GetByID - 查询插件凭证")

	cred, xErr := l.repo.pluginCredential.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toResponse(cred, false), nil
}

// UpdateDescription 更新插件凭证描述
func (l *PluginCredentialLogic) UpdateDescription(ctx context.Context, id xSnowflake.SnowflakeID, description string) (*apiPC.PluginCredentialResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateDescription - 更新插件凭证描述")

	updates := map[string]interface{}{
		"description": description,
	}
	if xErr := l.repo.pluginCredential.Update(ctx, id, updates); xErr != nil {
		return nil, xErr
	}

	cred, xErr := l.repo.pluginCredential.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toResponse(cred, false), nil
}

// ResetSecretKey 重置插件密钥
func (l *PluginCredentialLogic) ResetSecretKey(ctx context.Context, id xSnowflake.SnowflakeID) (*apiPC.PluginCredentialResponse, *xError.Error) {
	l.log.Info(ctx, "ResetSecretKey - 重置插件密钥")

	newKey := xUtil.Security().GenerateLongKey()

	if xErr := l.repo.pluginCredential.UpdateSecretKey(ctx, id, newKey); xErr != nil {
		return nil, xErr
	}

	cred, xErr := l.repo.pluginCredential.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toResponse(cred, true), nil
}

// Delete 删除插件凭证
func (l *PluginCredentialLogic) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "Delete - 删除插件凭证")
	return l.repo.pluginCredential.Delete(ctx, id)
}

func maskSecretKey(key string) string {
	if len(key) < 20 {
		return "****"
	}
	return key[:8] + "****" + key[len(key)-8:]
}

func (l *PluginCredentialLogic) toResponse(cred *entity.PluginCredential, showFullKey bool) *apiPC.PluginCredentialResponse {
	secretKey := cred.SecretKey
	if !showFullKey {
		secretKey = maskSecretKey(cred.SecretKey)
	}
	return &apiPC.PluginCredentialResponse{
		ID:          cred.ID.String(),
		Name:        cred.Name,
		Description: cred.Description,
		SecretKey:   secretKey,
		CreatedAt:   cred.CreatedAt,
	}
}
