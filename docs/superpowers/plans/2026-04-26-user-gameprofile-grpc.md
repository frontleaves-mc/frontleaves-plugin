# User + GameProfile 重命名 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 User 实体 + Player 全量重命名为 GameProfile + 扩展 gRPC GetUserInfo

**Architecture:** 分层改造 — 从 Entity 层开始自底向上重命名，最后修改 Route 和 Middleware。User 为新增 Snowflake 实体，GameProfile 由 Player 改名而来，新增 UserID 外键关联。

**Tech Stack:** Go 1.25 + GORM + Gin + gRPC + PostgreSQL + Redis

---

### Task 1: 新增 GeneUser 常量 + User 实体

**Files:**
- Modify: `internal/constant/gene_number.go`
- Create: `internal/entity/user.go`

- [ ] **Step 1: 在 gene_number.go 新增 GeneUser = 31**

```go
// internal/constant/gene_number.go
package bConst

import xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"

const (
	GeneUser xSnowflake.Gene = 31 // 新增：用户

	Demo xSnowflake.Gene = 32

	// 称号系统 + 成就系统
	GeneTitle             xSnowflake.Gene = 33 // 称号定义
	GenePlayerTitle       xSnowflake.Gene = 34 // 玩家称号关联
	GeneAchievement       xSnowflake.Gene = 35 // 成就定义
	GenePlayerAchievement xSnowflake.Gene = 36 // 玩家成就
	GeneAchievementClaim  xSnowflake.Gene = 38 // 成就申领记录
)
```

- [ ] **Step 2: 创建 entity/user.go**

```go
// internal/entity/user.go
package entity

import (
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type User struct {
	xModels.BaseEntity
	Username string `gorm:"not null;type:varchar(64);comment:用户名" json:"username"`

	GameProfiles []GameProfile `gorm:"foreignKey:UserID" json:"-"`
}

func (_ *User) GetGene() xSnowflake.Gene {
	return bConst.GeneUser
}
```

- [ ] **Step 3: 编译验证 entity 包**

Run: `go build ./internal/entity/...`
Expected: 编译通过（GameProfile 引用暂不存在，先验证 User 自身无语法错误）

Wait — GameProfile 在 entity/user.go 中被引用但尚未创建。先跳过编译，等 Task 2 完成 GameProfile 后再一起编译。

- [ ] **Step 4: Commit**

```bash
git add internal/constant/gene_number.go internal/entity/user.go
git commit -m "feat(entity): 新增 GeneUser 常量和 User 实体"
```

---

### Task 2: Player 实体重命名为 GameProfile

**Files:**
- Delete: `internal/entity/player.go`
- Create: `internal/entity/game_profile.go`

> 注意：GameProfile 被其他实体通过外键引用，GORM 会通过 `references:UUID` 自动关联。表名从 `fp_player` 变更为 `fp_game_profile`，迁移脚本在后续 Task 处理。

- [ ] **Step 1: 删除旧文件 player.go**

Run: `rm internal/entity/player.go`

- [ ] **Step 2: 创建 game_profile.go**

```go
// internal/entity/game_profile.go
package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
)

type GameProfile struct {
	UserID     xSnowflake.SnowflakeID `gorm:"not null;comment:所属用户ID" json:"user_id"`
	UUID       uuid.UUID              `gorm:"type:uuid;primaryKey;comment:玩家UUID" json:"uuid"`
	Username   string                 `gorm:"not null;type:varchar(64);comment:MC用户名" json:"username"`
	GroupName  string                 `gorm:"not null;type:varchar(64);comment:当前权限组" json:"group_name"`
	ReportedAt time.Time              `gorm:"not null;type:timestamptz;comment:最后上报时间" json:"reported_at"`
	CreatedAt  time.Time              `gorm:"not null;type:timestamptz;autoCreateTime:milli;comment:创建时间" json:"-"`
	UpdatedAt  time.Time              `gorm:"not null;type:timestamptz;autoUpdateTime:milli;comment:更新时间" json:"-"`

	User *User `gorm:"foreignKey:UserID;references:ID" json:"-"`
}
```

变更要点：
- 新增 `UserID` 字段 + `User` 外键
- 移除了 `BeforeCreate`/`BeforeUpdate` hooks（autoCreateTime/autoUpdateTime 标签已足够）

- [ ] **Step 3: 验证 entity 包编译**

Run: `go build ./internal/entity/...`
Expected: 编译通过

- [ ] **Step 4: Commit**

```bash
git rm internal/entity/player.go
git add internal/entity/game_profile.go
git commit -m "refactor(entity): Player 重命名为 GameProfile，新增 UserID 外键"
```

---

### Task 3: 关联实体重命名（PlayerTitle / PlayerAchievement / PlayerAchievementClaim）

**Files:**
- Delete: `internal/entity/player_title.go`
- Create: `internal/entity/game_profile_title.go`
- Delete: `internal/entity/player_achievement.go`
- Create: `internal/entity/game_profile_achievement.go`
- Delete: `internal/entity/player_achievement_claim.go`
- Create: `internal/entity/game_profile_achievement_claim.go`

- [ ] **Step 1: 创建 game_profile_title.go**

