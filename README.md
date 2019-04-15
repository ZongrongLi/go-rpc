# go-rpc
主要参考
https://github.com/megaredfan/rpc-demo
实现的微服务框架RPC框架




其他的实现阅读：<Br/>
https://github.com/smallnest/rpcx<Br/>
https://github.com/koding/kite<Br/>

支持服务治理的相关功能，包括：<Br/>
超时控制 <Br/>
服务注册与发现<Br/>
服务负载均衡<Br/>
限流和熔断<Br/>
身份认证<Br/>
监控和链路追踪<Br/>
健康检查，包括端到端的心跳以及注册中心对服务实例的检查<Br/>
暂时没实现idl的方式 <Br/>

系统设计<Br/>
![Image](http://176.122.170.140/wp-content/uploads/2019/04/169438cf843524b3.png)


service 是面向用户的接口，比如客户端和服务端实例的初始化和运行等等<Br/>
client和server表示客户端和服务端的实例，它们负责发出请求和返回响应<Br/>
selector 表示负载均衡，它负责决定具体要向哪个server发出请求
registery 表示注册中心，server在初始化完毕甚至是运行时都要向注册中心注册自身的相关信息，client能从注册中心查找到需要的server<Br/>
codec 表示编解码，也就是将对象和二进制数据互相转换<Br/>
protocol 表示通信协议 <Br/>
transport 表示通讯，它负责具体的网络通讯，将按照protocol组装好的二进制数据通过网络发送出去，并根据protocol指定的方式从网络读取数据<Br/>

过滤器链<Br/>
采用类似过滤器链一样的方式处理请求和响应，以此来达到对扩展开放，对修改关闭的效果。这熔断降级和限流、身份认证，鉴权，日志，链路追踪等功能在过滤器中实现。<Br/>



消息协议<Br/>
接下来设计具体的消息协议，所谓消息协议大概就是两台计算机为了互相通信而做的约定。举个例子，TCP协议约定了一个TCP数据包的具体格式，比如前2个byte表示源端口，第3和第4个byte表示目标端口，接下来是序号和确认序号等等。而在我们的RPC框架中，也需要定义自己的协议。一般来说，网络协议都分为head和body部分，head是一些元数据，是协议自身需要的数据，body则是上一层传递来的数据，只需要原封不动的接着传递下去就是了。<Br/>

-------------------------------------------------------------------------------------------------<Br/>
|2byte|1byte  |4byte       |4byte        | header length |(total length - header length - 4byte)|<Br/>
-------------------------------------------------------------------------------------------------<Br/>
|magic|version|total length|header length|     header    |                    body              |<Br/>
-------------------------------------------------------------------------------------------------<Br/>
两个byte的magic number开头，这样一来我们就可以快速的识别出非法的请求<Br/>
一个byte表示协议的版本，目前可以一律设置为0<Br/>
4个byte表示消息体剩余部分的总长度（total length）<Br/>
4个byte表示消息头的长度（header length）<Br/>
消息头（header），其长度根据前面解析出的长度（header length）决定<Br/>
消息体（body），其长度为前面解析出的总长度减去消息头所占的长度（total length - 4 - header length)<Br/>




三方组件：<Br/>
日志库：<Br/>
https://github.com/golang/glog<Br/>

序列化库：<Br/>
https://github.com/vmihailenco/msgpack<Br/>

kv客户端<Br/>
https://github.com/docker/libkv<Br/>

链路跟踪api<Br/>
github.com/opentracing/opentracing-go<Br/>

自增序列号：
https://github.com/google/uuid<Br/>


使用方法：<Br/>
·启动服务，创建客户端：<Br/>
1，服务端：<Br/>
配置服务端：每个配置项都可以自定义添加插件，自由选用<Br/>
```
servertOption := server.Option{
		ProtocolType:   protocol.Default,
		SerializeType:  protocol.SerializeTypeMsgpack,
		CompressType:   protocol.CompressTypeNone,
		TransportType:  transport.TCPTransport,
		ShutDownWait:   time.Second * 12,
		Registry:       r1,
		RegisterOption: registry.RegisterOption{"my-app"},
    
    //基于标签的路由策略
		Tags:           map[string]string{"idc": "lf"}, //只允许机房为lf的请求，客户端取到信息会自己进行转移
	}
```
启动服务：<Br/>
```
func StartServer(op *server.Option) {
	go func() {
		s, err := server.NewSGServer(op)
		if err != nil {
			glog.Error("new serializer failed", err)
			return
		}

		go s.Serve("tcp", "127.0.0.1:8888", nil)
	}()
}
```

