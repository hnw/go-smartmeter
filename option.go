// Functional Option Patternによるオプション指定
// Deviceとqueryの両方共通で使うためにfunc(interface{})になっている
// Deviceで指定したオプションは全queryに引き継がれる

package smartmeter

import (
	"time"
)

type Option func(interface{}) error

func ID(id string) Option {
	return func(tgt interface{}) error {
		if s, ok := tgt.(*Device); ok {
			s.ID = id
		}
		return nil
	}
}

func Password(pw string) Option {
	return func(tgt interface{}) error {
		if s, ok := tgt.(*Device); ok {
			s.Password = pw
		}
		return nil
	}
}

func Channel(channel string) Option {
	return func(tgt interface{}) error {
		if s, ok := tgt.(*Device); ok {
			s.Channel = channel
		}
		return nil
	}
}

func IPAddr(ipAddr string) Option {
	return func(tgt interface{}) error {
		if s, ok := tgt.(*Device); ok {
			s.IPAddr = ipAddr
		}
		return nil
	}
}

func DualStackSK(v bool) Option {
	return func(tgt interface{}) error {
		if s, ok := tgt.(*Device); ok {
			s.DualStackSK = v
		}
		return nil
	}
}

func Retry(count int) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.retry = count
		}
		return nil
	}
}

func RetryInterval(d time.Duration) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.retryInterval = d
		}
		return nil
	}
}

func Timeout(d time.Duration) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.timeout = d
		}
		return nil
	}
}

func Reader(callback func(string) (bool, error)) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.reader = callback
		}
		return nil
	}
}

func Debug(v bool) Option {
	return func(tgt interface{}) error {
		if s, ok := tgt.(*Device); ok {
			s.Debug = v
		}
		if q, ok := tgt.(*query); ok {
			q.debug = v
		}
		return nil
	}
}
