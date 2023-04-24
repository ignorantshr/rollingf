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
	"time"
)

// Checker is a file checker, it checks if a file is shall be rolled.
type Checker interface {
	Name() string
	Check(filePath string, st *Rstat) (bool, error)
}

var (
	_ Checker = (*intervalChecker)(nil)
	_ Checker = (*maxSizeChecker)(nil)
)

// intervalChecker checks whether a file should be rolled at regular intervals
//
// If interval <= 0, it will never roll.
type intervalChecker struct {
	interval time.Duration
}

func IntervalChecker(interval time.Duration) *intervalChecker {
	return &intervalChecker{
		interval: interval,
	}
}

func (c *intervalChecker) Name() string {
	return "IntervalChecker"
}

func (c *intervalChecker) Check(_ string, st *Rstat) (bool, error) {
	if c.interval <= 0 {
		return false, nil
	}

	ok, brithTime := st.Birthtimespec()
	if !ok {
		return false, nil
	}
	return time.Now().After(time.Unix(brithTime.Unix()).Add(c.interval)), nil
}

// maxSizeChecker checks whether a file should be rolled when its size exceeds maxSize
type maxSizeChecker struct {
	maxSize int64
}

func MaxSizeChecker(maxSize int64) *maxSizeChecker {
	return &maxSizeChecker{
		maxSize: maxSize,
	}
}

func (c *maxSizeChecker) Name() string {
	return "MaxSizeChecker"
}

func (c *maxSizeChecker) Check(_ string, st *Rstat) (bool, error) {
	if c.maxSize <= 0 {
		return false, nil
	}

	return st.Size() >= c.maxSize, nil
}