·添加服务：<Br/>
定义服务的结构体：<Br/>
```
type TestService struct {
}
```
添加他的方法：<Br/>
```
func (t TestService) Add(ctx context.Context, req *TestRequest, res *TestResponse) error {
	res.Reply = req.A + req.B
	return nil
}
```
同时定义 request和response<Br/>
的结构<Br/>
```
type TestRequest struct {
	A int //发送的参数
	B int
}

type TestResponse struct {
	Reply int //返回的参数
}
```
在sgserver中注册这个服务<Br/>
```
s.Register(service.TestService{})
```




2客户端：<Br/>

配置：<Br/>
```
op := &client.DefaultSGOption
	op.AppKey = "my-app"
	op.RequestTimeout = time.Millisecond * 100
	op.DialTimeout = time.Millisecond * 100
  
  //心跳、降级
	op.HeartbeatInterval = time.Second
	op.HeartbeatDegradeThreshold = 5
	op.Heartbeat = true
  
  
	op.SerializeType = protocol.SerializeTypeMsgpack
	op.CompressType = protocol.CompressTypeNone
	op.TransportType = transport.TCPTransport
	op.ProtocolType = protocol.Default
  
  //容错
	op.FailMode = client.FailRetry
	op.Retries = 3
  
  //鉴权
	op.Auth = "hello01"
  
  
  //熔断
	//一秒钟失败20次 就会进入贤者模式.. 因为lastupdate时间在不断更新，熔断后继续调用有可能恢复
	op.CircuitBreakerThreshold = 20
	op.CircuitBreakerWindow = time.Second

	//基于标签的路由策略
	op.Tagged = true
	op.Tags = map[string]string{"idc": "lf"}
  r2 := libkv.NewKVRegistry(store.ZK, "my-app", "/root/lizongrong/service",
		[]string{"127.0.0.1:1181", "127.0.0.1:2181", "127.0.0.1:3181"}, 1e10, nil)
	
	op.Registry = r2
  
  //限流
        op.Wrappers = append(op.Wrappers, &client.RateLimitInterceptor{Limit: ratelimit.NewRateLimiter(10, 2)}) //一秒10个，最多有两个排队
```
创建客户端：<Br/>
```
	c := client.NewSGClient(*op)
```

使用客户端调用：<Br/>
```
c.Call(ctx, "ArithService.Add", &Testrequest, &Testresponse)
```

添加中间件：<Br/>
client接口：<Br/>
```
type Wrapper interface {
	WrapCall(option *SGOption, callFunc CallFunc) CallFunc
}
```

服务端接口：<Br/>
```
type Wrapper interface {
	WrapServe(s *SGServer, serveFunc ServeFunc) ServeFunc
	WrapHandleRequest(s *SGServer, requestFunc HandleRequestFunc) HandleRequestFunc
}
```
实现接口，并且在客户端和服务端初始化的时候或者之前加入到option中去：<Br/>
```
s.option.Wrappers = append(s.option.Wrappers, &DefaultServerWrapper{})
```

如果需要扩展接口方法，需要实现把之前的wrapper都添加上扩展的方法，并且在wrapper.go中添加函数原型<Br/>



http网关的使用方法：
将配置信息放到httpheader中：


使用post方法，路径为invoke
request参数使用约定好的序列化方法序列化后放到body中


一下为使用http包发送http请求代码：
```
func MakeRequest(req *http.Request,
	msgtype protocol.MessageType,
	comrpesstype protocol.CompressType,
	serliazetype protocol.SerializeType,
	statuscode protocol.StatusCode,
	servicename string,
	methodname string,
	err string,
	meta *map[string]interface{}) *http.Request {

	req.Header.Set(server.HEADER_SEQ, "1")
	req.Header.Set(server.HEADER_MESSAGE_TYPE, strconv.FormatUint((uint64)(msgtype), 10))
	req.Header.Set(server.HEADER_COMPRESS_TYPE, strconv.FormatUint((uint64)(comrpesstype), 10))
	req.Header.Set(server.HEADER_SERIALIZE_TYPE, strconv.FormatUint((uint64)(serliazetype), 10))
	req.Header.Set(server.HEADER_STATUS_CODE, strconv.FormatUint((uint64)(statuscode), 10))
	req.Header.Set(server.HEADER_SERVICE_NAME, servicename)
	req.Header.Set(server.HEADER_METHOD_NAME, methodname)
	req.Header.Set(server.HEADER_ERROR, err)

	metaJson, _ := json.Marshal(meta)
	req.Header.Set(server.HEADER_META_DATA, string(metaJson))
	return req
}
```

	arg := service.TesthRequest{a, b}

	data, _ := msgpack.Marshal(arg)
	body := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", "http://localhost:5080/invoke", body)
	if err != nil {
		glog.Info(err)
		return
	}
实现：<Br/>

