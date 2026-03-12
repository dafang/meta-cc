package analyzer

import (
	"github.com/yaleh/meta-cc/internal/types"
)

// SequenceAnalysis represents tool sequence analysis results
type SequenceAnalysis struct {
	Sequences []types.SequencePattern `json:"sequences"`
}

// FileChurnAnalysis represents file churn analysis results
type FileChurnAnalysis struct {
	HighChurnFiles []FileChurnDetail `json:"high_churn_files"`
}

// FileChurnDetail represents detailed file access statistics
type FileChurnDetail struct {
	File          string `json:"file"`
	ReadCount     int    `json:"read_count"`
	EditCount     int    `json:"edit_count"`
	WriteCount    int    `json:"write_count"`
	TotalAccesses int    `json:"total_accesses"`
	TimeSpanMin   int    `json:"time_span_minutes"`
	FirstAccess   int64  `json:"first_access"`
	LastAccess    int64  `json:"last_access"`
}

// IdlePeriodAnalysis represents idle period analysis results
type IdlePeriodAnalysis struct {
	IdlePeriods []IdlePeriod `json:"idle_periods"`
}

// IdlePeriod represents a detected idle period
type IdlePeriod struct {
	StartTurn      int          `json:"start_turn"`
	EndTurn        int          `json:"end_turn"`
	DurationMin    float64      `json:"duration_minutes"`
	StartTimestamp int64        `json:"start_timestamp"`
	EndTimestamp   int64        `json:"end_timestamp"`
	ContextBefore  *TurnContext `json:"context_before,omitempty"`
	ContextAfter   *TurnContext `json:"context_after,omitempty"`
}

// TurnContext represents context around an event
type TurnContext struct {
	Turn    int    `json:"turn"`
	Role    string `json:"role,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Status  string `json:"status,omitempty"`
	Preview string `json:"preview,omitempty"`
}
