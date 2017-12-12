package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"backup/github.com/pkg/errors"
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	In      string
	PathFmt string
	Period  time.Duration
}

type outFile struct {
	config
	period chan bool
	out    *os.File
	//wbuf   *bufio.Writer
}

func time2Path(p string, t time.Time) string {
	p = strings.Replace(p, "%Y", fmt.Sprintf("%04d", t.Year()), -1)
	p = strings.Replace(p, "%y", fmt.Sprintf("%02d", t.Year()%100), -1)
	p = strings.Replace(p, "%m", fmt.Sprintf("%02d", t.Month()), -1)
	p = strings.Replace(p, "%d", fmt.Sprintf("%02d", t.Day()), -1)
	p = strings.Replace(p, "%H", fmt.Sprintf("%02d", t.Hour()), -1)
	p = strings.Replace(p, "%M", fmt.Sprintf("%02d", t.Minute()), -1)
	p = strings.Replace(p, "%S", fmt.Sprintf("%02d", t.Second()), -1)
	if strings.Index(p, "%N") == -1 { // 数値指定がなければ終わる
		return p
	}
	now := time.Now()
	now = now.Truncate(time.Hour).Add(-time.Duration(now.Hour()) * time.Hour)
	num := int((now.Sub(t) / 24).Hours())
	p = strings.Replace(p, "%N", fmt.Sprintf("%d", num), -1)
	return p

}

func (o *outFile) lineFunc(b []byte) error {
	select {
	case <-o.period:
		//o.wbuf.Flush()
		o.out.Close()
		o.out = openFile(time2Path(o.PathFmt, time.Now()))
		//o.wbuf = bufio.NewWriter(o.out)
	default:
	}
	//o.wbuf.Write(b)
	o.out.Write(b)
	return nil
}

func openFile(f string) *os.File {
	file, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func newOutFile(c config) *outFile {
	o := outFile{
		config: c,
		period: make(chan bool),
	}
	o.out = openFile(time2Path(o.PathFmt, time.Now()))
	//o.wbuf = bufio.NewWriter(o.out)
	go makePathWorker(o, o.period)
	return &o
}

func makePathWorker(o outFile, period chan bool) {
	timeSlice := truncate(time.Now(), o.Period)
	t := time.NewTimer(timeSlice.Sub(time.Now()))
	for {
		<-t.C
		period <- true
		timeSlice = timeSlice.Add(o.Period)
		t = time.NewTimer(timeSlice.Sub(time.Now()))
	}
}

// ReadSplitFile is scanning split file.
func ReadSplitFile(inFile string, lineFunc func([]byte) error) error {
	var err error
	in := &os.File{}
	r := bufio.NewReader(in)
	for {
		in, err = os.Open(inFile)
		if err != nil {
			log.Fatal(err)
		}
		r.Reset(in)
	L:
		for {
			line, err := r.ReadBytes('\n')
			if err != nil && err != io.EOF {
				return errors.Wrap(err, "ReadSplitFile Readbytes err")
			}
			if len(line) > 0 {
				if err := lineFunc(line); err != nil {
					return errors.Wrap(err, "lineFunc err")
				}
			}
			if err == io.EOF {
				in.Close()
				break L
			}
		}
	}
	return nil
}

// time.Truncateを 1 day(24*time.Hour)を指定された場合にtimezoneを考慮するように
// see: http://qiita.com/umisama/items/b50df4888665fc36346e
func truncate(t time.Time, d time.Duration) time.Time {
	if d == 24*time.Hour {
		return t.Truncate(time.Hour).Add(-time.Duration(t.Hour()) * time.Hour)
	}
	return t.Truncate(d)
}

func main() {
	var c config
	if err := envconfig.Process("", &c); err != nil {
		log.Fatal(err)
	}
	o := newOutFile(c)
	ReadSplitFile(c.In, o.lineFunc)

}