```go
// internal/entity/game_profile_title.go
package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type TitleSource int16

const (
	TitleSourceAuto        TitleSource = 1
	TitleSourceGroup       TitleSource = 2
	TitleSourceAdmin       TitleSource = 3
	TitleSourceAchievement TitleSource = 4
)

func (s TitleSource) String() string {
	switch s {
	case TitleSourceAuto:
		return "自动获得"
	case TitleSourceGroup:
		return "权限组匹配"
	case TitleSourceAdmin:
		return "管理员分配"
	case TitleSourceAchievement:
		return "成就奖励"
	default:
		return "未知来源"
	}
}

type GameProfileTitle struct {
	xModels.BaseEntity
	GameProfileUUID uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_gpt_gameprofile_title;comment:玩家UUID" json:"game_profile_uuid"`
	TitleID         xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_gpt_gameprofile_title;index:idx_gpt_title;comment:称号ID" json:"title_id"`
	Source          TitleSource            `gorm:"not null;type:smallint;comment:获得来源" json:"source"`
	IsEquipped      bool                   `gorm:"not null;default:false;index:idx_gpt_equipped;comment:是否装备" json:"is_equipped"`
	GrantedAt       time.Time              `gorm:"not null;type:timestamptz;comment:授予时间" json:"granted_at"`

	Title        *Title        `gorm:"foreignKey:TitleID;references:ID;constraint:OnDelete:CASCADE" json:"title,omitempty"`
	GameProfile  *GameProfile  `gorm:"foreignKey:GameProfileUUID;references:UUID;constraint:OnDelete:CASCADE" json:"game_profile,omitempty"`
}

func (_ *GameProfileTitle) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerTitle
}
```

- [ ] **Step 2: 创建 game_profile_achievement.go**

```go
// internal/entity/game_profile_achievement.go
package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type AchievementStatus int16

const (
	AchievementStatusInProgress AchievementStatus = 0
	AchievementStatusCompleted  AchievementStatus = 1
	AchievementStatusClaimed    AchievementStatus = 2
)

func (s AchievementStatus) String() string {
	switch s {
	case AchievementStatusInProgress:
		return "进行中"
	case AchievementStatusCompleted:
		return "已完成"
	case AchievementStatusClaimed:
		return "已领奖"
	default:
		return "未知状态"
	}
}

type GameProfileAchievement struct {
	xModels.BaseEntity
	GameProfileUUID uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_gpa_gameprofile_ach;comment:玩家UUID" json:"game_profile_uuid"`
	AchievementID   xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_gpa_gameprofile_ach;index:idx_gpa_ach;comment:成就ID" json:"achievement_id"`
	Status          AchievementStatus      `gorm:"not null;type:smallint;default:0;comment:状态" json:"status"`
	Progress        int64                  `gorm:"not null;default:0;comment:当前进度" json:"progress"`
	CompletedAt     *time.Time             `gorm:"type:timestamptz;comment:完成时间" json:"completed_at,omitempty"`

	Achievement  *Achievement  `gorm:"foreignKey:AchievementID;references:ID;constraint:OnDelete:CASCADE" json:"achievement,omitempty"`
	GameProfile  *GameProfile  `gorm:"foreignKey:GameProfileUUID;references:UUID;constraint:OnDelete:CASCADE" json:"game_profile,omitempty"`
}

func (_ *GameProfileAchievement) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerAchievement
}
```

- [ ] **Step 3: 创建 game_profile_achievement_claim.go**

```go
// internal/entity/game_profile_achievement_claim.go
package entity

import (
	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type GameProfileAchievementClaim struct {
	xModels.BaseEntity
	GameProfileUUID uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_gpac_gameprofile_ach;comment:玩家UUID" json:"game_profile_uuid"`
	AchievementID   xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_gpac_gameprofile_ach;comment:成就ID" json:"achievement_id"`
	TitleClaimed    bool                   `gorm:"not null;default:false;comment:称号奖励是否已发放" json:"title_claimed"`

	Achievement  *Achievement  `gorm:"foreignKey:AchievementID;references:ID;constraint:OnDelete:CASCADE" json:"achievement,omitempty"`
	GameProfile  *GameProfile  `gorm:"foreignKey:GameProfileUUID;references:UUID;constraint:OnDelete:CASCADE" json:"game_profile,omitempty"`
}

func (_ *GameProfileAchievementClaim) GetGene() xSnowflake.Gene {
	return bConst.GeneAchievementClaim
}
```

- [ ] **Step 4: 删除旧文件**

```bash
rm internal/entity/player_title.go
rm internal/entity/player_achievement.go
rm internal/entity/player_achievement_claim.go
```

- [ ] **Step 5: 验证编译**

Run: `go build ./internal/entity/...`
Expected: 编译通过

- [ ] **Step 6: Commit**

```bash
git rm internal/entity/player_title.go internal/entity/player_achievement.go internal/entity/player_achievement_claim.go
git add internal/entity/game_profile_title.go internal/entity/game_profile_achievement.go internal/entity/game_profile_achievement_claim.go
git commit -m "refactor(entity): 关联实体 Player* 重命名为 GameProfile*"
```

---

### Task 4: 新增 UserRepo + 重命名 PlayerRepo → GameProfileRepo

**Files:**
- Create: `internal/repository/user_repo.go`
- Delete: `internal/repository/player_repo.go`
- Create: `internal/repository/game_profile_repo.go`

- [ ] **Step 1: 创建 user_repo.go**

```go
// internal/repository/user_repo.go
package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewUserRepo(db *gorm.DB, rdb *redis.Client) *UserRepo {
	return &UserRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "UserRepo"),
	}
}

func (r *UserRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.User, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询用户")
	var user entity.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "用户不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询用户失败", false, err)
	}
	return &user, nil
}

func (r *UserRepo) Upsert(ctx context.Context, user *entity.User) *xError.Error {
	r.log.Info(ctx, "Upsert - 创建或更新用户")
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"username"}),
	}).Create(user).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "同步用户失败", false, err)
	}
	return nil
}
```

- [ ] **Step 2: 创建 game_profile_repo.go**

```go
// internal/repository/game_profile_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type GameProfileRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewGameProfileRepo(db *gorm.DB, rdb *redis.Client) *GameProfileRepo {
	return &GameProfileRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileRepo"),
	}
}

