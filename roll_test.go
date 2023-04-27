package rollingf

import (
	"bufio"
	"os"
	"sync"
	"testing"
	"time"
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

func TestNewC(t *testing.T) {
	r := NewC("/tmp/any_app/app.log")
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
	}).WithDefaultMatcher().WithDefaultProcessor()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
	r.Write([]byte("ccccccccccccccccccc\n"))
}

func TestNewRollSimple(t *testing.T) {
	r := New(NewRollConf("/tmp/any_app/app.log", 1*time.Minute, 100, 2*time.Minute, 2))
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
	}).WithDefaultMatcher().WithDefaultProcessor()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
	r.Write([]byte("ccccccccccccccccccc\n"))
}

func TestOptionCompress(t *testing.T) {
	r := New(NewRollConf("/tmp/any_app/app.log", 1*time.Minute, 100, 10*time.Minute, 5)).WithOptions(
		Compress(Gzip),
	)

	SetDebug(true)
	defer r.Close()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
	r.Write([]byte("ccccccccccccccccccc\n"))
}

func TestCompressorDegrade(t *testing.T) {
	r := New(
		NewRollConf("/tmp/any_app/app.log", 1*time.Minute, 100, 10*time.Minute, 5),
	).WithProcessor(Compressor("no support"))
	SetDebug(true)
	defer r.Close()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
	r.Write([]byte("ccccccccccccccccccc\n"))
}

func TestConccurent(t *testing.T) {
	r := NewC("/tmp/any_app/app.log").
		WithChecker(IntervalChecker(24 * time.Hour)).
		WithChecker(MaxSizeChecker(1024 * 1024)).
		WithFilter(MaxBackupsFilter(5)).
		WithFilter(MaxAgeFilter(28 * 24 * time.Hour)).
		WithDefaultMatcher().
		WithDefaultProcessor()
	if r == nil {
		t.Fatal("nil roll")
	}
	SetDebug(true)
	defer r.Close()

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
			r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
			r.Write([]byte("ccccccccccccccccccc\n"))
		}()
	}
	wg.Wait()
}

func BenchmarkNewC(b *testing.B) {
	r := NewC("/tmp/any_app/app.log").
		WithChecker(IntervalChecker(24 * time.Hour)).
		WithChecker(MaxSizeChecker(1024 * 1024)).
		WithFilter(MaxBackupsFilter(5)).
		WithFilter(MaxAgeFilter(28 * 24 * time.Hour)).
		WithDefaultMatcher().
		WithDefaultProcessor()
	if r == nil {
		b.Fatal("nil roll")
	}
	defer r.Close()

	wg := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
			r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
			r.Write([]byte("ccccccccccccccccccc\n"))
		}()
	}
	wg.Wait()
}

func BenchmarkNewCWithoutLock(b *testing.B) {
	r := NewC("/tmp/any_app/app.log").
		WithChecker(IntervalChecker(24 * time.Hour)).
		WithChecker(MaxSizeChecker(1024 * 1024)).
		WithFilter(MaxBackupsFilter(50)).
		WithFilter(MaxAgeFilter(28 * 24 * time.Hour)).
		WithDefaultMatcher().
		WithDefaultProcessor().
		WithOptions(Lock(false))
	if r == nil {
		b.Fatal("nil roll")
	}
	defer r.Close()

	wg := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
			r.Write([]byte("bbbbbbbbbbbbbbbbbbb\n"))
			r.Write([]byte("ccccccccccccccccccc\n"))
		}()
	}
	wg.Wait()
}

func TestAlign(t *testing.T) {
	pre := "/tmp/any_app/"
	fn := []string{
		"app.log",
		// "app.log.1144",
		// "app.log.2254",
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
