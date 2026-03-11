package types_test

import "github.com/yaleh/meta-cc/internal/types"

// Compile-time assertion: TimeRange exists with correct string fields
var _ types.TimeRange = types.TimeRange{Start: "", End: ""}
