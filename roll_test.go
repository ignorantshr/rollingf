package rollinguf

import (
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
	defer r.Close()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("baaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("caaaaaaaaaaaaaaaaaa\n"))
}

func TestNewRoll(t *testing.T) {
	r := NewRoll("/tmp/any_app/app.log")
	if r == nil {
		t.Fatal("nil roll")
	}
	defer r.Close()

	r.WithDefaultChecker(RollCheckerConf{
		Interval: 1 * time.Minute,
		MaxSize:  100,
	}).WithDefaultFilter(RollFilterConf{
		MaxBackups: 20,
		MaxAge:     2 * time.Minute,
	}).WithDefaultProcessor()

	r.Write([]byte("aaaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("baaaaaaaaaaaaaaaaaa\n"))
	r.Write([]byte("caaaaaaaaaaaaaaaaaa\n"))
}
