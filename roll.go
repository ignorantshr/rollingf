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

package rollingf

import (
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"sync"
	"time"
)

var _ io.WriteCloser = (*Roll)(nil)

type Roll struct {
	filePath    string
	tmpFilePath string

	checkers  []Checker
	filters   []Filter
	matcher   Matcher
	processor Processor

	f        *os.File
	st       *Rstat
	mu       *sync.Mutex
	rotateCh chan struct{}
}

// NewC creates a customizable Roll
//
// The following components need to be populated:
//   - Checker
//   - Mather
//   - Filter
//   - Processor
func NewC(filePath string, opts ...Option) *Roll {
	r := baseR(filePath)
	if r == nil {
		return nil
	}

	return r.WithOptions(opts...)
}

// New roll creates a Roll with default components
func New(c RollConf, opts ...Option) *Roll {
	r := baseR(c.FilePath)
	if r == nil {
		return nil
	}

	r = r.WithDefaultChecker(c.RollCheckerConf)
	r = r.WithDefaultFilter(c.RollFilterConf)
	r = r.WithDefaultMatcher()
	r = r.WithDefaultProcessor()

	return r.WithOptions(opts...)
}

func baseR(filePath string) *Roll {
	r := &Roll{
		filePath: filePath,
		mu:       &sync.Mutex{},
		rotateCh: make(chan struct{}),
	}

	if err := r.Open(); err != nil {
		debug("[NewRoll] %v", err)
		return nil
	}

	dir, base := path.Split(filePath)
	r.tmpFilePath = dir + "_" + base

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

func (r *Roll) WithDefaultMatcher() *Roll {
	r.WithMatcher(DefaultMatcher())
	return r
}

func (r *Roll) WithDefaultProcessor() *Roll {
	r.WithProcessor(DefaultProcessor())
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

func (r *Roll) WithMatcher(m Matcher) *Roll {
	m.Init(path.Base(r.filePath))
	r.matcher = m
	return r
}

func (r *Roll) WithProcessor(p Processor) *Roll {
	r.processor = p
	return r
}

func (r *Roll) WithOptions(opts ...Option) *Roll {
	for _, opt := range opts {
		opt.apply(r)
	}
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
	// r.Lock()
	// defer r.Unlock()

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
	return r.openFile(r.filePath)
}

func (r *Roll) openFile(filePath string) error {
	debug("[openFile]")

	var err error
	r.f, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	rs := &Rstat{}
	err = rs.reset(filePath)
	if err != nil {
		return err
	}
	r.st = rs

	return nil
}

func (r *Roll) Close() error {
	debug("[Close]")
	r.Lock()
	defer r.Unlock()

	return r.closeFile()
}

func (r *Roll) closeFile() error {
	debug("[closeFile]")
	return r.f.Close()
}

func (r *Roll) roll() error {
	err := func() error {
		r.Lock()
		defer r.Unlock()
		return r.openNew()
	}()
	if err != nil {
		return err
	}

	go r.process()
	r.rotateCh <- struct{}{}
	return nil
}

func (r *Roll) checkChain() (bool, error) {
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

func (r *Roll) filterChain(files []os.DirEntry) ([]os.DirEntry, error) {
	var remains = files
	for _, f := range r.filters {
		items, tmp, err := f.Filter(remains)
		if err != nil {
			return nil, err
		}
		if len(tmp) > 0 {
			debugArray(tmp, func(idx int) string {
				return tmp[idx].Name()
			}, "[%s]", f.Name())
			f.DealFiltered(path.Dir(r.filePath), tmp)
		}
		remains = items
	}

	return remains, nil
}

func (r *Roll) Lock() {
	if r.mu != nil {
		r.mu.Lock()
	}
}

func (r *Roll) Unlock() {
	if r.mu != nil {
		r.mu.Unlock()
	}
}

func (r *Roll) openNew() error {
	if err := r.closeFile(); err != nil {
		debug("[closeFile] err: %v", err)
		return err
	}

	return r.openFile(r.tmpFilePath)
}

func (r *Roll) process() {
	select {
	case <-r.rotateCh:
		r.processOnce()
	case <-time.After(time.Second * 5):
		return
	}
}

func (r *Roll) processOnce() error {
	debug("[rprocessOnce]")
	r.Lock()
	defer r.Unlock()

	dir := path.Dir(r.filePath)
	base := path.Base(r.filePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// match
	if r.matcher == nil {
		return nil
	}
	var files []fs.DirEntry
	for _, e := range entries {
		if e.Type().IsRegular() && r.matcher.Match(base, e.Name()) {
			files = append(files, e)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// filter
	remains, err := r.filterChain(files)
	if err != nil {
		return err
	}

	debugArray(remains, func(idx int) string {
		return remains[idx].Name()
	}, "[remain]")

	// processor
	if r.processor == nil {
		return nil
	}
	debug("[processor]")
	if err := r.processor.Process(dir, remains); err != nil {
		return err
	}

	return os.Rename(r.tmpFilePath, r.filePath)
}
