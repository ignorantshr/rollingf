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
	"time"
)

// Filter filters out the files that need to be deleted sorted by file name
type Filter interface {
	Name() string
	Filter(input []os.DirEntry) (remains []os.DirEntry, removed []os.DirEntry, err error)
}

var (
	_ Filter = (*MaxBackupsFilter)(nil)
	_ Filter = (*MaxAgeFilter)(nil)
)

// MaxSizeFilter filter files by size
type MaxBackupsFilter struct {
	maxBackups int
}

func NewMaxBackupsFilter(maxBackups int) *MaxBackupsFilter {
	return &MaxBackupsFilter{
		maxBackups: maxBackups,
	}
}

func (f *MaxBackupsFilter) Name() string {
	return "MaxBackupsFilter"
}

func (f *MaxBackupsFilter) Filter(files []os.DirEntry) ([]os.DirEntry, []os.DirEntry, error) {
	var removes []os.DirEntry
	if f.maxBackups >= 0 && len(files) > f.maxBackups {
		removes = files[f.maxBackups:]
	}
	return files[:min(len(files), f.maxBackups)], removes, nil
}

// MaxAgeFilter filter files by age
type MaxAgeFilter struct {
	maxAge time.Duration
}

func NewMaxAgeFilter(maxAge time.Duration) (obj *MaxAgeFilter) {
	return &MaxAgeFilter{
		maxAge: maxAge,
	}
}

func (f *MaxAgeFilter) Name() string {
	return "MaxAgeFilter"
}

func (f *MaxAgeFilter) Filter(files []os.DirEntry) ([]os.DirEntry, []os.DirEntry, error) {
	// todo binary search improve
	if f.maxAge <= 0 {
		return nil, nil, nil
	}

	var idx int
	for ; idx < len(files); idx++ {
		info, err := files[idx].Info()
		if err != nil {
			return nil, nil, err
		}
		if time.Since(info.ModTime()) >= f.maxAge {
			break
		}
	}
	return files[:idx], files[idx:], nil
}