func (r *GameProfileRepo) GetByUUID(ctx context.Context, profileUUID uuid.UUID) (*entity.GameProfile, *xError.Error) {
	r.log.Info(ctx, "GetByUUID - 查询玩家信息")
	var gp entity.GameProfile
	if err := r.db.WithContext(ctx).Where("uuid = ?", profileUUID).First(&gp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "玩家不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家失败", false, err)
	}
	return &gp, nil
}

func (r *GameProfileRepo) CreateOrUpdate(ctx context.Context, gp *entity.GameProfile) *xError.Error {
	r.log.Info(ctx, "CreateOrUpdate - 创建或更新玩家")
	result := r.db.WithContext(ctx).Where("uuid = ?", gp.UUID).
		Assign(map[string]interface{}{
			"user_id":     gp.UserID,
			"username":    gp.Username,
			"group_name":  gp.GroupName,
			"reported_at": gp.ReportedAt,
		}).
		FirstOrCreate(gp)
	if result.Error != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建或更新玩家失败", false, result.Error)
	}
	return nil
}

func (r *GameProfileRepo) List(ctx context.Context, page, pageSize int) ([]entity.GameProfile, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询玩家列表")
	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.GameProfile{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家总数失败", false, err)
	}
	var gps []entity.GameProfile
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&gps).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家列表失败", false, err)
	}
	return gps, total, nil
}
```

- [ ] **Step 3: 删除旧文件**

```bash
rm internal/repository/player_repo.go
```

- [ ] **Step 4: 验证 repository 包编译**

Run: `go build ./internal/repository/...`
Expected: 编译失败 — 其他 repo 文件仍引用 `entity.PlayerTitle` 等旧名，属于预期，下一 Task 修复。

- [ ] **Step 5: Commit**

```bash
git rm internal/repository/player_repo.go
git add internal/repository/user_repo.go internal/repository/game_profile_repo.go
git commit -m "refactor(repo): 新增 UserRepo，PlayerRepo 重命名为 GameProfileRepo"
```

---

### Task 5: 重命名关联 Repository（PlayerTitleRepo / PlayerAchievementRepo / PlayerAchievementClaimRepo）

**Files:**
- Delete: `internal/repository/player_title_repo.go`
- Create: `internal/repository/game_profile_title_repo.go`
- Delete: `internal/repository/player_achievement_repo.go`
- Create: `internal/repository/game_profile_achievement_repo.go`
- Delete: `internal/repository/player_achievement_claim_repo.go`
- Create: `internal/repository/game_profile_achievement_claim_repo.go`

这三个文件的变更内容为纯机械替换：
- `PlayerTitleRepo` → `GameProfileTitleRepo`
- `PlayerAchievementRepo` → `GameProfileAchievementRepo`
- `PlayerAchievementClaimRepo` → `GameProfileAchievementClaimRepo`
- `entity.PlayerTitle` → `entity.GameProfileTitle`
- `entity.PlayerAchievement` → `entity.GameProfileAchievement`
- `entity.PlayerAchievementClaim` → `entity.GameProfileAchievementClaim`
- `player_uuid` → `game_profile_uuid`（GORM 查询条件中的列名）
- `playerTitles`/`pas` 变量名 → `gpts`/`gpas`
- `"PlayerTitleRepo"` → `"GameProfileTitleRepo"` 等 NamedLogger 字符串

- [ ] **Step 1: 创建 game_profile_title_repo.go**

```go
// internal/repository/game_profile_title_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type GameProfileTitleRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewGameProfileTitleRepo(db *gorm.DB, rdb *redis.Client) *GameProfileTitleRepo {
	return &GameProfileTitleRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileTitleRepo"),
	}
}

func (r *GameProfileTitleRepo) Create(ctx context.Context, gpt *entity.GameProfileTitle) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家称号关联")
	if err := r.db.WithContext(ctx).Create(gpt).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家称号关联失败", false, err)
	}
	return nil
}

func (r *GameProfileTitleRepo) Delete(ctx context.Context, profileUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除玩家称号关联")
	if err := r.db.WithContext(ctx).Where("game_profile_uuid = ? AND title_id = ?", profileUUID, titleID).Delete(&entity.GameProfileTitle{}).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除玩家称号关联失败", false, err)
	}
	return nil
}

func (r *GameProfileTitleRepo) GetByGameProfileUUID(ctx context.Context, profileUUID uuid.UUID) ([]entity.GameProfileTitle, *xError.Error) {
	r.log.Info(ctx, "GetByGameProfileUUID - 查询玩家拥有的称号")
	var gpts []entity.GameProfileTitle
	if err := r.db.WithContext(ctx).Preload("Title").Where("game_profile_uuid = ?", profileUUID).Find(&gpts).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家称号列表失败", false, err)
	}
	return gpts, nil
}

func (r *GameProfileTitleRepo) GetEquippedByGameProfileUUID(ctx context.Context, profileUUID uuid.UUID) (*entity.GameProfileTitle, *xError.Error) {
	r.log.Info(ctx, "GetEquippedByGameProfileUUID - 查询玩家装备的称号")
	var gpt entity.GameProfileTitle
	if err := r.db.WithContext(ctx).Preload("Title").Where("game_profile_uuid = ? AND is_equipped = ?", profileUUID, true).First(&gpt).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询装备称号失败", false, err)
	}
	return &gpt, nil
}

