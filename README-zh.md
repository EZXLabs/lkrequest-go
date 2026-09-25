# lkrequest-go

[`lkrequest`](https://github.com/EZ-XLabs/lkrequest) 的 Go 绑定 —— 一个支持字节级
**TLS / HTTP2 / HTTP3(QUIC) 指纹控制**的 Rust HTTP 客户端,适用于浏览器模拟、反检测研究和网页抓取。

[English](README.md) | [简体中文](README-zh.md)

本包在 `lkrequest-ffi` C ABI 之上封装了一层轻量、符合 Go 习惯的 API:

- `Client` —— 不可变、可共享的客户端配置(指纹、TLS、超时)
- `Session` —— 连接池、代理、重定向、重试与 cookie 状态
- `Request` —— 单所有者、发送即失效的请求构造器
- `Response` / `StreamingResponse` —— 缓冲式与流式(`io.ReadCloser`)读取

## 特性

- **浏览器指纹预设** —— 按名称使用 Chrome / Firefox / Safari 的 TLS+HTTP2 指纹
- **指纹随机化** —— 逐连接的 TLS 扩展顺序抖动(漂移 JA3,仍是真实浏览器);配合 `synthetic-fp` 库可合成跨层身份
- **HTTP/2 & HTTP/3(QUIC)** —— Alt-Svc 发现、请求级协议偏好、H3 头顺序
- **双引擎** —— 默认 `purego`(无需 C 编译器,内嵌动态库)或 `lkcgo`(CGo 静态链接)
- **流式响应** —— `SendStreaming()` 返回 `io.ReadCloser`
- **Cookie 管理** —— cookie jar、带属性的 `SetCookieWithAttrs`、请求级覆盖
- **Multipart 上传** —— 文本 + 文件字段,边界与 Chrome 一致(`----WebKitFormBoundary`)
- **重试策略** —— 固定间隔 / 指数退避(可带 jitter)
- **代理** —— 会话级 / 请求级 / 代理池 / 会话池,支持轮换、健康检查、坏代理标记;通过 `ProxyConfig` 使用静态头部凭据(`Preshared`、`PrivateToken`、`Bearer` 等)
- **MASQUE 代理(实验性)** —— 经 RFC 9298 CONNECT-UDP 代理隧道承载 QUIC,代理一跳可使用私有 CA;需要启用 `masque` 的库
- **连接预热** —— `Preconnect` / `PreconnectAsync`
- **自动解压** —— gzip / br / deflate / zstd,可通过 `AcceptEncoding` 控制
- **证书管理** —— 自定义 CA、校验开关、系统证书、ECH、TLS keylog
- **Header / Cookie 顺序** —— 可在客户端 / 会话 / 请求三层配置;未显式排序的 Cookie 按 Chrome 自身的顺序发送
- **自定义 DNS** —— 内置 DoH 解析器(Google / Cloudflare / Quad9)或自定义解析回调
- **SOCKS5 UDP 中继探测** —— 检测 QUIC-over-SOCKS5 所需的 UDP ASSOCIATE 支持
- **请求诊断** —— 通过 `DiagnosticsJSON()` 获取分阶段耗时(DNS / TCP / TLS / TTFB)
- **Context 支持** —— 可取消(`SendWithContext`)与异步(`SendAsync`)发送
- **类型化错误** —— `LkError` 暴露错误码、阶段、可重试标志与诊断信息

## 安装

```bash
go get github.com/EZ-XLabs/lkrequest-go/lkrequest@latest
```

```go
import "github.com/EZ-XLabs/lkrequest-go/lkrequest"
```

## 快速开始

顶层辅助函数使用一个进程内共享的默认客户端:

```go
resp, err := lkrequest.Get("https://httpbin.org/get")
if err != nil {
	log.Fatal(err)
}
defer resp.Close()

fmt.Println(resp.StatusCode())
fmt.Println(resp.String())
```

完整的 `Client → Session → Request → Response` 流程(所有句柄都需显式关闭):

```go
client, err := lkrequest.NewClient("chrome_150") // 或 NewDefaultClient()
if err != nil {
	log.Fatal(err)
}
defer client.Close()

session, err := lkrequest.NewSession(client)
if err != nil {
	log.Fatal(err)
}
defer session.Close()

resp, err := session.Get("https://httpbin.org/get").Send()
if err != nil {
	log.Fatal(err)
}
defer resp.Close()

fmt.Println(resp.StatusCode(), resp.Header("content-type"))
```

## 用法

### 构造与发送请求

`Session` 为每个 HTTP 动词提供了便捷方法,均返回 `*Request` 供继续链式设置。
构造失败会随请求一起传递,由 `Send()` 统一报告,不需要单独的错误检查。

```go
resp, err := session.Get("https://httpbin.org/get").Send()

// Get / Post / Put / Delete / Head / Patch / Options,
// 其余方法用 Request(method, url)。
resp, err = session.Request("PROPFIND", "https://example.com/dav").Send()
```

`NewRequest(session, method, url)` 是同样的构造,但错误立即返回。
`SessionPoolGuard` 拥有同一套便捷方法。

`Request` 是**发送即失效**的 —— 链式设置后发送,需复用请通过 `Clone()`。

```go
resp, err := session.Post("https://httpbin.org/post").
	AddHeader("accept", "application/json").
	AddQuery("page", "1").
	SetBearerAuth("my-token").                     // 或 SetBasicAuth(user, pass)
	SetJSON(map[string]string{"hello": "world"}).  // 用 encoding/json 序列化
	Send()                                         // SetJSONBody 则接收已编码好的文档
```

可取消与异步发送:

```go
resp, err := req.SendWithContext(ctx)          // 响应 context 取消
respCh, errCh := req.SendAsync(ctx)            // 在 goroutine 中运行,返回 channel
```

### 自定义客户端

`ClientBuilder` 是链式构造器,setter 的错误延迟到 `Build()` 时报告。

```go
client, err := lkrequest.NewClientBuilder().
	SetPreset("chrome_144").
	SetVerify(true).
	SetUseNativeCerts(true).
	AddDefaultHeader("Accept-Language", "zh-CN").
	SetTimeoutTotal(30000).             // 毫秒
	SetMaxConnectionsPerSession(16).
	SetFallbackH2ToH1(true).
	Build()
```

### 会话配置与重试

```go
session, err := lkrequest.NewSessionBuilder(client).
	SetProxy("socks5://user:pass@host:1080").
	SetMaxRedirects(5).
	SetHTTP2Only().
	SetDefaultAcceptEncoding(lkrequest.AcceptEncodingGzip | lkrequest.AcceptEncodingBr).
	SetRetryExponential(3, 50, 500, true). // maxRetries, baseMs, maxMs, jitter
	Build()

// 改用固定间隔:SetRetryFixed(2, 100)  // maxRetries, intervalMs
```

### Cookie

```go
session.SetCookie("https://example.com", "token", "abc123")
session.SetCookieWithAttrs("https://example.com", "sid", "xyz", lkrequest.CookieAttrs{
	Path: "/api", Domain: "example.com", Secure: true, HTTPOnly: true,
})
val, _ := session.GetCookie("https://example.com", "token")
jar, _ := session.GetCookies("https://example.com") // []Cookie;GetCookiesJSON 返回原始 JSON

req.SetCookieOverride("token", "fresh-value") // 请求级覆盖 jar 中的同名 cookie
```

### Multipart 上传

```go
mp := lkrequest.NewMultipart().
	AddText("title", "report").
	AddFile("document", "report.pdf", "application/pdf", pdfBytes)
req.SetMultipart(mp) // 转移所有权,该请求之后不能再 Clone
```

### 流式响应

```go
stream, err := req.SendStreaming()
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

body, err := io.ReadAll(stream) // StreamingResponse 实现了 io.ReadCloser
```

### 代理池与会话池

```go
// 代理池 —— 轮换 URL,标记坏代理
pool, err := lkrequest.NewProxyPoolBuilder().
	AddProxies([]string{"socks5://p1:1080", "http://p2:8080"}).
	SetRotation(lkrequest.RotationRoundRobin).
	Build()
guard, _ := pool.Acquire()
defer guard.Close()
proxyURL := guard.URL()

// 会话池 —— 每个 session 绑定不同的代理
sp, err := lkrequest.NewSessionPoolBuilder(client).
	AddProxy("http://p1:8080").
	SetMaxSessions(8).
	Build()
g, _ := sp.Acquire()
defer g.Close()
poolReq, _ := g.NewRequest("GET", "https://example.com")
```

### 用户名/密码以外的代理凭据

代理 URL 只能携带用户名和密码。`ProxyConfig` 还能设置静态 HTTP 凭据,以及承载它的请求头:

```go
proxy, err := lkrequest.ParseProxyConfig("https://proxy.example:443")
if err != nil {
	log.Fatal(err)
}
defer proxy.Close() // 构造器会复制配置

proxy.SetHTTPAuth("Preshared", "<psk>"). // 发送为 "Preshared <psk>"
	SetAuthHeader(lkrequest.ProxyAuthHeaderAuthorization) // 默认:Proxy-Authorization

session, err := lkrequest.NewSessionBuilder(client).SetProxyConfig(proxy).Build()
// 或放入代理池:lkrequest.NewProxyPoolBuilder().AddProxyConfig(proxy)
```

`proxy.Identity()` 才是 `ProxyPool.MarkBad` 需要的键,代理 URL 不是。标记是异步生效的,`MarkBad` 返回后稍后才会落地。

### MASQUE(实验性)

`masque://` 代理把到源站的 QUIC 连接放进 CONNECT-UDP 隧道承载。`MasqueConfig` 配置到代理的这一跳;
它的信任设置与客户端的 `SetVerify` 和 CA 证书相互独立。

```go
masque, err := lkrequest.NewMasqueConfig() // 库不支持 MASQUE 时返回错误
if err != nil {
	log.Fatal(err)
}
defer masque.Close()
masque.AddCACertsPEM(caPEM). // 代理一跳使用的私有 CA
	SetUseNativeCerts(false)

client, _ := lkrequest.NewClientBuilder().SetMasqueConfig(masque).Build()
proxy, _ := lkrequest.ParseProxyConfig("masque://proxy.example") // 端口默认 443
session, _ := lkrequest.NewSessionBuilder(client).SetHTTP3Only().SetProxyConfig(proxy).Build()
```

- MASQUE 路由只承载 HTTP/3。经它发出的 HTTP/1.1 和 HTTP/2 请求会直接失败,而不是改为直连,即使开启了 `SetFallbackProxyToDirect(true)`。
- 代理池 / 会话池在健康检查中会跳过 `masque://` 代理,除非同时设置 `SetMasqueTunnelProbe(masque)` 与 `SetHealthCheck`。
- 内嵌库构建时未启用 `masque` feature,因此使用内嵌库时 `NewMasqueConfig` 会返回错误,`masque://` URL 也会被拒绝。

### 浏览器指纹与随机化

```go
presets, _ := lkrequest.ListPresetsJSON()            // 枚举可用预设
detail, _ := lkrequest.GetPresetDetailJSON("chrome_150")

// 档位 1 —— 逐连接 JA3 抖动,始终可用,仍是真实浏览器
client, _ := lkrequest.NewClientBuilder().
	SetRandomize(lkrequest.RandomizeExtensionOrder, lkrequest.FingerprintLayersAll).
	Build()
```

`RandomizeRecombine` / `RandomizeFull` 会合成非浏览器指纹(用于黑名单类目标),需要用
`synthetic-fp` feature 构建的库,否则 `SetRandomize` 返回错误。

### QUIC / HTTP3

```go
if lkrequest.FeatureSupported("quic-h3") {
	session, _ := lkrequest.NewSessionBuilder(client).
		SetHTTP3WithFallback(). // 优先 H3,可回退到 H2
		Build()
}
```

### 自定义 DNS

```go
client, _ := lkrequest.NewClientBuilder().
	SetDNS(lkrequest.DnsCloudflareHTTPS). // 内置 DoH;或 SetDNSCustom("1.1.1.1:53")
	Build()
```

### 日志

```go
lkrequest.InitLog("info", "/tmp/lkrequest.log") // 级别字符串,日志文件

lkrequest.InitLogCallback(func(level lkrequest.LogLevel, target, msg string) {
	log.Printf("[%v] %s: %s", level, target, msg)
}, lkrequest.LogLevelInfo)
```

### 错误处理

```go
resp, err := req.Send()
if err != nil {
	var lkErr *lkrequest.LkError
	if errors.As(err, &lkErr) {
		fmt.Println(lkErr.Code(), lkErr.Phase(), lkErr.IsRetryable())
	}
	log.Fatal(err)
}
defer resp.Close()

if err := resp.ErrorForStatus(); err != nil {
	// 4xx / 5xx
}
```

## 双引擎

两种可互换的绑定引擎填充同一套 `ffi_lk_*` 函数指针:

| 引擎 | build tag | 如何加载库 | 需要 C 编译器 |
|---|---|---|---|
| `purego`(默认) | `!lkcgo` | 把内嵌动态库解压到临时目录后 `dlopen` | 否 |
| `cgo` | `lkcgo` | `#include "lkrequest.h"` 并静态链接 `lib/<platform>/` | 是 |

```bash
go build ./lkrequest/...              # purego(默认)
go build -tags lkcgo ./lkrequest/...  # cgo
```

## 平台支持

| 平台 | 状态 |
|---|---|
| linux / amd64 | ✅ 已提供(内嵌库) |
| windows / amd64 | ✅ 已提供 |
| darwin / arm64 | ✅ 已提供(自 0.2.1 起) |

## 环境要求

- **Go 1.21+**。

## API 概览

| 分组 | 符号 |
|---|---|
| 顶层辅助函数 | `Get`、`PostJSON`、`NewClient`、`NewDefaultClient`、`NewSession`、`NewRequest`、`NewMultipart`、`ParseProxyConfig`、`NewMasqueConfig` |
| HTTP 动词 | `Session.Get` / `Post` / `Put` / `Delete` / `Head` / `Patch` / `Options` / `Request` —— `SessionPoolGuard` 同样具备 |
| 构造器 | `NewClientBuilder`、`NewSessionBuilder`、`NewProxyPoolBuilder`、`NewSessionPoolBuilder` |
| 核心类型 | `Client`、`Session`、`Request`、`Response`、`StreamingResponse`、`ProxyPool`、`SessionPool`、`ProxyConfig`、`MasqueConfig`、`Multipart`、`Cookie`、`Redirect` |
| 内省 | `ABIVersion`、`LibraryVersion`、`FeatureSupported`、`ListPresetsJSON`、`GetPresetDetailJSON` |
| 日志 | `InitLog`、`InitLogCallback` |
| 枚举 | `RandomizeMode`、`FingerprintLayers`、`AcceptEncoding`、`RotationStrategy`、`HttpVersion`、`PreferredHTTPVersion`、`DnsConfig`、`LogLevel`、`ProxyAuthHeader` |

完整 API 文档:`go doc github.com/EZ-XLabs/lkrequest-go/lkrequest`。

## 说明

- `Client` 和 `Session` 可安全地跨 goroutine 共享;`Request` 发送即失效(复用请用 `Clone()`)。
- `StreamingResponse` 实现了 `io.ReadCloser`。
- `ClientBuilder.Build()` **不再**隐式套用 Chrome 头顺序预设;需要特定顺序时请设置指纹,或调用 `AddHeaderOrder` / `AddCookieOrder`。
- QUIC/H3、自定义 DNS 解析器、SOCKS5 UDP 探测、协议版本偏好/协商、`ProxyConfig` / `MasqueConfig`、以及合成档 `SetRandomize` 需要库导出对应符号。在较旧的内嵌库上调用会返回 `lk: ... not supported by the loaded lkrequest library` 错误(构造器会延迟到 `Build()` 报告);包本身仍能正常加载。
