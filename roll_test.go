package rollinguf

import (
	"bufio"
	"log"
	"os"
	"testing"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

func TestNew(t *testing.T) {
	r := New(RollConf{
		FilePath: "/tmp/any_app/app.log",
		RollCheckerConf: RollCheckerConf{
			Interval: 1 * time.Minute,
			MaxSize:  100,
		},
		RollFilterConf: RollFilterConf{
			MaxBackups: 20,
			MaxAge:     2 * time.Minute,
		},
	})
	if r == nil {
		t.Fatal("nil roll")
	}
	SetDebug(true)
	defer func() {
		r.Close()
	}()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
	r.Write([]byte("ccccccccccccccccccc\n"))
}

func TestNewRoll(t *testing.T) {
	r := NewRoll("/tmp/any_app/app.log")
	if r == nil {
		t.Fatal("nil roll")
	}
	SetDebug(true)
	defer func() {
		r.Close()
	}()

	r.WithDefaultChecker(RollCheckerConf{
		Interval: 1 * time.Minute,
		MaxSize:  100,
	}).WithDefaultFilter(RollFilterConf{
		MaxBackups: 2,
		MaxAge:     2 * time.Minute,
	}).WithDefaultProcessor()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
	r.Write([]byte("ccccccccccccccccccc\n"))
}

func BenchmarkNewRollStd(b *testing.B) {
	r := NewRoll("/tmp/any_app/app.log").
		WithChecker(IntervalChecker(24 * time.Hour)).
		WithChecker(MaxSizeChecker(1024 * 1024)).
		WithFilter(MaxBackupsFilter(1)).
		WithFilter(MaxAgeFilter(28 * 24 * time.Hour)).
		WithProcessor(NewDefaultProcessor())

	log.SetOutput(r)
	for i := 0; i < b.N; i++ {
		log.Println("aaaaaaaaaaaaaaaaaaa")
		log.Println("bbbbbbbbbbbbbbbbbbb")
		log.Println("ccccccccccccccccccc")
	}
}

func BenchmarkLumberjackStd(b *testing.B) {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/tmp/any_app/app.lumberjack.log",
		MaxSize:    1, // megabytes
		MaxBackups: 1,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	for i := 0; i < b.N; i++ {
		log.Println("aaaaaaaaaaaaaaaaaaa")
		log.Println("bbbbbbbbbbbbbbbbbbb")
		log.Println("ccccccccccccccccccc")
	}
}

func BenchmarkNewRoll(b *testing.B) {
	l := NewRoll("/tmp/any_app/app.log").
		WithChecker(IntervalChecker(24 * time.Hour)).
		WithChecker(MaxSizeChecker(1024 * 1024)).
		WithFilter(MaxBackupsFilter(1)).
		WithFilter(MaxAgeFilter(28 * 24 * time.Hour)).
		WithProcessor(NewDefaultProcessor())
	if l == nil {
		b.Fatal("nil roll")
	}
	defer l.Close()

	for i := 0; i < b.N; i++ {
		go func() {
			l.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
			l.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
			l.Write([]byte("ccccccccccccccccccc\n"))
		}()
	}
}

func BenchmarkLumberjack(b *testing.B) {
	l := &lumberjack.Logger{
		Filename:   "/tmp/any_app/lumberjack.log",
		MaxSize:    1, // megabytes
		MaxBackups: 1,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
	defer l.Close()

	for i := 0; i < b.N; i++ {
		go func() {
			l.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
			l.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
			l.Write([]byte("ccccccccccccccccccc\n"))
		}()
	}
}

func TestAlign(t *testing.T) {
	pre := "/tmp/any_app/"
	fn := []string{
		"app.log",
		"app.log.1",
		"lumberjack.log",
		"lumberjack-2023-04-23T09-49-34.771.log",
	}

	for _, f := range fn {
		testAlign(pre+f, t)
	}
}

func testAlign(fn string, t *testing.T) {
	f, err := os.Open(fn)
	if err != nil {
		t.Fatal(fn, err)
	}
	defer f.Close()

	scan := bufio.NewScanner(f)
	last := 0
	n := 0
	for scan.Scan() {
		n++
		if last == 0 {
			last = len(scan.Text())
			continue
		}
		if last != len(scan.Text()) {
			t.Fatal(n)
		}
	}
}