func (r *GameProfileTitleRepo) HasTitle(ctx context.Context, profileUUID uuid.UUID, titleID xSnowflake.SnowflakeID) (bool, *xError.Error) {
	r.log.Info(ctx, "HasTitle - 检查玩家是否拥有称号")
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ? AND title_id = ?", profileUUID, titleID).Count(&count).Error; err != nil {
		return false, xError.NewError(nil, xError.DatabaseError, "检查玩家称号失败", false, err)
	}
	return count > 0, nil
}

func (r *GameProfileTitleRepo) EquipTitle(ctx context.Context, db *gorm.DB, profileUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "EquipTitle - 装备称号")
	tx := db.WithContext(ctx)
	if err := tx.Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ?", profileUUID).Update("is_equipped", false).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "卸下旧称号失败", false, err)
	}
	if err := tx.Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ? AND title_id = ?", profileUUID, titleID).Update("is_equipped", true).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "装备称号失败", false, err)
	}
	return nil
}

func (r *GameProfileTitleRepo) UnequipTitle(ctx context.Context, db *gorm.DB, profileUUID uuid.UUID) *xError.Error {
	r.log.Info(ctx, "UnequipTitle - 卸下称号")
	if err := db.WithContext(ctx).Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ? AND is_equipped = ?", profileUUID, true).Update("is_equipped", false).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "卸下称号失败", false, err)
	}
	return nil
}
```

- [ ] **Step 2: 创建 game_profile_achievement_repo.go**

```go
// internal/repository/game_profile_achievement_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type GameProfileAchievementRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewGameProfileAchievementRepo(db *gorm.DB, rdb *redis.Client) *GameProfileAchievementRepo {
	return &GameProfileAchievementRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileAchievementRepo"),
	}
}

func (r *GameProfileAchievementRepo) Create(ctx context.Context, gpa *entity.GameProfileAchievement) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家成就")
	if err := r.db.WithContext(ctx).Create(gpa).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家成就失败", false, err)
	}
	return nil
}

func (r *GameProfileAchievementRepo) GetByGameProfileAndAchievement(ctx context.Context, profileUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*entity.GameProfileAchievement, *xError.Error) {
	r.log.Info(ctx, "GetByGameProfileAndAchievement - 查询玩家成就")
	var gpa entity.GameProfileAchievement
	if err := r.db.WithContext(ctx).Preload("Achievement").Where("game_profile_uuid = ? AND achievement_id = ?", profileUUID, achievementID).First(&gpa).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家成就失败", false, err)
	}
	return &gpa, nil
}

func (r *GameProfileAchievementRepo) ListByGameProfile(ctx context.Context, profileUUID uuid.UUID) ([]entity.GameProfileAchievement, *xError.Error) {
	r.log.Info(ctx, "ListByGameProfile - 查询玩家所有成就")
	var gpas []entity.GameProfileAchievement
	if err := r.db.WithContext(ctx).Preload("Achievement").Where("game_profile_uuid = ?", profileUUID).Order("created_at DESC").Find(&gpas).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家成就列表失败", false, err)
	}
	return gpas, nil
}

func (r *GameProfileAchievementRepo) UpdateProgress(ctx context.Context, id xSnowflake.SnowflakeID, progress int64) *xError.Error {
	r.log.Info(ctx, "UpdateProgress - 更新成就进度")
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileAchievement{}).Where("id = ?", id).Update("progress", progress).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就进度失败", false, err)
	}
	return nil
}

func (r *GameProfileAchievementRepo) UpdateStatus(ctx context.Context, id xSnowflake.SnowflakeID, status entity.AchievementStatus) *xError.Error {
	r.log.Info(ctx, "UpdateStatus - 更新成就状态")
	updates := map[string]interface{}{"status": status}
	if status == entity.AchievementStatusCompleted {
		updates["completed_at"] = gorm.Expr("NOW()")
	}
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileAchievement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就状态失败", false, err)
	}
	return nil
}
```

- [ ] **Step 3: 创建 game_profile_achievement_claim_repo.go**

```go
// internal/repository/game_profile_achievement_claim_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type GameProfileAchievementClaimRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewGameProfileAchievementClaimRepo(db *gorm.DB, rdb *redis.Client) *GameProfileAchievementClaimRepo {
	return &GameProfileAchievementClaimRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileAchievementClaimRepo"),
	}
}

func (r *GameProfileAchievementClaimRepo) Create(ctx context.Context, claim *entity.GameProfileAchievementClaim) *xError.Error {
	r.log.Info(ctx, "Create - 创建申领记录")
	if err := r.db.WithContext(ctx).Create(claim).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建申领记录失败", false, err)
	}
	return nil
}

func (r *GameProfileAchievementClaimRepo) GetByGameProfileAndAchievement(ctx context.Context, profileUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*entity.GameProfileAchievementClaim, *xError.Error) {
	r.log.Info(ctx, "GetByGameProfileAndAchievement - 查询申领记录")
	var claim entity.GameProfileAchievementClaim
	if err := r.db.WithContext(ctx).Where("game_profile_uuid = ? AND achievement_id = ?", profileUUID, achievementID).First(&claim).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询申领记录失败", false, err)
	}
	return &claim, nil
}

