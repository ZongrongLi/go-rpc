package service

type TestRequest struct {
	A int //发送的参数
	B int
}

type TestResponse struct {
	Reply int //返回的参数
}

type TestService struct {
}

func (t TestService) Add(req *TestRequest, res *TestResponse) {
	res.Reply = req.A + req.B
}
