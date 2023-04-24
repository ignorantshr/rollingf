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

import "time"

type RollConf struct {
	FilePath string // file's location
	RollCheckerConf
	RollFilterConf
}

type RollCheckerConf struct {
	// interval to roll file
	Interval time.Duration

	// the max bytes to roll file
	MaxSize int64
}

type RollFilterConf struct {
	// the max day to keep old files, only triggered when call Write()
	MaxAge time.Duration

	// the max number of old log files to retain
	MaxBackups int
}

func NewRollConf(filePath string, interval time.Duration, maxSize int64, maxAge time.Duration, maxBackups int) RollConf {
	return RollConf{
		FilePath: filePath,

		RollCheckerConf: RollCheckerConf{
			Interval: interval,
			MaxSize:  maxSize,
		},

		RollFilterConf: RollFilterConf{
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
		},
	}
}

func DefaultChecker(c RollCheckerConf) []Checker {
	return []Checker{
		IntervalChecker(c.Interval),
		MaxSizeChecker(c.MaxSize),
	}
}

func DefaultFilter(c RollFilterConf) []Filter {
	return []Filter{
		MaxBackupsFilter(c.MaxBackups),
		MaxAgeFilter(c.MaxAge),
	}
}
