package apiMatrixStatistic

import (
	"encoding/json"
	"time"
)

type StatisticResponse struct {
	ID                  string          `json:"id"`
	PlayerUUID          string          `json:"player_uuid"`
	PlayerName          string          `json:"player_name"`
	BlocksBreak         json.RawMessage `json:"blocks_break"`
	BlocksPlace         json.RawMessage `json:"blocks_place"`
	EntitiesKill        json.RawMessage `json:"entities_kill"`
	Deaths              json.RawMessage `json:"deaths"`
	ItemsUsed           json.RawMessage `json:"items_used"`
	TotalBlocksBroken   int64           `json:"total_blocks_broken"`
	TotalBlocksPlaced   int64           `json:"total_blocks_placed"`
	TotalEntitiesKilled int64           `json:"total_entities_killed"`
	TotalDeaths         int64           `json:"total_deaths"`
	TotalPlayTimeMs     int64           `json:"total_play_time_ms"`
	CurrentSessionStart time.Time       `json:"current_session_start"`
	TotalSessions       int64           `json:"total_sessions"`
}