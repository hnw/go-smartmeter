// Functional Option Patternによるオプション指定
// Deviceとqueryの両方共通で使うためにfunc(interface{})になっている
// Deviceで指定したオプションは全queryに引き継がれる

package smartmeter

import (
	"log"
	"time"
)

// Option configures a Device or query.
type Option func(interface{}) error

// ID sets the B-route authentication ID.
func ID(id string) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.ID = id
		}
		return nil
	}
}

// Password sets the B-route authentication password.
func Password(pw string) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.Password = pw
		}
		return nil
	}
}

// Channel sets the channel used for scanning or joining.
func Channel(channel string) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.Channel = channel
		}
		return nil
	}
}

// IPAddr sets the IPv6 address of the smart meter.
func IPAddr(ipAddr string) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.IPAddr = ipAddr
		}
		return nil
	}
}

// DualStackSK enables or disables the dual stack SK behavior.
func DualStackSK(v bool) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.DualStackSK = v
		}
		return nil
	}
}

// Retry sets how many times a query should retry on ErrRetryable.
func Retry(count int) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.retry = count
		}
		return nil
	}
}

// RetryInterval sets the duration between retries.
func RetryInterval(d time.Duration) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.retryInterval = d
		}
		return nil
	}
}

// Timeout sets the query timeout duration.
func Timeout(d time.Duration) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.timeout = d
		}
		return nil
	}
}

// Reader sets a custom line reader callback for SK command responses.
func Reader(callback func(string) (bool, error)) Option {
	return func(tgt interface{}) error {
		if q, ok := tgt.(*query); ok {
			q.reader = callback
		}
		return nil
	}
}

// Logger sets the logger for Device and query.
func Logger(logger *log.Logger) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.logger = logger
		}
		if q, ok := tgt.(*query); ok {
			q.logger = logger
		}
		return nil
	}
}

// Verbosity sets the logging verbosity for Device and query.
func Verbosity(v int) Option {
	return func(tgt interface{}) error {
		if d, ok := tgt.(*Device); ok {
			d.Verbosity = v
		}
		if q, ok := tgt.(*query); ok {
			q.verbosity = v
		}
		return nil
	}
}
