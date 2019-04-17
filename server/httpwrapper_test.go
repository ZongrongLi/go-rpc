package server

import (
	"fmt"
	"net/http"
	"testing"
)

func testfun1(rw *http.ResponseWriter, r *http.Request, c *Middleware) {
	fmt.Println("before===testfunc1")
	c.Next(nil, nil)

	fmt.Println("after===testfunc1")
}

func testfun2(rw *http.ResponseWriter, r *http.Request, c *Middleware) {
	fmt.Println("before===testfunc2")
	c.Next(nil, nil)

	fmt.Println("after===testfunc2")
}

func testfun3(rw *http.ResponseWriter, r *http.Request, c *Middleware) {
	fmt.Println("before===testfunc3")
	c.Next(nil, nil)
	fmt.Println("after===testfunc3")
}

var funs = []HTTPServeFunc{testfun1, testfun2, testfun3}

func TestWrapper(t *testing.T) {
	h := chain(DefaultHTTPServeFunc, funs...)
	h.Next(nil, nil)
}
