package server

import (
	"net/http"
)

type HTTPServeFunc func(rw *http.ResponseWriter, r *http.Request, next *Middleware)

type Middleware struct {
	F    HTTPServeFunc
	GoOn *Middleware
}

func DefaultHTTPServeFunc(rw *http.ResponseWriter, r *http.Request, next *Middleware) {
	//	fmt.Println("========================================black function")
	return
}

var Wrappers []HTTPServeFunc

func chain(next *Middleware, others ...HTTPServeFunc) *Middleware {

	for i := len(others) - 1; i >= 0; i-- {
		next = &Middleware{
			F:    others[i],
			GoOn: next,
		}
	}

	return next
}
func (m *Middleware) Next(rw *http.ResponseWriter, r *http.Request) {
	if m == nil {
		return
	}
	m.F(rw, r, m.GoOn)
}
