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
	"strconv"
	"strings"
	"sync"
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
	rwmu     *sync.RWMutex
	rotateCh chan struct{}
	checkCh  chan struct{}
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
		rwmu:     &sync.RWMutex{},
		rotateCh: make(chan struct{}, 1),
		checkCh:  make(chan struct{}, 1),
		st:       &Rstat{},
	}

	if err := r.Open(); err != nil {
		debug("[NewRoll] %v", err)
		return nil
	}

	go r.checkAndRoll()

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

	r.fWLock()
	defer r.fWUnlock()

	re, err := r.f.Write(p)
	if err != nil {
		return 0, err
	}

	r.st.update(int64(re))
	go r.checkOnce()
	return re, nil
}

func (r *Roll) Open() error {
	err := r.openFile(r.filePath)
	if err != nil {
		return err
	}
	return r.st.reset(r.filePath)
}

func (r *Roll) openFile(filePath string) error {
	debug("[openFile] %v", filePath)

	var err error
	r.f, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *Roll) Close() error {
	r.fOpLock()
	defer r.fOpUnlock()
	r.rotateCh <- struct{}{}
	debug("[Close]")

	if err := r.closeFile(); err != nil {
		return err
	}
	r.f = nil
	return nil
}

func (r *Roll) closeFile() error {
	debug("[closeFile]")
	return r.f.Close()
}

func (r *Roll) checkOnce() {
	select {
	case r.checkCh <- struct{}{}:
		r.checkAndRoll()
	default:
	}
}

func (r *Roll) checkAndRoll() {
	for range r.checkCh {
		rolling, err := r.checkChain()
		if err != nil {
			debug("[checkAndRoll] [check] err: %v", err)
		}

		if rolling {
			if err = r.roll(); err != nil {
				debug("[checkAndRoll] [check] err: %v", err)
			}
		}
	}
}

func (r *Roll) roll() error {
	r.fOpLock()
	defer r.fOpUnlock()

	if r.f == nil {
		return nil
	}
	if err := r.openNew(); err != nil {
		return err
	}

	go r.process()
	return nil
}

func (r *Roll) checkChain() (bool, error) {
	r.fWLock()
	defer r.fWUnlock()
	for _, checker := range r.checkers {
		rolling, err := checker.Check(r.filePath, r.st)
		if err != nil {
			return false, err
		}
		if rolling {
			debug("[%s] hint %d", checker.Name(), r.st.Size())
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

func (r *Roll) openNew() error {
	err := r.st.reset(r.filePath)
	if err != nil {
		return err
	}

	if err = r.closeFile(); err != nil {
		debug("[closeFile] err: %v", err)
		return err
	}

	return r.openFile(r.tmpFilePath)
}

func (r *Roll) process() {
	select {
	case r.rotateCh <- struct{}{}:
		r.rollOnce()
	default:
	}
}

func (r *Roll) rollOnce() error {
	debug("[rollingOnce]")
	defer func() {
		<-r.rotateCh
	}()

	dir := path.Dir(r.filePath)
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
		if e.Type().IsRegular() && r.matcher.Match(e.Name()) {
			files = append(files, e)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		f1 := files[i].Name()
		f2 := files[j].Name()
		if len(f2) != len(f1) {
			return len(f2) > len(f1)
		}

		idx1 := strings.LastIndexByte(f1, '.')
		if idx1 == -1 {
			idx1 = 0
		}
		idx2 := strings.LastIndexByte(f2, '.')
		if idx2 == -1 {
			idx2 = 0
		}

		n1, _ := strconv.Atoi(f1[idx1+1:])
		n2, _ := strconv.Atoi(f2[idx2+1:])

		return n1 < n2
	})

	debugArray(files, func(idx int) string {
		return files[idx].Name()
	}, "[sorted]")

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

// lock for writing file, exlusive for close and open
func (r *Roll) fWLock() {
	if r.rwmu != nil {
		r.rwmu.RLock()
	}
}

func (r *Roll) fWUnlock() {
	if r.rwmu != nil {
		r.rwmu.RUnlock()
	}
}

// lock for operating file, exlusive all file operation
func (r *Roll) fOpLock() {
	if r.rwmu != nil {
		r.rwmu.Lock()
	}
}

func (r *Roll) fOpUnlock() {
	if r.rwmu != nil {
		r.rwmu.Unlock()
	}
}
