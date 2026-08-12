package route

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// probeSpec 是 fake probe 对单个 host 的配置。
type probeSpec struct {
	data    map[string]any
	latency time.Duration
	err     error
	delay   time.Duration // 可选阻塞延迟，用于并发/取消测试
}

// recorder 记录 probe 调用，供断言探测顺序与去重。
type recorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *recorder) add(host string) {
	r.mu.Lock()
	r.calls = append(r.calls, host)
	r.mu.Unlock()
}

func (r *recorder) hosts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.calls...)
}

func (r *recorder) count(host string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, h := range r.calls {
		if h == host {
			n++
		}
	}
	return n
}

// fakeProbe 构造记录调用、按配置返回的可控 probe。onRequestStart 立即触发（测试默认假定
// 请求紧接构造开始），可被探测体覆盖以模拟慢构造。
func fakeProbe(specs map[string]probeSpec, rec *recorder) Probe {
	return func(ctx context.Context, host string, onRequestStart func()) (time.Duration, map[string]any, error) {
		rec.add(host)
		spec, ok := specs[host]
		if !ok {
			return 0, nil, errors.New("unexpected host " + host)
		}
		if spec.delay > 0 {
			select {
			case <-ctx.Done():
				return 0, nil, ctx.Err()
			case <-time.After(spec.delay):
			}
		}
		if onRequestStart != nil {
			onRequestStart()
		}
		if spec.err != nil {
			return spec.latency, nil, spec.err
		}
		return spec.latency, spec.data, nil
	}
}

// startupWithDomains 用真实派生 key/IV 加密一份包含给定 apiDomains 的启动 payload。
func startupWithDomains(domains ...string) map[string]any {
	payload, err := json.Marshal(map[string]any{"apiDomains": domains})
	if err != nil {
		panic(err)
	}
	padLen := aes.BlockSize - len(payload)%aes.BlockSize
	return map[string]any{"backup_domains_data": encryptForTest(payload, padLen)}
}

func TestSelectPreferredSuccessSkipsBootstraps(t *testing.T) {
	const preferred = "https://apidd.spthgb.com"
	rec := &recorder{}
	probe := fakeProbe(map[string]probeSpec{
		preferred: {latency: 30 * time.Millisecond, data: map[string]any{"ok": true}},
	}, rec)
	result, err := Select(context.Background(), SelectorOptions{PreferredHost: preferred}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != preferred || !result.ReusedPreferred {
		t.Fatalf("result = %+v, want host %s reused", result, preferred)
	}
	if result.Latency != 30*time.Millisecond {
		t.Fatalf("result latency = %v, want 30ms", result.Latency)
	}
	if got := rec.hosts(); len(got) != 1 || got[0] != preferred {
		t.Fatalf("probed hosts = %v, want only %s", got, preferred)
	}
}

func TestSelectPreferredFailureFallsThrough(t *testing.T) {
	const preferred = "https://preferred.example"
	rec := &recorder{}
	probe := fakeProbe(map[string]probeSpec{
		preferred:                   {err: errors.New("preferred offline")},
		model.HostMirror:            {latency: 10 * time.Millisecond, data: map[string]any{}},
		"https://apidd.spthgb.com":  {latency: 20 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {latency: 30 * time.Millisecond, data: map[string]any{}},
		model.HostMain:              {latency: 40 * time.Millisecond, data: map[string]any{}},
	}, rec)
	result, err := Select(context.Background(), SelectorOptions{PreferredHost: preferred}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.ReusedPreferred || result.Host != model.HostMirror {
		t.Fatalf("result = %+v, want fastest bootstrap %s", result, model.HostMirror)
	}
	if rec.count(preferred) != 1 {
		t.Fatalf("preferred probed %d times, want 1", rec.count(preferred))
	}
}

func TestSelectRejectsInvalidPreferredURL(t *testing.T) {
	rec := &recorder{}
	probe := fakeProbe(nil, rec)
	_, err := Select(context.Background(), SelectorOptions{PreferredHost: "not-a-url"}, probe)
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("error = %v, want ErrInvalidURL", err)
	}
	if len(rec.hosts()) != 0 {
		t.Fatalf("probe called for invalid preferred host: %v", rec.hosts())
	}
}

func TestSelectPicksFastestDynamicCandidate(t *testing.T) {
	fast := "https://fast.example"
	slow := "https://slow.example"
	rec := &recorder{}
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 200 * time.Millisecond, data: startupWithDomains(fast, slow)},
		"https://apidd.spthgb.com":  {latency: 210 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {latency: 220 * time.Millisecond, data: map[string]any{}},
		model.HostMain:              {latency: 230 * time.Millisecond, data: map[string]any{}},
		fast:                        {latency: 5 * time.Millisecond, data: map[string]any{}},
		slow:                        {latency: 50 * time.Millisecond, data: map[string]any{}},
	}, rec)
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != fast || result.Latency != 5*time.Millisecond {
		t.Fatalf("result = %+v, want %s at 5ms", result, fast)
	}
	if rec.count(fast) != 1 || rec.count(slow) != 1 {
		t.Fatalf("dynamic candidates probed more than once: %v", rec.hosts())
	}
}

