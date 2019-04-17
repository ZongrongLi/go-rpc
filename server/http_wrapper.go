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

func chain(beginPoint HTTPServeFunc, others ...HTTPServeFunc) *Middleware {

	goon := &Middleware{
		F:    beginPoint,
		GoOn: nil,
	}
	for i := len(others) - 1; i >= 0; i-- {
		goon = &Middleware{
			F:    others[i],
			GoOn: goon,
		}
	}

	return goon
}
func (m *Middleware) Next(rw *http.ResponseWriter, r *http.Request) {
	if m == nil {
		return
	}
	m.F(rw, r, m.GoOn)
}
