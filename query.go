package smartmeter

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type query struct {
	s             *Device
	command       string
	retry         int
	retryInterval time.Duration
	timeout       time.Duration
	reader        func(string) (bool, error)
	logger        *log.Logger
	verbosity     int
}

var RetryableError = errors.New("retrying...")

func NewSKQuery(s *Device, command string, opts ...Option) (*query, error) {
	q := &query{
		s:             s,
		command:       command,
		retryInterval: 1 * time.Second,
		timeout:       10 * time.Second,
		reader: func(line string) (bool, error) {
			// デフォルreader。「OK」まで読む。SKコマンドの大半はこれで対応できる。
			if line == "OK" {
				return true, nil
			}
			return false, nil
		},
	}
	for _, opt := range opts {
		if err := opt(q); err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (q *query) Exec() (res string, err error) {
	q.debugf(">> %s\n", q.command)
	_, err = q.s.writer.WriteString(q.command + "\r\n")
	if err != nil {
		return
	}
	err = q.s.writer.Flush()
	if err != nil {
		return
	}

	tm := time.NewTimer(q.timeout)
	for {
		select {
		case <-tm.C:
			return "", fmt.Errorf("SK command timeout (%dsec)", q.timeout/time.Second)
		case line, ok := <-q.s.inputChan:
			if !ok {
				return "", errors.New("SK command read error")
			}
			q.debugf("<< %s\n", line)
			if strings.HasPrefix(line, "FAIL ") {
				return "", fmt.Errorf("SK command response error: %s", line)
			}
			var ret bool
			ret, err = q.reader(line)
			if err != nil {
				if errors.Is(err, RetryableError) {
					q.retry--
					if q.retry >= 0 {
						q.warnf("Error: %+v\n", err)
						time.Sleep(q.retryInterval)
						//本当はループにすべきなんだけど手抜きで再帰
						return q.Exec()
					}
				}
				return
			}
			res += "\n" + line
			if ret == true {
				return
			}
		}
	}
}

func (q *query) warnf(fmt string, v ...interface{}) {
	if q.verbosity >= 1 {
		q.logf(fmt, v...)
	}
}

func (q *query) infof(fmt string, v ...interface{}) {
	if q.verbosity >= 2 && q.logger != nil {
		q.logf(fmt, v...)
	}
}

func (q *query) debugf(fmt string, v ...interface{}) {
	if q.verbosity >= 3 && q.logger != nil {
		q.logf(fmt, v...)
	}
}

func (q *query) logf(fmt string, v ...interface{}) {
	if q.logger != nil {
		q.logger.Printf(fmt, v...)
	} else {
		log.Printf(fmt, v...)
	}
}
