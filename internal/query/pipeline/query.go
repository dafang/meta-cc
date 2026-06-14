package pipeline

import (
	"fmt"

	"github.com/yaleh/meta-cc/internal/types"
)

// Query executes a unified query on session entries.
func Query(entries []types.SessionEntry, params QueryParams) (interface{}, error) {
	params = ApplyDefaults(params)
	if err := ValidateQueryParams(params); err != nil {
		return nil, fmt.Errorf("invalid query parameters: %w", err)
	}

	resources, err := SelectResource(entries, params.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to select resource: %w", err)
	}

	filtered := ApplyFilter(resources, params.Filter)
	aggregated := ApplyAggregate(filtered, params.Aggregate)

	return aggregated, nil
}
