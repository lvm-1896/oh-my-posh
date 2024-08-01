package shell

import (
	_ "embed"
)

//go:embed scripts/omp.yash
var yashInit string

func (f Feature) Yash() Code {
	switch f {
	case CursorPositioning:
		return "_omp_cursor_positioning=1"
	case Upgrade:
		return unixUpgrade
	case Notice:
		return unixNotice
	case PromptMark, RPrompt, PoshGit, Azure, LineError, Jobs, Tooltips, Transient:
		fallthrough
	default:
		return ""
	}
}