func TestSelectTieBreaksByDynamicOrder(t *testing.T) {
	a := "https://a.example"
	b := "https://b.example"
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 100 * time.Millisecond, data: startupWithDomains(a, b)},
		"https://apidd.spthgb.com":  {latency: 100 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {latency: 100 * time.Millisecond, data: map[string]any{}},
		model.HostMain:              {latency: 100 * time.Millisecond, data: map[string]any{}},
		a:                           {latency: 9 * time.Millisecond, data: map[string]any{}},
		b:                           {latency: 9 * time.Millisecond, data: map[string]any{}},
	}, &recorder{})
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != a {
		t.Fatalf("result host = %s, want %s (dynamic response order)", result.Host, a)
	}
}

func TestSelectTieBreakDynamicBeforeBootstrap(t *testing.T) {
	dyn := "https://dyn.example"
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 100 * time.Millisecond, data: startupWithDomains(dyn)},
		"https://apidd.spthgb.com":  {latency: 7 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {latency: 7 * time.Millisecond, data: map[string]any{}},
		model.HostMain:              {latency: 7 * time.Millisecond, data: map[string]any{}},
		dyn:                         {latency: 7 * time.Millisecond, data: map[string]any{}},
	}, &recorder{})
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	// 动态候选与成功 bootstrap 同耗时时，动态响应顺序优先。
	if result.Host != dyn {
		t.Fatalf("result host = %s, want dynamic candidate %s", result.Host, dyn)
	}
}

func TestSelectTieBreakBootstrapOrderWhenNoDynamicSource(t *testing.T) {
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 100 * time.Millisecond, data: map[string]any{}},
		"https://apidd.spthgb.com":  {latency: 100 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {latency: 100 * time.Millisecond, data: map[string]any{}},
		model.HostMain:              {latency: 100 * time.Millisecond, data: map[string]any{}},
	}, &recorder{})
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != model.HostMirror {
		t.Fatalf("result host = %s, want %s (fixed bootstrap order)", result.Host, model.HostMirror)
	}
}

func TestSelectBootstrapFallbackWhenDynamicConfigMissing(t *testing.T) {
	// 所有 bootstrap 成功但都没有 backup_domains_data；已成功 bootstrap 仍可用。
	fast := "https://apidd.spthgb.com"
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 90 * time.Millisecond, data: map[string]any{"settings": "x"}},
		fast:                        {latency: 10 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {latency: 80 * time.Millisecond, data: map[string]any{}},
		model.HostMain:              {latency: 70 * time.Millisecond, data: map[string]any{}},
	}, &recorder{})
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != fast {
		t.Fatalf("result host = %s, want %s", result.Host, fast)
	}
}

