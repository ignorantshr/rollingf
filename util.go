// Copyright 2023 ignorantshr.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rollinguf

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

type Compare interface {
	int | int64 | float64
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
