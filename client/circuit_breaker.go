/*
 * File: circuit_breaker.go
 * Project: client
 * File Created: Thursday, 11th April 2019 6:07:03 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Saturday, 13th April 2019 3:33:29 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */

package client

import (
	"sync/atomic"
	"time"
)

type CircuitBreaker interface {
	AllowRequest() bool
	Success()
	Fail(err error)
}

type DefaultCircuitBreaker struct {
	lastFail  time.Time
	fails     uint64
	threshold uint64
	window    time.Duration
}

func (cb *DefaultCircuitBreaker) AllowRequest() bool {
	if time.Since(cb.lastFail) > cb.window {
		cb.reset()
		return true
	}
	failures := atomic.LoadUint64(&cb.fails)
	return failures < cb.threshold
}

func NewDefaultCircuitBreaker(threshold uint64, window time.Duration) *DefaultCircuitBreaker {
	return &DefaultCircuitBreaker{
		threshold: threshold,
		window:    window,
	}
}

func (cb *DefaultCircuitBreaker) Success() {
	cb.reset()
}

func (cb *DefaultCircuitBreaker) Fail(err error) {
	atomic.AddUint64(&cb.fails, 1)
	cb.lastFail = time.Now()
}

func (cb *DefaultCircuitBreaker) reset() {
	atomic.StoreUint64(&cb.fails, 0)
	cb.lastFail = time.Now()
}