网络通信模型：<Br/>
![Image](http://176.122.170.140/wp-content/uploads/2019/04/20190415192710.png)

用户调用方法：<Br/>
方法的参数必须为三个，而且第一个必须是context.Context类型<Br/>
第二个是服务名.方法名<Br/>
后面两个是request和response<Br/>
方法返回值必须是error类型<Br/>
客户端通过"Type.Method"的形式来引用服务方法，其中Type是方法实现类的全类名，Method就是方法名<Br/>



服务发现中心：<Br/>
使用zookeeper，可以自由选用其他 如ETCD，cousul<!-- wp:paragraph -->
<p> 定时拉取和监听数据 ，推拉结合</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p> 定时拉取服务列表并缓存本地</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p>查询时直接返回缓存</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p> 注册时在zk添加节点 注销时在zk删除节点</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p> 监听时并不监听每个服务提供者，而是监听其父级目录，有变更时再统一拉取服务提供者列表</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p>注册和注销时需要更改父级目录的内容（lastUpdate）来触发监听<br><br></p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p>使用了libkv的库来做zk的客户端，所以不能使用临时节点自动触发下线，其他注册中心也不支持临时节点，需要客户端做探活</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p>开一个heartbeat协程做探活：</p>
<!-- /wp:paragraph -->
<!-- wp:paragraph -->
<p>发送方法名为空的rpc调用请求作为探活，ptorocol里面request类型为探活类型</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>探活不受到降级，鉴权，和标签路由的拦截</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p></p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p></p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>探活的结果会触发降级</p>
<!-- /wp:paragraph -->



负载均衡：<Br/>
只有随机选择，面对本地缓存的服务列表，从中随机选择一个<Br/>

长连接及网络重连：<Br/>
为了减少频繁创建和断开网络连接的开销，我们维持了客户端到服务端的长连接，并把创建好的连接（RPCClient对象）用map缓存起来，key就是对应的服务端的标识。客户端在调用前根据负载均衡的结果检索到缓存好的RPCClient然后发起调用。当我们检索不到对应的客户端或者发现缓存的客户端已经失效时，需要重新建立连接（重新创建RPCClient对象）<Br/>


集群容错：<Br/>

客户端中配置：<Br/>
```
type FailMode byte
const (
	FailFast FailMode = iota //快速失败
	FailOver //重试其他服务器
	FailRetry //重试同一个服务器
	FailSafe //忽略失败，直接返回
)
```


客户端心跳：<Br/>

发送方法名为空的rpc调用请求作为探活，ptorocol里面request类型为探活类型<Br/>

探活不能受到降级，鉴权，和标签路由的拦截<Br/>

探活的结果会触发降级<Br/>

降级机制：<Br/>

调用里设置fileter，触发降级的时候标记相关服务为降级，filter会过滤掉，建立一个degradwraaper来实现<Br/>
设置一个计数器<Br/>
探活成功会将计数器置0，连续失败多次触发降级，如果再次成功，会触发服务标记为非降级正常工作<Br/>

鉴权：<Br/>

meta信息中带有标签，不符合规则的标签会被屏蔽，并且发送失败的response，同样使用中间件来完成<Br/>


熔断：<Br/>

调用失败的时候触发不再重试 分数量阈值x和时间阈值y，必须在时间y内失败次数够x次。才会触发熔断，每次调用之前在wapper中触发 
AllowRequest 判断是否触发熔断，触发的话就禁止请求，同样放到selector里面<Br/>

同时实现了服务端和客户端的，服务端主要用来做集群熔断，待以后实现<Br/>




限流：ratelimiter<Br/>

机制：<Br/>

开一个额外的协程，每隔一段时间往里面放一个时间戳作为token，每次判断是否响应的或者是否请求的时候从中取一个，如果已经被取光了就阻塞住等待，协程的大小决定了允许瞬间峰值的大小，客户端和服务端都有实现，选一边就行，同样基于wrapper<Br/>

链路追踪：使用开源的opentracing<Br/>

1，根据请求方法名等信息生成链路信息<Br/>
2，通过rpc metadata传递追踪信息<Br/>


基于标签的路由策略：<Br/>
用于流量转移，切断某些rpc或者某些身份的流量<Br/>

跟降级差不多，实现一个filter，放到client中供selector调用<Br/>

并且服务端和客户端在meta中打入自己的标签，不匹配的请求将会被禁止<Br/>


实现http网关：<Br/>

通过http来请求服务而不是通过rpc请求，需要将http请求转换成rpc交给自己运行，收到rpcx的启发，gateway是实现resultful的前提<Br/>

实现方法是 将原先rpc的协议头放到http中，接收到以后再将http头中的内容提取出来，合成rpc包，交给原先的接口<Br/>
