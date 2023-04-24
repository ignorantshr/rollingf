package rollinguf

import (
	"log"
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

func BenchmarkNewRoll(b *testing.B) {
	r := NewRoll("/tmp/any_app/app.log").
		WithChecker(NewMaxSizeChecker(1024 * 1024)).
		WithFilter(NewMaxBackupsFilter(1)).
		WithFilter(NewMaxAgeFilter(28 * 24 * time.Hour)).
		WithProcessor(NewDefaultProcessor())

	defer func() {
		r.Close()
	}()

	log.SetOutput(r)
	for i := 0; i < b.N; i++ {
		log.Println([]byte("aaaaaaaaaaaaaaaaaaa\n"))
		log.Println([]byte("bbbbbbbbbbbbbbbbbbb\n"))
		log.Println([]byte("ccccccccccccccccccc\n"))
	}
}

func BenchmarkLumberjack(b *testing.B) {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/tmp/any_app/app.lumberjack.log",
		MaxSize:    1, // megabytes
		MaxBackups: 1,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	for i := 0; i < b.N; i++ {
		log.Println([]byte("aaaaaaaaaaaaaaaaaaa\n"))
		log.Println([]byte("bbbbbbbbbbbbbbbbbbb\n"))
		log.Println([]byte("ccccccccccccccccccc\n"))
	}
}