func TestSelectReusesBootstrapResultForDynamicCandidate(t *testing.T) {
	// apidd.spthgb 既是成功 bootstrap 也是动态候选；整个选线只探测它一次。
	const shared = "https://apidd.spthgb.com"
	rec := &recorder{}
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 100 * time.Millisecond, data: startupWithDomains(shared)},
		shared:                      {latency: 10 * time.Millisecond, data: map[string]any{}},
		"https://apidd.czssdgz.com": {err: errors.New("offline")},
		model.HostMain:              {err: errors.New("offline")},
	}, rec)
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != shared {
		t.Fatalf("result host = %s, want %s", result.Host, shared)
	}
	if n := rec.count(shared); n != 1 {
		t.Fatalf("shared host probed %d times, want 1 (result reuse)", n)
	}
}

func TestSelectAllFailAggregateError(t *testing.T) {
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {err: errors.New("mirror down")},
		"https://apidd.spthgb.com":  {err: errors.New("spthgb down")},
		"https://apidd.czssdgz.com": {err: errors.New("czssdgz down")},
		model.HostMain:              {err: errors.New("main down")},
	}, &recorder{})
	_, err := Select(context.Background(), SelectorOptions{}, probe)
	if err == nil {
		t.Fatal("expected aggregate error")
	}
	for _, host := range []string{model.HostMirror, "https://apidd.spthgb.com", "https://apidd.czssdgz.com", model.HostMain} {
		if !strings.Contains(err.Error(), host) {
			t.Fatalf("aggregate error missing %s: %v", host, err)
		}
	}
}

