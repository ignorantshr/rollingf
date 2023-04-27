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
	"fmt"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const tsFormat = "2006-01-02 15:04:05"

// Rstat wraps os.FileInfo with local stored information about the file.
type Rstat struct {
	info fs.FileInfo

	rSize         int64
	modeTime      time.Time
	birthTimespec *syscall.Timespec

	mu sync.RWMutex // todo test
}

// FileInfo returns the underlying fs.FileInfo.
func (r *Rstat) FileInfo() fs.FileInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.info
}

func (r *Rstat) Name() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.info.Name()
}

func (r *Rstat) Size() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return atomic.LoadInt64(&r.rSize)
}

func (r *Rstat) Mode() fs.FileMode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.info.Mode()
}

func (r *Rstat) ModTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.modeTime
}

func (r *Rstat) IsDir() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.info.IsDir()
}

// Birthtimespec returns the file's birth time.
func (r *Rstat) Birthtimespec() (bool, syscall.Timespec) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.birthTimespec == nil {
		return false, syscall.Timespec{}
	}

	return true, *r.birthTimespec
}

func (r *Rstat) String() string {
	return fmt.Sprintf("%s, rsize: %d bytes, modeTime: %v, birthTimespec: %v",
		r.info.Name(), r.rSize, r.modeTime.Format(tsFormat), time.Unix(r.birthTimespec.Sec, 0).Format(tsFormat))
}

func (r *Rstat) reset(filePath string) error {
	r.Lock()
	defer r.Unlock()

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	r.info = info
	r.rSize = info.Size()
	r.modeTime = info.ModTime()

	stat, ok := r.info.Sys().(*syscall.Stat_t)
	if ok {
		r.birthTimespec = &stat.Birthtimespec
	}

	debug("[reset]")
	return nil
}

func (r *Rstat) update(size int64) {
	r.Lock()
	defer r.Unlock()

	atomic.AddInt64(&r.rSize, size)
	r.modeTime = time.Now()
}

func (r *Rstat) clone() *Rstat {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c := &r
	return *c
}

// Lock
func (r *Rstat) Lock() {
	r.mu.Lock()
}

func (r *Rstat) RLock() {
	r.mu.RLock()

}

// Unlock
func (r *Rstat) Unlock() {
	r.mu.Unlock()
}

func (r *Rstat) RUnlock() {
	r.mu.RUnlock()
}
