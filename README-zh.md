# lkrequest-go

[`lkrequest`](https://github.com/EZXLabs/lkrequest-rs) 的 Go 绑定 —— 一个支持字节级
**TLS / HTTP2 / HTTP3(QUIC) 指纹控制**的 Rust HTTP 客户端,适用于浏览器模拟、反检测研究和网页抓取。

[English](README.md) | [简体中文](README-zh.md)

本包在 `lkrequest-ffi` C ABI 之上封装了一层轻量、符合 Go 习惯的 API:

- `Client` —— 不可变、可共享的客户端配置(指纹、TLS、超时)
- `Session` —— 连接池、代理、重定向、重试与 cookie 状态
- `Request` —— 单所有者、发送即失效的请求构造器
- `Response` / `StreamingResponse` —— 缓冲式与流式(`io.ReadCloser`)读取

## 特性

- **浏览器指纹预设** —— 按名称使用 Chrome / Firefox / Safari 的 TLS+HTTP2 指纹,也可从 profile bundle 加载(可导出、编辑后再加载)
- **指纹随机化** —— 逐连接的 TLS 扩展顺序抖动(漂移 JA3,仍是真实浏览器);配合 `synthetic-fp` 库可合成跨层身份
- **HTTP/2 & HTTP/3(QUIC)** —— Alt-Svc 发现、请求级协议偏好、H3 头顺序、请求级 HTTP/2 优先级权重
- **双引擎** —— 默认 `purego`(无需 C 编译器,内嵌动态库)或 `lkcgo`(CGo,链接随附的动态库)
- **流式响应** —— `SendStreaming()` 返回 `io.ReadCloser`
- **Cookie 管理** —— cookie jar、带属性的 `SetCookieWithAttrs` / `GetCookiesWithAttrs` / `GetAllCookies`、请求级覆盖
- **Multipart 上传** —— 文本 + 文件字段,边界与 Chrome 一致(`----WebKitFormBoundary`)
- **重试与尝试上限** —— 固定间隔 / 指数退避(可带 jitter);连接关闭后的重放与 H2→H1 回退需显式开启;可限制建连次数与额外提交次数,并以 JSON 汇总一次请求的各次尝试
- **超时** —— 分阶段超时、总超时与响应读空闲超时,均可单独关闭
- **代理** —— 会话级 / 请求级 / 代理池 / 会话池,支持轮换、健康检查、坏代理标记;通过 `ProxyConfig` 使用静态头部凭据(`Preshared`、`PrivateToken`、`Bearer` 等)
- **MASQUE 代理(实验性)** —— 经 RFC 9298 CONNECT-UDP 代理隧道承载 QUIC,代理一跳可使用私有 CA;需要启用 `masque` 的库
- **连接预热** —— `Preconnect` / `PreconnectAsync`
- **自动解压** —— gzip / br / deflate / zstd,可通过 `AcceptEncoding` 控制
- **证书管理** —— 自定义 CA、校验开关、系统证书、ECH、TLS keylog、信任锚标识(TAI)协商、TLS 会话缓存控制
- **Header / Cookie 顺序** —— 可在客户端 / 会话 / 请求三层配置;未显式排序的 Cookie 按 Chrome 自身的顺序发送;HTTP/1.1 头名可选 Title-Case 或小写
- **自定义 DNS** —— 内置 DoH 解析器(Google / Cloudflare / Quad9)或自定义解析回调
- **SOCKS5 UDP 中继探测** —— 检测 QUIC-over-SOCKS5 所需的 UDP ASSOCIATE 支持
- **请求诊断** —— 通过 `DiagnosticsJSON()` 获取分阶段耗时(DNS / TCP / TLS / TTFB)
- **Context 支持** —— 可取消(`SendWithContext`)与异步(`SendAsync`)发送
- **类型化错误** —— `LkError` 暴露错误码、阶段、可重试标志与诊断信息

## 安装

```bash
go get github.com/EZXLabs/lkrequest-go/lkrequest@latest
```

```go
import "github.com/EZXLabs/lkrequest-go/lkrequest"
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

HTTP/1.1 请求的头名默认为 Title-Case(`Content-Type`)。`SetHTTP1TitleCaseHeaders(false)` 改为小写
(`content-type`),`Host` 除外;默认头和 `content-length` 等自动生成的头同样适用。两种模式都不保留
`AddHeader` 传入的原始拼写。HTTP/2 与 HTTP/3 始终使用小写头名。

```go
client, err := lkrequest.NewClientBuilder().
	SetHTTP1TitleCaseHeaders(false). // 仅影响 HTTP/1.1
	Build()
```

### 超时、重放与会话缓存

每个超时都可以关闭。总超时默认 120 秒,覆盖一次请求从建连到读完响应体的全过程(流式响应体同样);
`SetTimeoutDNS`、`SetTimeoutTCPConnect`、`SetTimeoutTLSHandshake`、`SetTimeoutTTFB`、`SetTimeoutQUICConnect`
限制各个阶段,`SetTimeoutReadIdle`(默认关闭)限制每次读取响应体时等待数据的时长。每个都有对应的
`DisableTimeout…` 用来关闭,之后再调用 setter 即重新启用;`0` 表示立即超时,而不是关闭。

```go
client, err := lkrequest.NewClientBuilder().
	DisableTimeoutTotal().     // 长时间的流式响应……
	SetTimeoutReadIdle(45000). // ……但不允许 45 秒没有数据
	Build()
```

复用的连接在收到响应前被关闭时,请求不会重发;HTTP/2 也不会回退到 HTTP/1.1。需要时在客户端上开启
`SetRetryOnConnectionClose(true)` / `SetFallbackH2ToH1(true)`。会话级重试策略(`SetRetryFixed`、
`SetRetryExponential`)与之分开配置。

TLS 与 QUIC 会话默认缓存以便复用。`SetTLSSessionCacheEnabled(false)` 让每条新连接都完整握手,
`SetTLSSessionCacheTTL(ms)` 限制会话可复用的时长(`0` 表示不缓存),`ResetTLSSessionCacheTTL()` 恢复默认时长。

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

### 尝试上限与尝试摘要

两个可选上限约束单个请求的开销。`SetConnectAttemptLimit` 限制它可以发起的 TCP / QUIC 建连次数:失败的建连
与竞速落败的连接都计入,复用连接池中的连接不计入,超出上限的请求以 `ErrResourceLimitExceeded` 失败。
`SetExtraHTTPAttemptLimit` 限制它在故障恢复时可以重发的次数,首次提交和重定向不计入。设在 `ClientBuilder`
上时作用于该客户端的每个请求,按请求计数,不是共享配额;`SessionBuilder` 与 `Request` 只能收紧继承来的上限,
更宽松的值由 `Build()` / `Send()` 报错。两者都不会开启重试,也不设超时。

```go
client, err := lkrequest.NewClientBuilder().
	SetRetryOnConnectionClose(true).
	SetExtraHTTPAttemptLimit(2).
	Build()
// ……
resp, err := session.Get(url).
	SetConnectAttemptLimit(1).
	EnableAttemptSummary(true).
	Send()
fmt.Println(resp.AttemptSummaryJSON())
// {"connect_initiations":1,"http_submissions":1,"extra_http_submissions":0,"outcome":"succeeded",…}
```

构造器、`Client`、`Session`、`Request` 上的 `ConnectAttemptLimit()` / `ExtraHTTPAttemptLimit()` 返回
`AttemptLimit`:本层设置的值、实际生效的值及其来源层级 `Scope`。`Response`、`StreamingResponse`、`LkError`
的 `AttemptSummaryJSON()` 返回由 `EnableAttemptSummary(true)` 开启的摘要 —— 提交次数、建连次数、恢复次数、
生效的上限,失败时还有 `denial_reason`;未开启时返回 `""`。

### Cookie

```go
session.SetCookie("https://example.com", "token", "abc123")
session.SetCookieWithAttrs("https://example.com", "sid", "xyz", lkrequest.CookieAttrs{
	Path: "/api", Domain: "example.com", Secure: true, HTTPOnly: true,
})
val, _ := session.GetCookie("https://example.com", "token")
jar, _ := session.GetCookies("https://example.com") // []Cookie;GetCookiesJSON 返回原始 JSON

// 带属性读取:domain、path、Secure、HttpOnly、SameSite 与过期时间(会话 cookie 为 nil)
scoped, _ := session.GetCookiesWithAttrs("https://example.com/api") // []CookieInfo,path 长的在前
all, _ := session.GetAllCookies()                                   // 跨所有域名,按创建时间从早到晚 —— 适合持久化

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

配置了代理的池不会回退为直连。所有代理都在冷却时,`Acquire` 会等待其中一个恢复;所有
代理都被永久移除后,它会一直等下去。用 `AcquireAsync(ctx)` 限定等待时间,`ctx` 结束时
它会放弃等待。只有不含任何代理的池才会发放直连租约,其 `URL()` 为 `""`。调用
`SessionPool.MarkBad(g)` 之后再关闭 `g`,其会话会被丢弃,而不是放回空闲池。

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

`proxy.Identity()` 才是 `ProxyPool.MarkBad` 需要的键,代理 URL 不是。`MarkBad` 返回时标记即已生效。

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

单个请求可以覆盖其 HTTP/2 HEADERS 帧旧式优先级字段里的权重,用于浏览器给出的权重与
preset 默认值不同的请求。它不改动 `priority` 请求头;HTTP/1.1 与 HTTP/3 会忽略它。

```go
resp, err := session.Get("https://example.com/").
	SetH2PriorityWeight(21). // 线上值,0-255(即权重 22)
	Send()
```

### 自定义 profile 与 profile bundle

指纹也可以来自数据,而不只是 preset 名。`ExportPresetJSON` 把内置 preset 导出为完整的 profile bundle:
一份 JSON 文档(`"format": "lkrequest-profile-bundle"`,`"schema_version": 2`),包含 preset 的全部层,
HTTP/2 运行时行为和协议策略也在其中;`SetPresetJSON` 或 `SetPresetFile` 可以把它(改过或没改过)重新加载。

```go
bundle, err := lkrequest.ExportPresetJSON("chrome_157") // 可编辑、保存、分发
client, err := lkrequest.NewClientBuilder().
	SetPresetJSON(bundle). // 或 SetPresetFile("chrome-157.json")
	Build()
```

也可以在 preset 或 bundle 之上,用原生 profile JSON(格式即 `FingerprintInfoJSON()` 中对应层的格式)单独
替换某一层:`SetTLSProfileJSON`、`SetH2ProfileJSON`(只含数据字段,preset 的 HTTP/2 运行时行为会退回基线)、
`SetQUICTLSProfileJSON`,以及让 QUIC 改用 TCP TLS profile 的 `ClearQUICFingerprint`。它们优先于 preset
或 bundle,与调用先后无关;传空字符串则把该层交还给 preset 或 bundle。

- 输入上限 1 MiB 并经过校验,库拒绝的输入由 `Build()` 报告。bundle 的错误是 `LkError`(`ErrInvalidConfig`、
  `ErrResourceLimitExceeded` 或 `ErrIO`),其 `DiagnosticsJSON()` 中的 `preset_error` 给出稳定的 `code`、
  JSON Pointer 形式的 `path` 和 `message`。
- 在 `SetPresetJSON` 之后调用 `SetPreset` 会替换 bundle,`SetPresetJSON("")` 则移除它。bundle 不能改变代理、
  CA 或证书校验设置,加载时除了读取你指定的文件外不做任何 I/O。
- `FingerprintInfoJSON()` 用 `quic_tls` 报告 QUIC 的 TLS profile;QUIC 使用 TCP 的 profile 时为 `null`。
- 加载 bundle 只是还原其中的字段,并不意味着指纹在线上就与浏览器一致。

### 信任锚标识(Trust Anchor Identifiers)

Chrome 预设会在 TLS `trust_anchors` 扩展(draft-ietf-tls-trust-anchor-ids)里声明信任锚标识,即客户端
接受的根证书的 ID。默认情况下,客户端像浏览器一样发送预设自带的列表,此外不做任何处理。
`SetTrustAnchorPolicy` 可以改为发送客户端已信任的根所对应的 ID,并在服务端列出它能提供的根之后、握手
仍失败时,让请求重连一次。这些都不会改变信任哪些证书。

```go
client, err := lkrequest.NewClientBuilder().
	SetPreset("chrome_157").
	SetTrustAnchorPolicy(lkrequest.TrustAnchorPolicy{
		Advertisement: lkrequest.TrustAnchorAdvertiseTrusted,
		Recovery:      lkrequest.TrustAnchorRecoveryOnce,
	}).
	Build()

// 私有根:照常信任它,再绑定其部署方分配的 ID。
identity, err := lkrequest.TrustAnchorIdentity(rootDER)
client, err = lkrequest.NewClientBuilder().
	AddCACertMemory(rootDER).
	AddTrustAnchorBinding(identity, "81fd5901").
	Build()
```

- 无法满足的策略由 `Build()` 以 `ErrInvalidConfig` 报告:重连需要开启证书校验,且不能与
  `SetFallbackProxyToDirect` 同时使用;零值以外的策略要求预设的 TLS profile 都带这个扩展(Chrome 带,
  Firefox 与 Safari 不带),也不能使用合成 TLS 随机化;绑定的根必须是客户端已信任的根。
- 响应或 `LkError` 的 `DiagnosticsJSON()` 里有 `trust_anchor` 对象,描述该请求新建的 TLS 连接:请求与
  可用的 ID、`verified_anchor`(校验证书链所用根的 `TrustAnchorIdentity`)、`authenticated`、`recovered`,
  以及失败的首次尝试 `previous_attempt`。复用连接池中连接的请求,该字段为 `null`。
- 重连也失败时,请求以不可重试的错误结束。服务端列出其信任锚之后的任何握手失败同样不可重试,与是否开启
  重连无关。

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
| `cgo` | `lkcgo` | `#include "lkrequest.h"` 并链接 `lib/<platform>/` 中的动态库 | 是 |

```bash
go build ./lkrequest/...              # purego(默认)
go build -tags lkcgo ./lkrequest/...  # cgo
```

cgo 编译的程序启动时加载该动态库:darwin 上通过指向构建时所用模块中 `lib/darwin_arm64` 的 rpath 找到它;
linux 上需把 `lib/linux_amd64` 加入 `LD_LIBRARY_PATH`(或把库安装到系统路径);windows 上把
`lkrequest_ffi.dll` 放在可执行文件旁或 `PATH` 中。

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
| 顶层辅助函数 | `Get`、`PostJSON`、`NewClient`、`NewDefaultClient`、`NewSession`、`NewRequest`、`NewMultipart`、`ParseProxyConfig`、`NewMasqueConfig`、`TrustAnchorIdentity`、`ExportPresetJSON` |
| HTTP 动词 | `Session.Get` / `Post` / `Put` / `Delete` / `Head` / `Patch` / `Options` / `Request` —— `SessionPoolGuard` 同样具备 |
| 构造器 | `NewClientBuilder`、`NewSessionBuilder`、`NewProxyPoolBuilder`、`NewSessionPoolBuilder` |
| 核心类型 | `Client`、`Session`、`Request`、`Response`、`StreamingResponse`、`ProxyPool`、`SessionPool`、`ProxyConfig`、`MasqueConfig`、`Multipart`、`Cookie`、`CookieInfo`、`Redirect`、`TrustAnchorPolicy`、`AttemptLimit` |
| 内省 | `ABIVersion`、`LibraryVersion`、`FeatureSupported`、`ListPresetsJSON`、`GetPresetDetailJSON` |
| 日志 | `InitLog`、`InitLogCallback` |
| 枚举 | `RandomizeMode`、`FingerprintLayers`、`AcceptEncoding`、`RotationStrategy`、`HttpVersion`、`PreferredHTTPVersion`、`DnsConfig`、`LogLevel`、`ProxyAuthHeader`、`TrustAnchorAdvertisement`、`TrustAnchorRecovery`、`AttemptLimitScope` |

完整 API 文档:`go doc github.com/EZXLabs/lkrequest-go/lkrequest`。

## 说明

- `Client` 和 `Session` 可安全地跨 goroutine 共享;`Request` 发送即失效(复用请用 `Clone()`)。
- `StreamingResponse` 实现了 `io.ReadCloser`。
- 未调用 `SetPreset` 的 `ClientBuilder`(以及 `NewDefaultClient()`)使用库的默认预设 —— 目前是 `chrome_155`,头顺序也随之生效。需要其他指纹时调用 `SetPreset`,需要固定顺序时调用 `AddHeaderOrder` / `AddCookieOrder`。
- QUIC/H3、自定义 DNS 解析器、SOCKS5 UDP 探测、协议版本偏好/协商、`ProxyConfig` / `MasqueConfig`、cookie 属性读取、请求级 H2 优先级权重、HTTP/1.1 头名大小写、信任锚标识、自定义 profile 与 profile bundle、读空闲超时与 `DisableTimeout…` 系列、TLS 会话缓存设置、尝试上限与尝试摘要、以及合成档 `SetRandomize` 需要库导出对应符号。在较旧的内嵌库上调用会返回 `lk: ... not supported by the loaded lkrequest library` 错误(构造器会延迟到 `Build()` 报告);包本身仍能正常加载。
- `FeatureSupported(name)` 可查询已加载的库是否支持某项功能,名称如 `preset-bundle-v2`、`read-idle-timeout`、`optional-total-timeout`、`optional-phase-timeouts`、`tls-session-cache`、`connect-attempt-limit`、`extra-http-attempt-limit`、`attempt-summary`。