func TestSelectContextCancellationDuringBootstrap(t *testing.T) {
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {delay: time.Hour},
		"https://apidd.spthgb.com":  {delay: time.Hour},
		"https://apidd.czssdgz.com": {delay: time.Hour},
		model.HostMain:              {delay: time.Hour},
	}, &recorder{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Select(ctx, SelectorOptions{}, probe)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestSelectCancelledDuringCandidateProbe(t *testing.T) {
	dyn := "https://dyn.example"
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 1 * time.Millisecond, data: startupWithDomains(dyn)},
		"https://apidd.spthgb.com":  {delay: time.Hour},
		"https://apidd.czssdgz.com": {delay: time.Hour},
		model.HostMain:              {delay: time.Hour},
		dyn:                         {delay: time.Hour},
	}, &recorder{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Select(ctx, SelectorOptions{}, probe)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

// TestSelectCancelsProvablySlowerBootstrap 验证只有请求已进行至少当前最优单次耗时
// （不可能再更快）的 bootstrap 才会被取消，且取消后不阻塞选线。
func TestSelectCancelsProvablySlowerBootstrap(t *testing.T) {
	const slow = "https://apidd.spthgb.com"
	dyn := "https://dyn.example"
	probe := func(ctx context.Context, host string, onRequestStart func()) (time.Duration, map[string]any, error) {
		switch host {
		case model.HostMirror:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 10 * time.Millisecond, startupWithDomains(dyn), nil
		case slow:
			onRequestStart() // 请求开始，随后极慢
			select {
			case <-ctx.Done():
				return 0, nil, ctx.Err()
			case <-time.After(time.Hour):
				return 0, nil, errors.New("slow bootstrap unexpectedly completed")
			}
		case dyn:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 5 * time.Millisecond, map[string]any{}, nil
		default:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 0, nil, errors.New("offline " + host)
		}
	}
	start := time.Now()
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != dyn {
		t.Fatalf("result host = %s, want %s", result.Host, dyn)
	}
	if elapsed > time.Second {
		t.Fatalf("selection took %v; provably slower bootstrap was not cancelled", elapsed)
	}
}

// TestSelectKeepsSlowConstructionFastRequestBootstrap 验证：动态候选正是构造慢但请求快的
// bootstrap 时，不会被取消（requestStart 尚未设置，绝不取消），其 5ms 请求胜出，而不是被
// 永久排除后错误选择较慢的动态来源。
func TestSelectKeepsSlowConstructionFastRequestBootstrap(t *testing.T) {
	const mirror = "https://jdforrepam.com"
	const apidd = "https://apidd.spthgb.com"
	probe := func(ctx context.Context, host string, onRequestStart func()) (time.Duration, map[string]any, error) {
		switch host {
		case mirror:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 200 * time.Millisecond, startupWithDomains(apidd), nil
		case apidd:
			time.Sleep(100 * time.Millisecond) // 慢构造
			if onRequestStart != nil {
				onRequestStart()
			}
			return 5 * time.Millisecond, map[string]any{}, nil
		default:
			return 0, nil, errors.New("offline " + host)
		}
	}
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != apidd {
		t.Fatalf("result host = %s, want %s (slow-construction fast-request candidate must win)", result.Host, apidd)
	}
}

// TestSelectKeepsSlowConstructionFastRequestNonDynamicBootstrap 验证：构造慢、请求快且不在
// 动态列表中的 bootstrap 也不会被取消而永久排除，其更短请求耗时仍会胜出。
func TestSelectKeepsSlowConstructionFastRequestNonDynamicBootstrap(t *testing.T) {
	dyn := "https://dyn.example"
	fast := "https://apidd.spthgb.com"
	probe := func(ctx context.Context, host string, onRequestStart func()) (time.Duration, map[string]any, error) {
		switch host {
		case model.HostMirror:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 200 * time.Millisecond, startupWithDomains(dyn), nil
		case fast:
			time.Sleep(100 * time.Millisecond) // 慢构造
			if onRequestStart != nil {
				onRequestStart()
			}
			return 5 * time.Millisecond, map[string]any{}, nil
		case dyn:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 150 * time.Millisecond, map[string]any{}, nil
		default:
			return 0, nil, errors.New("offline " + host)
		}
	}
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != fast {
		t.Fatalf("result host = %s, want %s (non-dynamic slow-construction fast-request bootstrap must win)", result.Host, fast)
	}
}

// TestSelectDoesNotCancelPotentialDynamicSource 验证：在取得首个合法动态来源前，即使某个
// bootstrap 已"不可能成为最快 bootstrap"，也不能被取消——它可能是唯一能返回 apiDomains 的
// 来源；取消会永久丢失请求耗时更短的动态候选。
func TestSelectDoesNotCancelPotentialDynamicSource(t *testing.T) {
	dyn := "https://dyn.example"
	slowWithDomains := "https://apidd.czssdgz.com" // 50ms，唯一动态来源
	probe := func(ctx context.Context, host string, onRequestStart func()) (time.Duration, map[string]any, error) {
		switch host {
		case model.HostMirror:
			onRequestStart()
			return 5 * time.Millisecond, map[string]any{}, nil
		case slowWithDomains:
			onRequestStart()
			time.Sleep(50 * time.Millisecond) // 请求慢
			return 50 * time.Millisecond, startupWithDomains(dyn), nil
		case dyn:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 1 * time.Millisecond, map[string]any{}, nil
		default:
			if onRequestStart != nil {
				onRequestStart()
			}
			return 0, nil, errors.New("offline " + host)
		}
	}
	result, err := Select(context.Background(), SelectorOptions{}, probe)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if result.Host != dyn {
		t.Fatalf("result host = %s, want %s (potential dynamic source must not be cancelled early)", result.Host, dyn)
	}
}

// TestStartupProbeLatencyExcludesTransportConstruction 验证测速 latency 只统计单次 /startup
// 请求耗时，不包含 transport 构造（TLS client/cookie jar/proxy 初始化）。
func TestStartupProbeLatencyExcludesTransportConstruction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"settings":"x"}}`))
	}))
	defer server.Close()

	const constructionDelay = 200 * time.Millisecond
	factory := func(opts client.Options) (*client.Client, error) {
		time.Sleep(constructionDelay) // 模拟慢 transport 构造
		return client.New(opts)
	}
	probe, err := newStartupProbe(SelectorOptions{}, factory)
	if err != nil {
		t.Fatalf("newStartupProbe() error = %v", err)
	}
	start := time.Now()
	latency, _, err := probe(context.Background(), server.URL, nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("probe error = %v", err)
	}
	if latency >= constructionDelay/2 {
		t.Fatalf("latency = %v, includes transport construction", latency)
	}
	if elapsed < constructionDelay {
		t.Fatalf("elapsed = %v, want >= %v (construction delay was not exercised)", elapsed, constructionDelay)
	}
}
