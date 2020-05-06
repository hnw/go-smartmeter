package smartmeter

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type skQuery struct {
	s             *Smartmeter
	command       string
	retry         int
	retryInterval time.Duration
	timeout       time.Duration
	receiver      func(string) (bool, error)
	debug         bool
}

var RetryableError = errors.New("retrying...")

func NewSKQuery(s *Smartmeter, command string, opts ...Option) (*skQuery, error) {
	q := &skQuery{
		s:             s,
		command:       command,
		retryInterval: 2 * time.Second,
		timeout:       10 * time.Second,
		receiver: func(line string) (bool, error) {
			// デフォルトのreceiver。「OK」まで読む。SKコマンドの大半はこれで対応できる。
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

func (q *skQuery) Exec() (res string, err error) {
	if q.debug {
		fmt.Printf(">> %s\n", q.command)
	}
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
			return "", fmt.Errorf("Q command timeout (%dsec)", q.timeout/time.Second)
		case line, ok := <-q.s.inputChan:
			if !ok {
				return "", errors.New("Q command read error")
			}
			if q.debug {
				fmt.Printf("<< %s\n", line)
			}
			if strings.HasPrefix(line, "FAIL ") {
				return "", fmt.Errorf("Q command response error: %s", line)
			}
			var ret bool
			ret, err = q.receiver(line)
			if err != nil {
				if errors.Is(err, RetryableError) {
					q.retry--
					if q.retry >= 0 {
						if q.debug {
							fmt.Printf("%+v\n", err)
						}
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
