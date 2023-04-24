package rollinguf

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

type Compare interface {
	int
}

func min[T Compare](a, b T) T {
	if a < b {
		return a
	}
	return b
}

var debugEnabled bool

func SetDebug(enabled bool) {
	debugEnabled = enabled
}

func debug(format string, args ...any) {
	if !debugEnabled {
		return
	}
	_, f, l, _ := runtime.Caller(1)
	log.Printf(fmt.Sprintf("%s:%d [rollwf] ", f, l)+format+"\n", args...)
}

func debugArray(arr any, formator func(idx int) string, format string, args ...any) {
	if !debugEnabled {
		return
	}
	_, f, l, _ := runtime.Caller(1)
	for i := range arr.([]os.DirEntry) {
		log.Printf(fmt.Sprintf("%s:%d [rollwf] ", f, l) + fmt.Sprintf(format, args...) + fmt.Sprintf(" %s\n", formator(i)))
	}
}
