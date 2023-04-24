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
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Processor mathes the files and processes them
type Processor interface {

	// Match return true if the file name matches the base
	Match(base string) bool

	// Process process the filtered files.
	Process(dir string, remains []os.DirEntry) error
}

type DefaultProcessor struct {
	reg  *regexp.Regexp
	once sync.Once
}

func NewDefaultProcessor() *DefaultProcessor {
	return &DefaultProcessor{}
}

func (p *DefaultProcessor) Match(base string) bool {
	return len(p.regexp(base).Find([]byte(path.Base(base)))) == len(base)
}

func (m *DefaultProcessor) regexp(file string) *regexp.Regexp {
	m.once.Do(func() {
		m.reg = regexp.MustCompile(strings.ReplaceAll(path.Base(file), ".", `\.`) + `\.?\d*`)
	})
	return m.reg
}

func (m *DefaultProcessor) Process(dir string, remains []os.DirEntry) error {
	if len(remains) > 0 {
		for i := len(remains) - 1; i >= 0; i-- {
			if err := m.each(dir, remains[i].Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *DefaultProcessor) each(dir, base string) error {
	newName := m.incrTailNumber(base)

	debug("[Rename] %v --> %v", base, newName)
	if err := os.Rename(path.Join(dir, base), path.Join(dir, newName)); err != nil {
		return err
	}
	return nil
}

// incrTailNumber increase the tail number of the file name.
//
// eg.
//
//	base: "abc.log",
//	return: "abc.log.1"
func (m *DefaultProcessor) incrTailNumber(base string) string {
	if len(base) == 0 {
		return base
	}

	idx := strings.LastIndexByte(base, '.')
	if idx == -1 {
		idx = 0
	}

	tail := 1
	pre := base[:idx]
	last := base[idx+1:]
	if IsNumeric(last) {
		tail, _ = strconv.Atoi(last)
		tail++
	} else {
		pre = base
	}
	return pre + "." + strconv.Itoa(tail)
}