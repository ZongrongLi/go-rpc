/*
 * File: rate_limit.go
 * Project: ratelimit
 * File Created: Saturday, 13th April 2019 7:22:27 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Saturday, 13th April 2019 7:22:58 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */

//TODO 1.限流支持按照各个方法来限流,而不是按照每个client
//2.服务端预留的接口是为了做分布式的集群限流
package ratelimit

import (
	"errors"
	"time"
)

type RateLimiter interface {
	Acquire()
	TryAcquire() bool
	AcquireWithTimeout(duration time.Duration) error
}

type DefaultRateLimiter struct {
	Num         int64
	rateLimiter chan time.Time
	Threshold   int64
}

func NewRateLimiter(numPerSecond int64, threshold int64) RateLimiter {
	r := new(DefaultRateLimiter)
	r.Num = numPerSecond
	r.Threshold = threshold
	r.rateLimiter = make(chan time.Time, threshold)
	go func() {
		d := time.Duration(numPerSecond)
		ticker := time.NewTicker(time.Second / d)
		for t := range ticker.C {
			r.rateLimiter <- t
		}
	}()

	return r
}

func (r *DefaultRateLimiter) Acquire() {
	<-r.rateLimiter
}

func (r *DefaultRateLimiter) TryAcquire() bool {
	select {
	case <-r.rateLimiter:
		return true
	default:
		return false
	}
}

func (r *DefaultRateLimiter) AcquireWithTimeout(timeout time.Duration) error {
	ticker := time.NewTicker(timeout)
	select {
	case <-r.rateLimiter:
		return nil
	case <-ticker.C:
		return errors.New("acquire timeout")

	}
}

type RateLimitWrapper struct {
	global       RateLimiter
	methodLimits map[string]RateLimiter //Service.Method为key
}
