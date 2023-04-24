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
	"syscall"
	"time"
)

// Checker is a file checker, it checks if a file is shall be rolled.
type Checker interface {
	Name() string
	Check(filePath string) (bool, error)
}

// IntervalChecker checks whether a file should be rolled at regular intervals
//
// If internal <= 0, it will never roll.
type IntervalChecker struct {
	internal time.Duration
}

func NewIntervalChecker(internal time.Duration) *IntervalChecker {
	return &IntervalChecker{
		internal: internal,
	}
}

func (c *IntervalChecker) Name() string {
	return "IntervalChecker"
}

func (c *IntervalChecker) Check(filePath string) (bool, error) {
	if c.internal <= 0 {
		return false, nil
	}

	var st syscall.Stat_t
	if err := syscall.Stat(filePath, &st); err != nil {
		return false, err
	}
	if time.Now().Before(time.Unix(st.Birthtimespec.Unix()).Add(c.internal)) {
		return false, nil
	}

	return true, nil
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

func (c *MaxSizeChecker) Check(file string) (bool, error) {
	if c.maxSize <= 0 {
		return false, nil
	}

	var st syscall.Stat_t
	if err := syscall.Stat(file, &st); err != nil {
		return false, err
	}

	if st.Size < c.maxSize {
		return false, nil
	}

	return true, nil
}
