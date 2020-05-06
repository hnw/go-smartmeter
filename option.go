// Functional Option Patternによるオプション指定
// SmartmeterとskQueryの両方共通で使うためにfunc(interface{})になっている
// Smartmeterで指定したオプションは全skQueryに引き継がれる

package smartmeter

import (
	"time"
)

type option func(interface{}) error

func ID(id string) option {
	return func(v interface{}) error {
		if s, ok := v.(*Smartmeter); ok {
			s.ID = id
		}
		return nil
	}
}

func Password(pw string) option {
	return func(v interface{}) error {
		if s, ok := v.(*Smartmeter); ok {
			s.Password = pw
		}
		return nil
	}
}

func Channel(channel string) option {
	return func(v interface{}) error {
		if s, ok := v.(*Smartmeter); ok {
			s.Channel = channel
		}
		return nil
	}
}

func IPAddr(ipAddr string) option {
	return func(v interface{}) error {
		if s, ok := v.(*Smartmeter); ok {
			s.IPAddr = ipAddr
		}
		return nil
	}
}

func DualStackSK() option {
	return func(v interface{}) error {
		if s, ok := v.(*Smartmeter); ok {
			s.DualStackSK = true
		}
		return nil
	}
}

func Retry(count int) option {
	return func(v interface{}) error {
		if q, ok := v.(*skQuery); ok {
			q.retry = count
		}
		return nil
	}
}

func RetryInterval(d time.Duration) option {
	return func(v interface{}) error {
		if q, ok := v.(*skQuery); ok {
			q.retryInterval = d
		}
		return nil
	}
}

func Timeout(d time.Duration) option {
	return func(v interface{}) error {
		if q, ok := v.(*skQuery); ok {
			q.timeout = d
		}
		return nil
	}
}

func Receiver(callback func(string) (bool, error)) option {
	return func(v interface{}) error {
		if q, ok := v.(*skQuery); ok {
			q.receiver = callback
		}
		return nil
	}
}

func Debug() option {
	return func(v interface{}) error {
		if s, ok := v.(*Smartmeter); ok {
			s.Debug = true
		}
		if q, ok := v.(*skQuery); ok {
			q.debug = true
		}
		return nil
	}
}
