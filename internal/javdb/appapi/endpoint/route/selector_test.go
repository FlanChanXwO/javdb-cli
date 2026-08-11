package route

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

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

// fakeProbe 构造记录调用、按配置返回的可控 probe。
func fakeProbe(specs map[string]probeSpec, rec *recorder) Probe {
	return func(ctx context.Context, host string) (time.Duration, map[string]any, error) {
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

// TestSelectCancelsSlowerBootstrapsAfterDynamicSource 验证找到动态来源后，仍未完成且
// 不可能更快的 bootstrap 请求会被取消：慢 bootstrap 不会阻塞选线，也不会被当作成功候选。
func TestSelectCancelsSlowerBootstrapsAfterDynamicSource(t *testing.T) {
	const slow = "https://apidd.spthgb.com"
	dyn := "https://dyn.example"
	rec := &recorder{}
	probe := fakeProbe(map[string]probeSpec{
		model.HostMirror:            {latency: 50 * time.Millisecond, data: startupWithDomains(dyn)},
		slow:                        {delay: time.Hour, latency: 999 * time.Millisecond},
		"https://apidd.czssdgz.com": {delay: time.Hour, latency: 998 * time.Millisecond},
		model.HostMain:              {delay: time.Hour, latency: 997 * time.Millisecond},
		dyn:                         {latency: 5 * time.Millisecond, data: map[string]any{}},
	}, rec)
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
		t.Fatalf("selection took %v; slower bootstraps were not cancelled", elapsed)
	}
	if rec.count(slow) != 1 {
		t.Fatalf("slow bootstrap probed %d times, want 1", rec.count(slow))
	}
}