func (r *GameProfileAchievementClaimRepo) UpdateTitleClaimed(ctx context.Context, id xSnowflake.SnowflakeID, claimed bool) *xError.Error {
	r.log.Info(ctx, "UpdateTitleClaimed - 更新称号申领状态")
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileAchievementClaim{}).Where("id = ?", id).Update("title_claimed", claimed).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新申领状态失败", false, err)
	}
	return nil
}
```

- [ ] **Step 4: 删除旧文件**

```bash
rm internal/repository/player_title_repo.go
rm internal/repository/player_achievement_repo.go
rm internal/repository/player_achievement_claim_repo.go
```

- [ ] **Step 5: 验证编译**

Run: `go build ./internal/repository/...`
Expected: 编译通过

- [ ] **Step 6: Commit**

```bash
git rm internal/repository/player_title_repo.go internal/repository/player_achievement_repo.go internal/repository/player_achievement_claim_repo.go
git add internal/repository/game_profile_title_repo.go internal/repository/game_profile_achievement_repo.go internal/repository/game_profile_achievement_claim_repo.go
git commit -m "refactor(repo): 关联 Repository 重命名为 GameProfile*"
```

---

### Task 6: Logic 层重命名（PlayerLogic → GameProfileLogic + 新增 UserLogic）

**Files:**
- Delete: `internal/logic/player_logic.go`
- Create: `internal/logic/game_profile_logic.go`
- Create: `internal/logic/user_logic.go`

- [ ] **Step 1: 创建 user_logic.go**

```go
// internal/logic/user_logic.go
package logic

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
)

type userRepo struct {
	user *repository.UserRepo
}

type UserLogic struct {
	logic
	repo userRepo
}

func NewUserLogic(ctx context.Context) *UserLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &UserLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "UserLogic"),
		},
		repo: userRepo{
			user: repository.NewUserRepo(db, rdb),
		},
	}
}

func (l *UserLogic) Upsert(ctx *gin.Context, userID xSnowflake.SnowflakeID, username string) error {
	l.log.Info(ctx, "Upsert - 同步用户信息")
	user := &entity.User{
		BaseEntity: xModels.BaseEntity{ID: userID},
		Username:   username,
	}
	if xErr := l.repo.user.Upsert(ctx.Request.Context(), user); xErr != nil {
		l.log.Warn(ctx, "同步用户失败: "+xErr.Error())
		return xErr
	}
	return nil
}
```

- [ ] **Step 2: 创建 game_profile_logic.go**

```go
// internal/logic/game_profile_logic.go
package logic

import (
	"context"
	"time"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiPlayer "github.com/frontleaves-mc/frontleaves-plugin/api/player"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
)

type gameProfileRepo struct {
	gameProfile *repository.GameProfileRepo
}

type GameProfileLogic struct {
	logic
	repo gameProfileRepo
}

func NewGameProfileLogic(ctx context.Context) *GameProfileLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &GameProfileLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "GameProfileLogic"),
		},
		repo: gameProfileRepo{
			gameProfile: repository.NewGameProfileRepo(db, rdb),
		},
	}
}

func (l *GameProfileLogic) GetPlayer(ctx *gin.Context, playerUUID uuid.UUID) (*apiPlayer.PlayerResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayer - 查询玩家信息")
	gp, xErr := l.repo.gameProfile.GetByUUID(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		return nil, xErr
	}
	return &apiPlayer.PlayerResponse{
		UUID:       gp.UUID.String(),
		Username:   gp.Username,
		GroupName:  gp.GroupName,
		ReportedAt: gp.ReportedAt,
	}, nil
}

func (l *GameProfileLogic) UpdatePlayerGroup(ctx *gin.Context, playerUUID uuid.UUID, username, groupName string) *xError.Error {
	l.log.Info(ctx, "UpdatePlayerGroup - 更新玩家权限组")
	gp := &entity.GameProfile{
		UUID:       playerUUID,
		Username:   username,
		GroupName:  groupName,
		ReportedAt: time.Now(),
	}
	if xErr := l.repo.gameProfile.CreateOrUpdate(ctx.Request.Context(), gp); xErr != nil {
		return xErr
	}
	return nil
}

func (l *GameProfileLogic) ListPlayers(ctx *gin.Context, page, pageSize int) ([]apiPlayer.PlayerResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListPlayers - 查询玩家列表")
	gps, total, xErr := l.repo.gameProfile.List(ctx.Request.Context(), page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}
	var resp []apiPlayer.PlayerResponse
	for _, p := range gps {
		resp = append(resp, apiPlayer.PlayerResponse{
			UUID:       p.UUID.String(),
			Username:   p.Username,
			GroupName:  p.GroupName,
			ReportedAt: p.ReportedAt,
		})
	}
	return resp, total, nil
}

