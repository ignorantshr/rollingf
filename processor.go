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
	"compress/gzip"
	"compress/zlib"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

// Processor processes the remaining files after filtering
type Processor interface {
	// Process process the remaining files after filtering
	Process(dir string, remains []os.DirEntry) error
}

var (
	_ Processor = (*defaultProcessor)(nil)
	_ Processor = (*compressor)(nil)

	_defaultProcessor = &defaultProcessor{}
)

type baseProcessor struct {
	each func(dir, base string) error
}

func (p *baseProcessor) Process(dir string, remains []os.DirEntry) error {
	if len(remains) == 0 {
		return nil
	}

	// process the files in reverse order
	for i := len(remains) - 1; i >= 0; i-- {
		if err := p.each(dir, remains[i].Name()); err != nil {
			return err
		}
	}

	return nil
}

type defaultProcessor struct {
	b *baseProcessor
}

// DefaultProcessor renames the files, increase the tail number of the file name.
func DefaultProcessor() *defaultProcessor {
	p := &defaultProcessor{}

	p.b = &baseProcessor{
		p.each,
	}
	return p
}

func (p *defaultProcessor) Process(dir string, remains []os.DirEntry) error {
	return p.b.Process(dir, remains)
}

func (p *defaultProcessor) each(dir, base string) error {
	newName := p.incrTailNumber(base)

	debug("[Rename] %v --> %v", base, newName)
	if err := os.Rename(path.Join(dir, base), path.Join(dir, newName)); err != nil {
		return err
	}
	return nil
}

// incrTailNumber increase the tail number of the file name.
//
// eg.
//
//	base: "abc.log",
//	return: "abc.log.1"
func (p *defaultProcessor) incrTailNumber(base string) string {
	if len(base) == 0 {
		return base
	}

	tail := 1
	last := path.Ext(base)
	if len(last) > 0 {
		last = last[1:]
	}
	pre := base
	if IsNumeric(last) {
		tail, _ = strconv.Atoi(last)
		tail++
		pre = base[:len(base)-len(last)-1]
	}
	return pre + "." + strconv.Itoa(tail)
}

type CompressFormat string

const (
	NoCompress CompressFormat = ""

	Gzip CompressFormat = "gzip"
	Zlib CompressFormat = "zlib"
)

var cfSuffix = map[CompressFormat]string{
	Gzip: ".gz",
	Zlib: ".z",
}

func getCompressWriter(format CompressFormat, f io.Writer) io.WriteCloser {
	var w io.WriteCloser
	switch format {
	case Gzip:
		w = gzip.NewWriter(f)
	case Zlib:
		w = zlib.NewWriter(f)
	}
	return w
}

type compressor struct {
	b *baseProcessor

	format      CompressFormat
	suffix      string
	suffixFirst string
	suffixLen   int
}

// Compressor compresses and rename the files
//
// eg.
//
//	base: "abc.log",
//	return: "abc.log.1.gz"
func Compressor(format CompressFormat) *compressor {
	c := &compressor{}

	c.b = &baseProcessor{
		c.each,
	}

	c.format = format
	c.suffix = cfSuffix[format]
	c.suffixLen = len(c.suffix)
	c.suffixFirst = ".1" + c.suffix
	if c.suffix == "" {
		c.format = NoCompress
	}

	return c
}

func (p *compressor) Process(dir string, remains []os.DirEntry) error {
	return p.b.Process(dir, remains)
}

func (p *compressor) each(dir, base string) error {
	var newName string
	if p.format == NoCompress {
		// dagrade to rename
		newName = _defaultProcessor.incrTailNumber(base)
	} else {
		newName = p.incrTailNumber(base)
	}

	if newName != base+p.suffixFirst || p.format == NoCompress {
		debug("[Rename] %v --> %v", base, newName)
		return renameFile(dir, base, newName)
	}

	debug("[Compress] %v --> %v", base, newName)
	of, err := os.OpenFile(path.Join(dir, base), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer of.Close()

	nf, err := os.OpenFile(path.Join(dir, newName), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer nf.Close()

	w := getCompressWriter(p.format, nf)
	defer w.Close()

	if _, err := io.Copy(w, of); err != nil {
		if err := removeFile(dir, newName); err != nil {
			return err
		}
		return err
	}

	return removeFile(dir, base)
}

func (p *compressor) incrTailNumber(base string) string {
	if len(base) == 0 {
		return base
	}

	groups := strings.Split(base, ".")

	var last string
	var penultimate string
	if len(base) > p.suffixLen {
		last = base[len(base)-p.suffixLen:]
		if last != p.suffix {
			last = ""
		} else {
			if len(groups) >= 3 {
				penultimate = groups[len(groups)-2]
			}
		}
	}

	tail := 1
	var pre string
	if IsNumeric(penultimate) {
		tail, _ = strconv.Atoi(penultimate)
		tail++
		pre = strings.Join(groups[:len(groups)-2], ".")
	} else {
		pre = base
	}
	return pre + "." + strconv.Itoa(tail) + p.suffix
}

func renameFile(dir, oldName, newName string) error {
	return os.Rename(path.Join(dir, oldName), path.Join(dir, newName))
}

func removeFile(dir, oldName string) error {
	return os.Remove(path.Join(dir, oldName))
}
