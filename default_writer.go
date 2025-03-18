package writer

// Default Writer to easy usage

import (
	"os"
	"runtime"
)

// Writer Component for the Logger Module
// ----------------------------------------------------

var (
	emptyFileSlice = make([]*os.File, 0)
	appendModeA    = "a"
	modeA          = Mode{mode: &appendModeA}
	maxPool        = uint64(runtime.NumCPU() * 4)
	message        = "Add text to write"
	// DefaultWriter is the default Writer instance.
	Dwriter = NewWriter(
		&emptyFileSlice, &modeA, &message, maxPool, 2, 100,
	)
)