func (l *GameProfileLogic) Upsert(ctx *gin.Context, userID xSnowflake.SnowflakeID, gpUUID uuid.UUID, name string) error {
	l.log.Info(ctx, "Upsert - 同步 GameProfile")
	gp := &entity.GameProfile{
		UserID:     userID,
		UUID:       gpUUID,
		Username:   name,
		GroupName:  "PLAYER",
		ReportedAt: time.Now(),
	}
	if xErr := l.repo.gameProfile.CreateOrUpdate(ctx.Request.Context(), gp); xErr != nil {
		l.log.Warn(ctx, "同步 GameProfile 失败: "+xErr.Error())
		return xErr
	}
	return nil
}
```

- [ ] **Step 3: 删除旧文件**

```bash
rm internal/logic/player_logic.go
```

- [ ] **Step 4: 验证编译**

Run: `go build ./internal/logic/...`
Expected: 编译失败 — title_logic.go 和 achievement_logic.go 仍引用旧 Repository 名。下一 Task 修复。

- [ ] **Step 5: Commit**

```bash
git rm internal/logic/player_logic.go
git add internal/logic/game_profile_logic.go internal/logic/user_logic.go
git commit -m "refactor(logic): PlayerLogic 重命名为 GameProfileLogic，新增 UserLogic"
```

---

### Task 7: 更新 title_logic.go 和 achievement_logic.go 中的引用

**Files:**
- Modify: `internal/logic/title_logic.go`
- Modify: `internal/logic/achievement_logic.go`

这两个文件中涉及 `PlayerTitleRepo`、`PlayerAchievementRepo`、`PlayerAchievementClaimRepo`、`entity.PlayerTitle`、`entity.PlayerAchievement`、`entity.PlayerAchievementClaim` 等旧名，需全局替换。

- [ ] **Step 1: 更新 title_logic.go**

进行以下替换（使用 Edit 工具逐项替换）：
- `playerTitle *repository.PlayerTitleRepo` → `gameProfileTitle *repository.GameProfileTitleRepo`
- `repository.NewPlayerTitleRepo` → `repository.NewGameProfileTitleRepo`
- `entity.PlayerTitle` → `entity.GameProfileTitle`（所有出现处）
- `"PlayerUUID"` → `"GameProfileUUID"`（GORM Preload 字符串）
- `repo.playerTitle` → `repo.gameProfileTitle`

- [ ] **Step 2: 更新 achievement_logic.go**

进行以下替换：
- `playerAch *repository.PlayerAchievementRepo` → `gameProfileAch *repository.GameProfileAchievementRepo`
- `claim *repository.PlayerAchievementClaimRepo` → `claim *repository.GameProfileAchievementClaimRepo`
- `playerTitle *repository.PlayerTitleRepo` → `gameProfileTitle *repository.GameProfileTitleRepo`
- `repository.NewPlayerAchievementRepo` → `repository.NewGameProfileAchievementRepo`
- `repository.NewPlayerAchievementClaimRepo` → `repository.NewGameProfileAchievementClaimRepo`
- `repository.NewPlayerTitleRepo` → `repository.NewGameProfileTitleRepo`
- `entity.PlayerAchievement` → `entity.GameProfileAchievement`
- `entity.PlayerAchievementClaim` → `entity.GameProfileAchievementClaim`
- `entity.PlayerTitle` → `entity.GameProfileTitle`
- `"PlayerUUID"` → `"GameProfileUUID"`（GORM Preload 字符串）

- [ ] **Step 3: 验证编译**

Run: `go build ./internal/logic/...`
Expected: 编译通过

- [ ] **Step 4: Commit**

```bash
git add internal/logic/title_logic.go internal/logic/achievement_logic.go
git commit -m "refactor(logic): 更新 title/achievement logic 中的 Player 引用"
```

---

### Task 8: Handler 层重命名 + API DTO 重命名

**Files:**
- Delete: `internal/handler/player.go`
- Create: `internal/handler/game_profile.go`
- Delete: `api/player/request.go` + `api/player/response.go`
- Create: `api/game_profile/request.go` + `api/game_profile/response.go`
- Modify: `internal/handler/handler.go`（更新 service 结构体）
- Modify: `internal/handler/title_player.go`（更新引用）
- Modify: `internal/handler/achievement_player.go`（更新引用 + `:achId` → `:ach_id`）

- [ ] **Step 1: 创建 API DTO 新目录**

```bash
mkdir -p api/game_profile
```

- [ ] **Step 2: 创建 api/game_profile/response.go**

```go
// api/game_profile/response.go
package apiPlayer

import "time"

type PlayerResponse struct {
	UUID       string    `json:"uuid"`
	Username   string    `json:"username"`
	GroupName  string    `json:"group_name"`
	ReportedAt time.Time `json:"reported_at"`
}
```

> 注意：包名保持 `apiPlayer` 不变，避免改动 logic 层的 import 别名。仅文件路径变更。

- [ ] **Step 3: 创建 api/game_profile/request.go**

```go
// api/game_profile/request.go
package apiPlayer

type UpdatePlayerGroupRequest struct {
	GroupName string `json:"group_name" binding:"required"`
}
```

- [ ] **Step 4: 删除旧 API DTO 文件**

```bash
rm -rf api/player
```

- [ ] **Step 5: 创建 handler/game_profile.go**

```go
// internal/handler/game_profile.go
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiPlayer "github.com/frontleaves-mc/frontleaves-plugin/api/game_profile"
)

type GameProfileHandler handler

func NewGameProfileHandler(ctx context.Context) *GameProfileHandler {
	return NewHandler[GameProfileHandler](ctx, "GameProfileHandler")
}

// GetPlayer 查询玩家信息
//
// @Summary     [玩家] 查询玩家信息
// @Description 根据玩家 UUID 查询玩家详细信息
// @Tags        玩家接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse{data=apiPlayer.PlayerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                "玩家不存在"
// @Router      /game-profiles/:uuid [GET]
func (h *GameProfileHandler) GetPlayer(ctx *gin.Context) {
	h.log.Info(ctx, "GetPlayer - 查询玩家信息")
	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}
	player, xErr := h.service.gameProfileLogic.GetPlayer(ctx, playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}
	xResult.SuccessHasData(ctx, "查询成功", player)
}

