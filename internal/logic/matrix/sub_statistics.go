package matrix

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
)

const flushInterval = 10

// StatisticsSub 统计聚合 Sub，聚合方块/实体/死亡统计到 DB
type StatisticsSub struct {
	mu         sync.Mutex
	statRepo   *repository.MatrixStatisticRepo
	playerUUID uuid.UUID
	playerName string
	serverName string
	batchCount int

	blocksBreak  map[string]int64
	blocksPlace  map[string]int64
	entitiesKill map[string]int64
	deaths       map[string]int64

	totalBlocksBroken   int64
	totalBlocksPlaced   int64
	totalEntitiesKilled int64
	totalDeaths         int64
}

// NewStatisticsSub 创建 StatisticsSub 实例
func NewStatisticsSub(playerUUID uuid.UUID, playerName, serverName string, statRepo *repository.MatrixStatisticRepo) *StatisticsSub {
	return &StatisticsSub{
		statRepo:     statRepo,
		playerUUID:   playerUUID,
		playerName:   playerName,
		serverName:   serverName,
		blocksBreak:  make(map[string]int64),
		blocksPlace:  make(map[string]int64),
		entitiesKill: make(map[string]int64),
		deaths:       make(map[string]int64),
	}
}

// Name 返回 sub 名称
func (s *StatisticsSub) Name() string {
	return "statistics"
}

// Process 处理单条遥测数据
func (s *StatisticsSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, evt := range msg.GetBlockBreaks() {
		material := evt.GetMaterial()
		s.blocksBreak[material]++
		s.totalBlocksBroken++
	}
	for _, evt := range msg.GetBlockPlaces() {
		material := evt.GetMaterial()
		s.blocksPlace[material]++
		s.totalBlocksPlaced++
	}
	for _, evt := range msg.GetEntityKills() {
		entityType := evt.GetEntityType()
		s.entitiesKill[entityType]++
		s.totalEntitiesKilled++
	}
	for _, evt := range msg.GetPlayerDeaths() {
		cause := evt.GetDeathCause()
		s.deaths[cause]++
		s.totalDeaths++
	}

	if msg.GetPlayerQuit() != nil {
		s.flushToDB(ctx)
		return nil
	}

	s.batchCount++
	if s.batchCount >= flushInterval {
		s.flushToDB(ctx)
	}

	return nil
}

// Drain 排水时强制刷盘到 DB
func (s *StatisticsSub) Drain(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushToDB(ctx)
	return nil
}

// flushToDB 将累积统计写入数据库
func (s *StatisticsSub) flushToDB(ctx context.Context) {
	if len(s.blocksBreak) == 0 && len(s.blocksPlace) == 0 &&
		len(s.entitiesKill) == 0 && len(s.deaths) == 0 {
		return
	}

	blocksBreakJSON, _ := json.Marshal(s.blocksBreak)
	blocksPlaceJSON, _ := json.Marshal(s.blocksPlace)
	entitiesKillJSON, _ := json.Marshal(s.entitiesKill)
	deathsJSON, _ := json.Marshal(s.deaths)

	stat := &entity.MatrixPlayerStatistic{
		PlayerUUID:         s.playerUUID,
		PlayerName:         s.playerName,
		BlocksBreak:        blocksBreakJSON,
		BlocksPlace:        blocksPlaceJSON,
		EntitiesKill:       entitiesKillJSON,
		Deaths:             deathsJSON,
		ItemsUsed:          json.RawMessage(`{}`),
		TotalBlocksBroken:  s.totalBlocksBroken,
		TotalBlocksPlaced:  s.totalBlocksPlaced,
		TotalEntitiesKilled: s.totalEntitiesKilled,
		TotalDeaths:        s.totalDeaths,
		CurrentSessionStart: time.Now(),
	}

	if xErr := s.statRepo.Upsert(ctx, stat); xErr != nil {
		// Error logged by repo, continue processing
		_ = xErr
	}

	// Reset batch counter but keep cumulative totals
	s.batchCount = 0
	s.blocksBreak = make(map[string]int64)
	s.blocksPlace = make(map[string]int64)
	s.entitiesKill = make(map[string]int64)
	s.deaths = make(map[string]int64)
}
