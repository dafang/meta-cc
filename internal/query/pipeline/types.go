package pipeline

import "github.com/yaleh/meta-cc/internal/types"

// QueryParams represents unified query parameters.
type QueryParams struct {
	Resource  string        `json:"resource"` // "entries" | "messages" | "tools"
	Scope     string        `json:"scope"`    // "session" | "project"
	Filter    FilterSpec    `json:"filter"`
	Transform TransformSpec `json:"transform"`
	Aggregate AggregateSpec `json:"aggregate"`
	Output    OutputSpec    `json:"output"`
	JQFilter  string        `json:"jq_filter,omitempty"`
}

// FilterSpec represents structured filter conditions.
type FilterSpec struct {
	Type         string     `json:"type,omitempty"`
	SessionID    string     `json:"session_id,omitempty"`
	UUID         string     `json:"uuid,omitempty"`
	ParentUUID   string     `json:"parent_uuid,omitempty"`
	GitBranch    string     `json:"git_branch,omitempty"`
	TimeRange    *TimeRange `json:"time_range,omitempty"`
	Role         string     `json:"role,omitempty"`
	ContentType  string     `json:"content_type,omitempty"`
	ContentMatch string     `json:"content_match,omitempty"`
	ToolName     string     `json:"tool_name,omitempty"`
	ToolStatus   string     `json:"tool_status,omitempty"`
	HasError     *bool      `json:"has_error,omitempty"`
}

// TimeRange is an alias for types.TimeRange.
type TimeRange = types.TimeRange

// TransformSpec represents transformation operations.
type TransformSpec struct {
	Extract []string  `json:"extract,omitempty"`
	GroupBy string    `json:"group_by,omitempty"`
	Join    *JoinSpec `json:"join,omitempty"`
}

// JoinSpec represents a join operation.
type JoinSpec struct {
	Type string `json:"type"`
	On   string `json:"on"`
}

// AggregateSpec represents aggregation operations.
type AggregateSpec struct {
	Function string `json:"function,omitempty"`
	Field    string `json:"field,omitempty"`
}

// OutputSpec represents output control options.
type OutputSpec struct {
	Format    string `json:"format,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// IsEmpty returns true if the filter has no conditions.
func (f FilterSpec) IsEmpty() bool {
	return f.Type == "" &&
		f.SessionID == "" &&
		f.UUID == "" &&
		f.ParentUUID == "" &&
		f.GitBranch == "" &&
		f.TimeRange == nil &&
		f.Role == "" &&
		f.ContentType == "" &&
		f.ContentMatch == "" &&
		f.ToolName == "" &&
		f.ToolStatus == "" &&
		f.HasError == nil
}

// IsEmpty returns true if the aggregate has no operations.
func (a AggregateSpec) IsEmpty() bool {
	return a.Function == ""
}

var ValidResourceTypes = []string{"entries", "messages", "tools"}
var ValidScopes = []string{"session", "project"}
var ValidAggregateFunctions = []string{"count", "sum", "avg", "min", "max", "group"}
var ValidOutputFormats = []string{"jsonl", "tsv", "summary"}

// ValidateQueryParams validates query parameters.
func ValidateQueryParams(params QueryParams) error {
	params = ApplyDefaults(params)

	if !isValidValue(params.Resource, ValidResourceTypes) {
		return &ValidationError{Field: "resource", Value: params.Resource, ValidValues: ValidResourceTypes}
	}
	if !isValidValue(params.Scope, ValidScopes) {
		return &ValidationError{Field: "scope", Value: params.Scope, ValidValues: ValidScopes}
	}
	if !params.Aggregate.IsEmpty() {
		if !isValidValue(params.Aggregate.Function, ValidAggregateFunctions) {
			return &ValidationError{Field: "aggregate.function", Value: params.Aggregate.Function, ValidValues: ValidAggregateFunctions}
		}
	}
	if params.Output.Format != "" {
		if !isValidValue(params.Output.Format, ValidOutputFormats) {
			return &ValidationError{Field: "output.format", Value: params.Output.Format, ValidValues: ValidOutputFormats}
		}
	}
	return nil
}

// ApplyDefaults applies default values to query parameters.
func ApplyDefaults(params QueryParams) QueryParams {
	if params.Resource == "" {
		params.Resource = "entries"
	}
	if params.Scope == "" {
		params.Scope = "project"
	}
	if params.Output.Format == "" {
		params.Output.Format = "jsonl"
	}
	return params
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field       string
	Value       string
	ValidValues []string
}

func (e *ValidationError) Error() string {
	if len(e.ValidValues) > 0 {
		return "invalid " + e.Field + ": \"" + e.Value + "\", valid values: " + joinStrings(e.ValidValues, ", ")
	}
	return "invalid " + e.Field + ": \"" + e.Value + "\""
}

func isValidValue(value string, validValues []string) bool {
	for _, v := range validValues {
		if value == v {
			return true
		}
	}
	return false
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