// ListPlayers 查询玩家列表
//
// @Summary     [玩家] 查询玩家列表
// @Description 分页查询玩家列表
// @Tags        玩家接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /game-profiles [GET]
func (h *GameProfileHandler) ListPlayers(ctx *gin.Context) {
	h.log.Info(ctx, "ListPlayers - 查询玩家列表")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	players, total, xErr := h.service.gameProfileLogic.ListPlayers(ctx, page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}
	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      players,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdatePlayerGroup 更新玩家权限组
//
// @Summary     [超管] 更新玩家权限组
// @Description 更新指定玩家的权限组
// @Tags        玩家接口
// @Accept      json
// @Produce     json
// @Param       uuid      path  string                              true  "玩家UUID"
// @Param       request   body  apiPlayer.UpdatePlayerGroupRequest  true  "更新权限组请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /internal/game-profiles/:uuid/group [PUT]
func (h *GameProfileHandler) UpdatePlayerGroup(ctx *gin.Context) {
	h.log.Info(ctx, "UpdatePlayerGroup - 更新玩家权限组")
	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}
	var req apiPlayer.UpdatePlayerGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}
	if xErr := h.service.gameProfileLogic.UpdatePlayerGroup(ctx, playerUUID, "", req.GroupName); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}
	xResult.SuccessHasData(ctx, "更新成功", nil)
}
```

- [ ] **Step 6: 删除旧 handler/player.go**

```bash
rm internal/handler/player.go
```

- [ ] **Step 7: 更新 handler.go — service 结构体**

替换 `playerLogic *logic.PlayerLogic` → `gameProfileLogic *logic.GameProfileLogic`，添加 `userLogic *logic.UserLogic`。

同时更新 `NewHandler` 中的初始化代码：
- `playerLogic: logic.NewPlayerLogic(ctx)` → `gameProfileLogic: logic.NewGameProfileLogic(ctx)`
- 新增 `userLogic: logic.NewUserLogic(ctx)`

- [ ] **Step 8: 更新 title_player.go 中的引用**

替换：
- `h.service.playerLogic` → `h.service.gameProfileLogic`（如果没有直接引用则跳过）
- `h.parsePlayerUUID` 方法保留，内部逻辑不变（该方法仅解析 UUID 参数）

- [ ] **Step 9: 更新 achievement_player.go**

替换：
- `ctx.Param("achId")` → `ctx.Param("ach_id")`
- Swagger 注释中 `@Param achId path` → `@Param ach_id path`
- Swagger 注释中 `:achId` → `:ach_id`
- `h.service.playerLogic` → `h.service.gameProfileLogic`（如果有引用）

- [ ] **Step 10: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 编译通过

- [ ] **Step 11: Commit**

```bash
git rm internal/handler/player.go api/player/request.go api/player/response.go
git add internal/handler/game_profile.go api/game_profile/ handler/handler.go title_player.go achievement_player.go
git commit -m "refactor(handler): PlayerHandler 重命名为 GameProfileHandler，API DTO 移至 game_profile/"
```

---

### Task 9: Route 层重命名

**Files:**
- Delete: `internal/app/route/route_player.go`
- Create: `internal/app/route/route_game_profile.go`
- Modify: `internal/app/route/route.go`
- Modify: `internal/app/route/route_title.go`
- Modify: `internal/app/route/route_achievement.go`

- [ ] **Step 1: 创建 route_game_profile.go**

```go
// internal/app/route/route_game_profile.go
package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) gameProfileRouter(router gin.IRouter) {
	gameProfileHandler := handler.NewGameProfileHandler(r.context)

	profileGroup := router.Group("/game-profiles")
	profileGroup.Use(middleware.LoginAuth(r.context))
	profileGroup.Use(middleware.Player(r.context))
	{
		profileGroup.GET("/:uuid", gameProfileHandler.GetPlayer)
		profileGroup.GET("", gameProfileHandler.ListPlayers)
	}

	internalGroup := router.Group("/internal/game-profiles")
	internalGroup.Use(middleware.LoginAuth(r.context))
	internalGroup.Use(middleware.SuperAdmin(r.context))
	{
		internalGroup.PUT("/:uuid/group", gameProfileHandler.UpdatePlayerGroup)
	}
}
```

- [ ] **Step 2: 删除旧 route_player.go**

```bash
rm internal/app/route/route_player.go
```

- [ ] **Step 3: 更新 route.go**

替换 `r.playerRouter(apiRouter)` → `r.gameProfileRouter(apiRouter)`

- [ ] **Step 4: 更新 route_title.go**

路径前缀替换：
- `router.Group("/players/:uuid/titles")` → `router.Group("/game-profiles/:uuid/titles")`

- [ ] **Step 5: 更新 route_achievement.go**

路径替换：
- `router.Group("/players/:uuid/achievements")` → `router.Group("/game-profiles/:uuid/achievements")`
- `/:achId/claim` → `/:ach_id/claim`

- [ ] **Step 6: 验证编译**

Run: `go build ./internal/app/...`
Expected: 编译失败 — startup_database.go 仍引用旧 entity。下一 Task 修复。

- [ ] **Step 7: Commit**

```bash
git rm internal/app/route/route_player.go
git add internal/app/route/route_game_profile.go internal/app/route/route.go internal/app/route/route_title.go internal/app/route/route_achievement.go
git commit -m "refactor(route): 路径 /players/ 重命名为 /game-profiles/，:achId 改为 :ach_id"
```

---

### Task 10: 更新 startup_database.go + prepare 迁移脚本

**Files:**
- Modify: `internal/app/startup/startup_database.go`
- Modify: `internal/app/startup/prepare/prepare.go`

- [ ] **Step 1: 更新 startup_database.go 的 migrateTables**

```go
var migrateTables = []interface{}{
	&entity.User{},
	&entity.GameProfile{},
	&entity.Title{},
	&entity.Achievement{},
	&entity.GameProfileTitle{},
	&entity.GameProfileAchievement{},
	&entity.GameProfileAchievementClaim{},
}
```

- [ ] **Step 2: 更新 prepare.go — 添加幂等表迁移**

在 `Prepare()` 方法中添加：

```go
func (p *Prepare) Prepare() {
	p.migratePlayerToGameProfile()
}

