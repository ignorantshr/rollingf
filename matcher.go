// Copyright 2023 ignorantshr.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rollingf

import (
	"regexp"
	"strings"
	"sync"
)

// Matcher mathes the files for further processing
type Matcher interface {
	// Match return true if other file base name matches
	Match(other string) bool

	// Init the matcher with the file's base name
	Init(base string)
}

var _ Matcher = (*regexMatcher)(nil)

type regexMatcher struct {
	suffixPattern string
	reg           *regexp.Regexp
	once          sync.Once
}

// DefaultMatcher matches the simple file names
//
// eg.
// app.log app.log.1 app.log.2 ...
func DefaultMatcher() *regexMatcher {
	return NewRegexMatcher(`(\.\d+)?$`)
}

// CompressMatcher matches the file names with the .1.gz suffix
//
// eg.
// app.log app.log.1.gz app.log.2.gz ...
func CompressMatcher(format CompressFormat) *regexMatcher {
	return NewRegexMatcher(`(\.\d+\` + cfSuffix[format] + `)?$`)
}

func NewRegexMatcher(suffixPattern string) *regexMatcher {
	return &regexMatcher{
		suffixPattern: suffixPattern,
		once:          sync.Once{},
	}
}

func (p *regexMatcher) Match(other string) bool {
	return len(p.reg.Find([]byte(other))) == len(other)
}

func (m *regexMatcher) Init(base string) {
	m.once.Do(func() {
		m.reg = regexp.MustCompile("^" + strings.ReplaceAll(base, ".", `\.`) + m.suffixPattern)
		debug("[regexMatcher] pattern: %v", m.reg)
	})
}
