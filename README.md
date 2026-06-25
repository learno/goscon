# gosconn

## 特性
* 断线重连: [scp协议介绍](https://github.com/ejoy/goscon/blob/master/protocol.md)
* 加密： [dh64密钥交换](https://en.wikipedia.org/wiki/Diffie%E2%80%93Hellman_key_exchange)及对称流加密
* 负载均衡
* 命名服务路由
* 配置热更新
* 支持`kcp`、`tcp`和`websocket`，且可以无缝切换

## 用法

断线重连服务器端代理。

```
client <--> goscon <---> server
```

`client`和`goscon`之间使用断线重连协议，`goscon`把客户端的请求内容转发到`server`。

无论`client`因为何种原因主动或被动断开连接，`goscon`都会维持对应的`server`连接，使`server`感受不到`client`断开。

在`goscon`维持连接期间，`client`可以使用断线重连协议，无缝重用之前的连接。

若`scp.reuse_time`秒没有被重用，`goscon`断开跟`server`的连接。

编译时开启`sproto`扩展，新建连接后自动给后端发送一条`sproto`消息，宣布客户端的原始`ip`地址信息。

## build & run & test

* deps: go v1.13+

* build
```bash
# normal compile
go build -mod=vendor

# enable sproto hook & debug
# go build -tags sproto,debug -mod=vendor

```

* config

配置选项含义，请参考[config.go](https://github.com/ejoy/goscon/blob/master/config.go)

当 WebSocket 监听置于 Nginx 等反向代理之后时，`conn.RemoteAddr()` 拿到的是代理机地址。
可通过 `websocket_option.real_ip_header` 指定从哪个 HTTP 头解析真实客户端 IP，
解析到的地址会透传给上游服务（`remote_addr`）：

```yaml
websocket_option:
  # 为空表示不信任代理头，直接使用 TCP 连接地址（默认）。
  # 置于可信反向代理之后时，可设为 X-Forwarded-For 或 X-Real-IP。
  # X-Forwarded-For 形如 "client, proxy1, proxy2" 时取第一个 IP。
  real_ip_header: "X-Forwarded-For"
```

> 注意：仅在确信请求经由可信代理时才开启，否则直连客户端可伪造该头。
> Nginx 侧需配置透传，例如 `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`。

* run
```bash
./goscon -logtostderr -v 10 -config config.yaml
```

* test

- 编译测试程序

```bash
# normal compile
go build -mod=vendor ./examples/client
```

- 启动服务端

```bash
./client -startEchoServer :11248
```

- 测试 tcp

```
./client -packets 10 -concurrent 100 -rounds 100
```

- 测试 kcp

```
./client kcp
```

- 测试 websocket

```
./client websocket
```

## maintenance

可以通过默认开启的管理端口`http://localhost:6620`进行配置热更新，查看内部状态。

* 热更新配置
    - 修改配置文件
    - 访问: `http://localhost:6620/reload`
* 查看内部状态
    - 当前配置：`http://localhost:6620/config`
    - 指标: `http://localhost:6620/metrics`
    - kcp snmp: `http://localhost:6620/kcp/snmp`