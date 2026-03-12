package query

import (
	"github.com/yaleh/meta-cc/internal/analyzer"
	"github.com/yaleh/meta-cc/internal/types"
)

type FileChurnOptions struct {
	Threshold int
}

func DetectFileChurn(entries []types.SessionEntry, opts FileChurnOptions) []analyzer.FileChurnDetail {
	result := analyzer.DetectFileChurn(entries, opts.Threshold)
	return result.HighChurnFiles
}
