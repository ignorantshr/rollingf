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
	"io/fs"
	"syscall"
	"time"
)

// Checker is a file checker, it checks if a file is shall be rolled.
type Checker interface {
	Name() string
	Check(filePath string, stat fs.FileInfo) (bool, error)
}

var (
	_ Checker = (*IntervalChecker)(nil)
	_ Checker = (*MaxSizeChecker)(nil)
)

// IntervalChecker checks whether a file should be rolled at regular intervals
//
// If interval <= 0, it will never roll.
type IntervalChecker struct {
	interval time.Duration
}

func NewIntervalChecker(interval time.Duration) *IntervalChecker {
	return &IntervalChecker{
		interval: interval,
	}
}

func (c *IntervalChecker) Name() string {
	return "IntervalChecker"
}

func (c *IntervalChecker) Check(_ string, stat fs.FileInfo) (bool, error) {
	if c.interval <= 0 {
		return false, nil
	}

	st := stat.Sys().(*syscall.Stat_t)
	return time.Now().After(time.Unix(st.Birthtimespec.Unix()).Add(c.interval)), nil
}

// MaxSizeChecker checks whether a file should be rolled when its size exceeds maxSize
type MaxSizeChecker struct {
	maxSize int64
}

func NewMaxSizeChecker(maxSize int64) *MaxSizeChecker {
	return &MaxSizeChecker{
		maxSize: maxSize,
	}
}

func (c *MaxSizeChecker) Name() string {
	return "MaxSizeChecker"
}

func (c *MaxSizeChecker) Check(_ string, stat fs.FileInfo) (bool, error) {
	if c.maxSize <= 0 {
		return false, nil
	}

	st := stat.Sys().(*syscall.Stat_t)
	return st.Size >= c.maxSize, nil
}
