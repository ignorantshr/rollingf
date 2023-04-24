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
	"os"
	"path"
	"time"
)

// Filter filters out the sorted-by-filename files that need to be processed
type Filter interface {
	Name() string
	Filter(input []os.DirEntry) (remains []os.DirEntry, filtered []os.DirEntry, err error)
	DealFiltered(dir string, filtered []os.DirEntry) error
}

var (
	_ Filter = (*maxBackupsFilter)(nil)
	_ Filter = (*maxAgeFilter)(nil)
)

// MaxSizeFilter filter files by size
type maxBackupsFilter struct {
	maxBackups int
}

func MaxBackupsFilter(maxBackups int) *maxBackupsFilter {
	return &maxBackupsFilter{
		maxBackups: maxBackups,
	}
}

func (f *maxBackupsFilter) Name() string {
	return "MaxBackupsFilter"
}

func (f *maxBackupsFilter) Filter(files []os.DirEntry) ([]os.DirEntry, []os.DirEntry, error) {
	var removes []os.DirEntry
	if f.maxBackups >= 0 && len(files) > f.maxBackups {
		removes = files[f.maxBackups:]
	}
	return files[:min(len(files), f.maxBackups)], removes, nil
}

func (f *maxBackupsFilter) DealFiltered(dir string, filtered []os.DirEntry) error {
	for _, file := range filtered {
		debug("[remove] %v", file.Name())
		if err := os.Remove(path.Join(dir, file.Name())); err != nil {
			return err
		}
	}
	return nil
}

// maxAgeFilter filter files by age
type maxAgeFilter struct {
	maxAge time.Duration
}

func MaxAgeFilter(maxAge time.Duration) (obj *maxAgeFilter) {
	return &maxAgeFilter{
		maxAge: maxAge,
	}
}

func (f *maxAgeFilter) Name() string {
	return "MaxAgeFilter"
}

func (f *maxAgeFilter) Filter(files []os.DirEntry) ([]os.DirEntry, []os.DirEntry, error) {
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

func (f *maxAgeFilter) DealFiltered(dir string, filtered []os.DirEntry) error {
	for _, file := range filtered {
		debug("[remove] %v", file.Name())
		if err := os.Remove(path.Join(dir, file.Name())); err != nil {
			return err
		}
	}
	return nil
}
