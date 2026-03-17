package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// MemOSConfig 存储 MemOS 连接配置，以 JSON 形式持久化到单个数据库列。
type MemOSConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	TopK    int    `json:"top_k,omitempty"`
	Async   bool   `json:"async,omitempty"`
}

func (c MemOSConfig) EffectiveTopK() int {
	if c.TopK > 0 {
		return c.TopK
	}
	return 10
}

func (c MemOSConfig) EffectiveUserID() string {
	if c.UserID != "" {
		return c.UserID
	}
	return "goclaw-user"
}

func (c MemOSConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *MemOSConfig) Scan(src any) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for MemOSConfig: %T", src)
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, c)
}

type Agent struct {
	ID            int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	UUID          string      `json:"uuid" gorm:"uniqueIndex;size:36;not null"`
	Name          string      `json:"name" gorm:"size:200;not null"`
	Description   string      `json:"description" gorm:"type:text"`
	SystemPrompt  string      `json:"system_prompt" gorm:"type:text"`
	ProviderID    int64       `json:"provider_id" gorm:"index;not null"`
	ModelName     string      `json:"model_name" gorm:"size:100;not null"`
	Temperature   float64     `json:"temperature" gorm:"default:0.7"`
	MaxTokens     int         `json:"max_tokens" gorm:"default:4096"`
	Timeout       int         `json:"timeout" gorm:"default:120"`
	MaxHistory    int         `json:"max_history" gorm:"default:50"`
	MaxIterations int         `json:"max_iterations" gorm:"default:10"`
	Token         string      `json:"token" gorm:"uniqueIndex;size:64"`
	ToolSearchEnabled bool        `json:"tool_search_enabled" gorm:"column:tool_search_enabled;default:false"`
	MemOSEnabled      bool        `json:"memos_enabled" gorm:"column:memos_enabled;default:false"`
	MemOSCfg          MemOSConfig `json:"memos_config" gorm:"column:memos_config;type:text"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`

	Tools      []Tool      `json:"tools,omitzero" gorm:"-"`
	Skills     []Skill     `json:"skills,omitzero" gorm:"-"`
	MCPServers []MCPServer `json:"mcp_servers,omitzero" gorm:"-"`
}

const (
	DefaultAgentMaxHistory    = 50
	DefaultAgentMaxIterations = 10
)

func (a *Agent) TimeoutSeconds() int {
	return a.Timeout
}

func (a *Agent) HistoryLimit() int {
	if a.MaxHistory > 0 {
		return a.MaxHistory
	}
	return DefaultAgentMaxHistory
}

func (a *Agent) IterationLimit() int {
	if a.MaxIterations > 0 {
		return a.MaxIterations
	}
	return DefaultAgentMaxIterations
}

type CreateAgentReq struct {
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	SystemPrompt  string      `json:"system_prompt"`
	ProviderID    int64       `json:"provider_id"`
	ModelName     string      `json:"model_name"`
	Temperature   float64     `json:"temperature"`
	MaxTokens     int         `json:"max_tokens"`
	Timeout       int         `json:"timeout"`
	MaxHistory    int         `json:"max_history"`
	MaxIterations int         `json:"max_iterations"`
	ToolSearchEnabled bool        `json:"tool_search_enabled"`
	MemOSEnabled      bool        `json:"memos_enabled"`
	MemOSCfg          MemOSConfig `json:"memos_config"`
	ToolIDs           []int64     `json:"tool_ids,omitzero"`
	SkillIDs      []int64     `json:"skill_ids,omitzero"`
	MCPServerIDs  []int64     `json:"mcp_server_ids,omitzero"`
}

type UpdateAgentReq struct {
	Name          *string      `json:"name,omitzero"`
	Description   *string      `json:"description,omitzero"`
	SystemPrompt  *string      `json:"system_prompt,omitzero"`
	ProviderID    *int64       `json:"provider_id,omitzero"`
	ModelName     *string      `json:"model_name,omitzero"`
	Temperature   *float64     `json:"temperature,omitzero"`
	MaxTokens     *int         `json:"max_tokens,omitzero"`
	Timeout       *int         `json:"timeout,omitzero"`
	MaxHistory    *int         `json:"max_history,omitzero"`
	MaxIterations *int         `json:"max_iterations,omitzero"`
	ToolSearchEnabled *bool        `json:"tool_search_enabled,omitzero"`
	MemOSEnabled      *bool        `json:"memos_enabled,omitzero"`
	MemOSCfg          *MemOSConfig `json:"memos_config,omitzero"`
	ToolIDs           []int64      `json:"tool_ids,omitzero"`
	SkillIDs      []int64      `json:"skill_ids,omitzero"`
	MCPServerIDs  []int64      `json:"mcp_server_ids,omitzero"`
}
