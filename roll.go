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
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"sync"
)

var _ io.WriteCloser = (*Roll)(nil)

type Roll struct {
	filePath string

	checkers  []Checker
	filters   []Filter
	processor Processor

	f       *os.File
	st      *Rstat
	lock    *sync.Mutex
	running bool
}

// NewRoll creates a customizable Roll
//
// The following components need to be populated:
//   - Checker
//   - Filter
//   - Processor
func NewRoll(filePath string) *Roll {
	r := &Roll{
		filePath: filePath,
		lock:     &sync.Mutex{},
		running:  true,
	}

	if err := r.Open(); err != nil {
		debug("[NewRoll] %v", err)
		return nil
	}

	return r
}

// New roll creates a Roll with default components
func New(c RollConf) *Roll {
	r := NewRoll(c.FilePath)
	if r == nil {
		return nil
	}

	r = r.WithDefaultChecker(c.RollCheckerConf)
	r = r.WithDefaultFilter(c.RollFilterConf)

	r.processor = NewDefaultProcessor()
	return r
}

func (r *Roll) WithDefaultChecker(c RollCheckerConf) *Roll {
	r.WithChecker(DefaultChecker(c)...)
	return r
}

func (r *Roll) WithDefaultFilter(c RollFilterConf) *Roll {
	r.WithFilter(DefaultFilter(c)...)
	return r
}

func (r *Roll) WithDefaultProcessor() *Roll {
	r.WithProcessor(NewDefaultProcessor())
	return r
}

func (r *Roll) WithChecker(c ...Checker) *Roll {
	r.checkers = append(r.checkers, c...)
	return r
}

func (r *Roll) WithFilter(f ...Filter) *Roll {
	r.filters = append(r.filters, f...)
	return r
}

func (r *Roll) WithProcessor(m Processor) *Roll {
	r.processor = m
	return r
}

// Write writes the given bytes to the file.
//
//  1. The rolling will be executed when trigger a Checker.
//  2. Then the file will be filter out remove files by Filters.
//  3. Then the filtered files will be removed.
//  4. Finally the remains will be rolled.
func (r *Roll) Write(p []byte) (n int, err error) {
	debug("[Write]")
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.Running() {
		return 0, nil
	}

	// check
	rolling, err := r.checkChain()
	if err != nil {
		return 0, err
	}

	if rolling {
		if err = r.roll(); err != nil {
			return 0, err
		}
	}

	re, err := r.f.Write(p)
	if err != nil {
		return 0, err
	}

	r.st.update(int64(re))
	return re, nil
}

func (r *Roll) Open() error {
	debug("[Open]")
	if !r.Running() {
		return errors.New("rollinguf is not running")
	}

	var err error
	r.f, err = os.OpenFile(r.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	rs := &Rstat{}
	err = rs.reset(r.filePath)
	if err != nil {
		return err
	}
	r.st = rs

	return nil
}

func (r *Roll) Close() error {
	debug("[Close]")
	r.lock.Lock()
	defer r.lock.Unlock()

	r.running = false
	return r.closeFile()
}

func (r *Roll) closeFile() error {
	debug("[CloseFile]")
	return r.f.Close()
}

func (r *Roll) roll() error {
	dir := path.Dir(r.filePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// match
	var files []fs.DirEntry
	for _, e := range entries {
		if e.Type().IsRegular() && r.processor.Match(e.Name()) {
			files = append(files, e)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// filter
	remains, removes, err := r.filterChain(files)
	if err != nil {
		return err
	}

	// remove
	for _, f := range removes {
		debug("[remove] %v", f.Name())
		if err := os.Remove(path.Join(dir, f.Name())); err != nil {
			return err
		}
	}

	debugArray(remains, func(idx int) string {
		return remains[idx].Name()
	}, "[remain]")

	if !r.Running() {
		return nil
	}

	if err := r.closeFile(); err != nil {
		return err
	}

	// process
	debug("[process]")
	if err := r.processor.Process(dir, remains); err != nil {
		return err
	}

	if err := r.Open(); err != nil {
		return err
	}

	return nil
}

func (r *Roll) checkChain() (bool, error) {
	debug("[rstat] %s", r.st)
	for _, checker := range r.checkers {
		debug("[%s]", checker.Name())
		rolling, err := checker.Check(r.filePath, r.st)
		if err != nil {
			return false, err
		}
		if rolling {
			debug("[%s] hint", checker.Name())
			return true, nil
		}
	}

	return false, nil
}

func (r *Roll) filterChain(files []os.DirEntry) ([]os.DirEntry, []os.DirEntry, error) {
	var remains = files
	var removed []os.DirEntry
	for _, f := range r.filters {
		items, tmp, err := f.Filter(remains)
		if err != nil {
			return nil, nil, err
		}
		if len(tmp) > 0 {
			debugArray(tmp, func(idx int) string {
				return tmp[idx].Name()
			}, "[%s]", f.Name())
			removed = append(removed, tmp...)
		}
		remains = items
	}

	return remains, removed, nil
}

func (r *Roll) Running() bool {
	return r.running
}