func (p *Prepare) migratePlayerToGameProfile() {
	// 检查旧表 fp_player 是否存在，存在则重命名
	var count int64
	if err := p.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", "fp_player").Scan(&count).Error; err != nil {
		p.log.Warn(p.ctx, "检查旧表 fp_player 失败: "+err.Error())
		return
	}
	if count == 0 {
		return // 旧表不存在，跳过
	}

	// 检查新表是否已有数据（幂等保护）
	if err := p.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", "fp_game_profile").Scan(&count).Error; err != nil {
		p.log.Warn(p.ctx, "检查新表 fp_game_profile 失败: "+err.Error())
		return
	}
	if count > 0 {
		var rowCount int64
		if err := p.db.Raw("SELECT COUNT(*) FROM fp_game_profile").Scan(&rowCount).Error; err == nil && rowCount > 0 {
			p.log.Info(p.ctx, "fp_game_profile 已有数据，跳过迁移")
			return
		}
	}

	p.log.Info(p.ctx, "正在迁移 fp_player → fp_game_profile...")
	if err := p.db.Exec("ALTER TABLE fp_player RENAME TO fp_game_profile").Error; err != nil {
		p.log.Warn(p.ctx, "重命名表失败: "+err.Error())
		return
	}
	p.log.Info(p.ctx, "表迁移完成: fp_player → fp_game_profile")
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./internal/app/...`
Expected: 编译通过

- [ ] **Step 4: Commit**

```bash
git add internal/app/startup/startup_database.go internal/app/startup/prepare/prepare.go
git commit -m "feat(startup): 更新 migrateTables，新增 Player→GameProfile 表迁移脚本"
```

---

### Task 11: 扩展 gRPC Client — 新增 GetUserInfo

**Files:**
- Modify: `internal/app/grpc/client.go`

- [ ] **Step 1: 在 client.go 新增 GetUserInfo 方法**

在 `Close()` 方法之前添加：

```go
// GetUserInfo 获取用户完整信息（含 GameProfile 列表）
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
```

> 注意：此方法依赖 Yggleaf proto 中已定义 `GetUserInfo` RPC。若 proto 尚未更新，编译会因 `authpb.GetUserInfoRequest`/`GetUserInfoResponse` 不存在而失败。如 Yggleaf 侧未就绪，此方法可临时用 `// +build` tag 或注释保留，等待 proto 更新后再启用。

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/app/grpc/...`
Expected: 若 Yggleaf proto 已更新，编译通过；否则报 undefined，需先在 Yggleaf 侧添加 proto 并重新生成 Go 代码。

- [ ] **Step 3: Commit**

```bash
git add internal/app/grpc/client.go
git commit -m "feat(grpc): AuthClient 新增 GetUserInfo 方法"
```

---

### Task 12: 更新 LoginAuth 中间件 — 同步 User + GameProfile

**Files:**
- Modify: `internal/app/middleware/login_auth.go`

- [ ] **Step 1: 在 ValidateToken 成功后添加 GetUserInfo 同步**

在 `login_auth.go` 中，`ValidateToken` 成功、写入缓存之后、`injectUser` 之前，添加以下代码：

```go
// 异步同步 User + GameProfile 到本地数据库（使用启动 ctx，其中包含 DB/Redis）
go func(startupCtx context.Context) {
	userResp, rpcErr := authClient.GetUserInfo(startupCtx, userInfo.UserID)
	if rpcErr != nil {
		log.Warn(c, "获取用户信息失败: "+rpcErr.Error())
		return
	}
	userLogic := logic.NewUserLogic(startupCtx)
	if err := userLogic.Upsert(c, parseSnowflakeID(userResp.GetUserId()), userResp.GetUsername()); err != nil {
		log.Warn(c, "同步用户失败: "+err.Error())
	}
	gpLogic := logic.NewGameProfileLogic(startupCtx)
	for _, p := range userResp.GetProfiles() {
		gpUUID, uuidErr := uuid.Parse(p.GetUuid())
		if uuidErr != nil {
			continue
		}
		if err := gpLogic.Upsert(c, parseSnowflakeID(userResp.GetUserId()), gpUUID, p.GetName()); err != nil {
			log.Warn(c, "同步 GameProfile 失败: "+err.Error())
		}
	}
}(ctx)
```

需要添加的 import：
```go
import (
	"context"
	"github.com/google/uuid"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
)
```

以及辅助函数：

```go
func parseSnowflakeID(s string) xSnowflake.SnowflakeID {
	id, _ := strconv.ParseInt(s, 10, 64)
	return xSnowflake.SnowflakeID(id)
}
```

需要添加 import：
```go
import (
	"strconv"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
)
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/app/...`
Expected: 编译通过（取决于 gRPC proto 是否就绪）

- [ ] **Step 3: Commit**

```bash
git add internal/app/middleware/login_auth.go
git commit -m "feat(middleware): LoginAuth 新增 User+GameProfile 异步同步"
```

---

### Task 13: 全量编译 + Swagger 验证

**Files:** 无新建，验证所有变更

- [ ] **Step 1: 全量编译**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 2: 生成 Swagger**

Run: `make swag`
Expected: Swagger 生成无报错

- [ ] **Step 3: 开发模式启动**

Run: `make dev`
Expected: 服务启动无报错，数据库迁移成功

- [ ] **Step 4: 验证 API 路径**

用 curl 验证新路径 `/api/v1/game-profiles` 可访问，旧路径 `/api/v1/players` 返回 404。

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: 全量编译通过，验证 Swagger 和启动"
```

---

## 验证清单

1. `go build ./...` 全量编译通过
2. `make swag` Swagger 生成无报错
3. `make dev` 服务正常启动
4. 数据库表 `fp_user`、`fp_game_profile`、`fp_game_profile_title` 等正确创建
5. 旧路径 `/api/v1/players/:uuid` 返回 404
6. 新路径 `/api/v1/game-profiles/:uuid` 正常响应
7. gRPC GetUserInfo 调用返回正确数据（需 Yggleaf 侧 proto 就绪）
