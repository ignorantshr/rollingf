package rollingf

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

// type Compare interface {
// 	int
// }

// func min[T Compare](a, b T) T {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type any = interface{}

var debugEnabled bool

func SetDebug(enabled bool) {
	debugEnabled = enabled
}

func debug(format string, args ...any) {
	if !debugEnabled {
		return
	}
	_, f, l, _ := runtime.Caller(1)
	log.Printf(fmt.Sprintf("%s:%d [rollingf] ", f, l)+format+"\n", args...)
}

func debugArray(arr any, formator func(idx int) string, format string, args ...any) {
	if !debugEnabled {
		return
	}
	_, f, l, _ := runtime.Caller(1)
	pre := fmt.Sprintf("%s:%d [rollingf] ", f, l) + fmt.Sprintf(format, args...)
	for i := range arr.([]os.DirEntry) {
		pre += fmt.Sprintf(" %s", formator(i))
	}
	log.Println(pre)
}
