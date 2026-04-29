package prepare

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

// preparePluginCredential 创建默认插件凭证种子数据
func (p *Prepare) preparePluginCredential() {
	p.log.Info(p.ctx, "准备插件凭证种子数据...")

	credentials := []struct {
		name      string
		secretKey string
	}{
		{name: "server-status", secretKey: generateSecretKey()},
		{name: "title-plugin", secretKey: generateSecretKey()},
	}

	for _, cred := range credentials {
		var existing entity.PluginCredential
		result := p.db.WithContext(p.ctx).Where("name = ?", cred.name).First(&existing)
		if result.Error == nil {
			p.log.Info(p.ctx, "插件凭证已存在，跳过: "+cred.name)
			continue
		}

		pluginCred := &entity.PluginCredential{
			Name:      cred.name,
			SecretKey: cred.secretKey,
			IsActive:  true,
		}
		pluginCred.ID = xSnowflake.GenerateID(bConst.GenePluginCredential)

		if err := p.db.WithContext(p.ctx).Create(pluginCred).Error; err != nil {
			p.log.Warn(p.ctx, "创建插件凭证失败: "+cred.name+" - "+err.Error())
			continue
		}

		p.log.Info(p.ctx, "创建插件凭证: "+cred.name+" (secret: "+cred.secretKey+")")
	}
}

func generateSecretKey() string {
	return xSnowflake.GenerateID(bConst.GenePluginCredential).String() + "-" +
		xSnowflake.GenerateID(bConst.GenePluginCredential).String()
}
