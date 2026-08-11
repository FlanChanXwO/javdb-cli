package route

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// bootstrapHosts 是并发探测的固定入口，顺序即 tie-break 的固定 bootstrap 顺序。
var bootstrapHosts = []string{
	model.HostMirror,
	"https://apidd.spthgb.com",
	"https://apidd.czssdgz.com",
	model.HostMain,
}

// SelectorOptions 是自动选线的 effective 网络选项（与具体 host 无关）。
type SelectorOptions struct {
	PreferredHost string
	Proxy         string
	DeviceUUID    string
	Timeout       time.Duration
	Lang          string
}

// Result 是自动选线结果。Latency 是缓存验证成功或全量测速赢家的单次 /startup 耗时。
type Result struct {
	Host            string
	Latency         time.Duration
	ReusedPreferred bool
}

// Probe 对单个 host 发起一次 /startup 请求并返回单次耗时与 startup data。
// selector 通过该函数隔离真实 transport，测试可注入可控 probe。
type Probe func(ctx context.Context, host string) (latency time.Duration, data map[string]any, err error)

// probeResult 是一次探测的原始结果。
type probeResult struct {
	host    string
	latency time.Duration
	data    map[string]any
	err     error
}

// NewStartupProbe 构造零重试的 /startup 探测函数。所有 probe 共享同一 effective 网络选项
// 与稳定 device UUID，不携带 bearer token。
func NewStartupProbe(opts SelectorOptions) (Probe, error) {
	return newStartupProbe(opts, client.New)
}

// probeTransport 构造零重试签名 transport 的可注入依赖，便于测试区分构造耗时与请求耗时。
type probeTransport func(opts client.Options) (*client.Client, error)

// newStartupProbe 是 NewStartupProbe 的可注入核心。
func newStartupProbe(opts SelectorOptions, factory probeTransport) (Probe, error) {
	deviceUUID := opts.DeviceUUID
	if deviceUUID == "" {
		// 本次选线共享一个稳定 UUID，避免每个 probe 各自生成不同随机 UUID。
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, fmt.Errorf("generate probe device uuid: %w", err)
		}
		deviceUUID = id.String()
	}
	zero := 0
	return func(ctx context.Context, host string) (time.Duration, map[string]any, error) {
		t, err := factory(client.Options{
			Host:       host,
			Proxy:      opts.Proxy,
			Timeout:    opts.Timeout,
			Lang:       opts.Lang,
			DeviceUUID: deviceUUID,
			Retries:    &zero,
		})
		if err != nil {
			return 0, nil, err
		}
		// Latency 只统计单次 /startup 请求耗时，不含 transport 构造（TLS/cookie jar/proxy）。
		start := time.Now()
		var data map[string]any
		if err := t.GetJSONContext(ctx, "/api/v1/startup", nil, &data); err != nil {
			return 0, nil, err
		}
		return time.Since(start), data, nil
	}, nil
}

// Select 执行并发动态线路选择。
//
// 顺序：preferred 严格校验并单次探测，成功立即复用；否则并发探测固定 bootstrap，首个
// 解出非空完整合法 apiDomains 的响应成为动态候选来源，取消仍未完成且已不可能更快的
// bootstrap；对动态候选完整读取、规范化、去重并并发测速，已探测 URL 复用结果；在所有
// 成功候选中选单次耗时最短者，相同耗时按动态响应顺序、再按固定 bootstrap 顺序稳定决胜。
// context 取消立即返回取消错误；全部候选失败才返回包含逐 host 原因的聚合错误。
func Select(ctx context.Context, opts SelectorOptions, probe Probe) (Result, error) {
	var failures []string

	if opts.PreferredHost != "" {
		normalized, err := normalizeCandidate(opts.PreferredHost)
		if err != nil {
			return Result{}, fmt.Errorf("preferred host: %w", err)
		}
		latency, _, err := probe(ctx, normalized)
		if err == nil {
			if ctx.Err() != nil {
				return Result{}, ctx.Err()
			}
			return Result{Host: normalized, Latency: latency, ReusedPreferred: true}, nil
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		failures = append(failures, fmt.Sprintf("%s: %v", normalized, err))
	}

	bootstraps, dynamicData, probedHosts, bootstrapFailures := probeBootstraps(ctx, probe)
	failures = append(failures, bootstrapFailures...)
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}

	// 动态候选：完整读取（提取在 probeBootstraps 已校验成功）、已探测 URL 复用结果。
	var dynamicCandidates []string
	if dynamicData != nil {
		dynamicCandidates, _ = APIHostsFromStartupData(dynamicData)
	}
	knownLatency := make(map[string]time.Duration, len(bootstraps))
	for _, b := range bootstraps {
		knownLatency[b.host] = b.latency
	}

	// 并发探测尚未探测过的动态候选（bootstrap 阶段已探测的 URL 复用其结果）。
	var toProbe []string
	for _, host := range dynamicCandidates {
		if !probedHosts[host] {
			toProbe = append(toProbe, host)
		}
	}
	if len(toProbe) > 0 {
		candResults := make(chan probeResult, len(toProbe))
		for _, host := range toProbe {
			go func(host string) {
				latency, _, err := probe(ctx, host)
				candResults <- probeResult{host: host, latency: latency, err: err}
			}(host)
		}
		for range toProbe {
			select {
			case <-ctx.Done():
				return Result{}, ctx.Err()
			case r := <-candResults:
				if r.err != nil {
					failures = append(failures, fmt.Sprintf("%s: %v", r.host, r.err))
					continue
				}
				knownLatency[r.host] = r.latency
			}
		}
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}

	bootstrapOrder := make(map[string]int, len(bootstrapHosts))
	for i, host := range bootstrapHosts {
		bootstrapOrder[host] = i
	}
	dynamicOrder := make(map[string]int, len(dynamicCandidates))
	for i, host := range dynamicCandidates {
		dynamicOrder[host] = i
	}

	type ranked struct {
		host      string
		latency   time.Duration
		dynOrder  int
		bootOrder int
	}
	var rankedList []ranked
	seen := make(map[string]bool)
	addRanked := func(host string, latency time.Duration) {
		if seen[host] {
			return
		}
		seen[host] = true
		do, ok := dynamicOrder[host]
		if !ok {
			do = -1
		}
		bo, ok := bootstrapOrder[host]
		if !ok {
			bo = -1
		}
		rankedList = append(rankedList, ranked{host: host, latency: latency, dynOrder: do, bootOrder: bo})
	}
	for _, host := range dynamicCandidates {
		if lat, ok := knownLatency[host]; ok {
			addRanked(host, lat)
		}
	}
	for _, b := range bootstraps {
		addRanked(b.host, b.latency)
	}

	if len(rankedList) == 0 {
		return Result{}, fmt.Errorf("route selection failed: %s", strings.Join(failures, "; "))
	}

	sort.SliceStable(rankedList, func(i, j int) bool {
		a, b := rankedList[i], rankedList[j]
		if a.latency != b.latency {
			return a.latency < b.latency
		}
		return rankLess(a.dynOrder, a.bootOrder, b.dynOrder, b.bootOrder)
	})
	winner := rankedList[0]
	return Result{Host: winner.host, Latency: winner.latency}, nil
}

// probeBootstraps 并发探测固定 bootstrap，返回成功者、首个合法 apiDomains 来源、全部已探测
// host 与失败原因。找到动态来源后取消仍未完成且不可能更快的其余 bootstrap 请求，但仍收集
// 已返回的成功者与失败原因。
func probeBootstraps(ctx context.Context, probe Probe) (bootstraps []probeResult, dynamicData map[string]any, probed map[string]bool, failures []string) {
	bootstrapCtx, cancelBootstrap := context.WithCancel(ctx)
	defer cancelBootstrap()
	results := make(chan probeResult, len(bootstrapHosts))
	for _, host := range bootstrapHosts {
		go func(host string) {
			latency, data, err := probe(bootstrapCtx, host)
			results <- probeResult{host: host, latency: latency, data: data, err: err}
		}(host)
	}

	probed = make(map[string]bool, len(bootstrapHosts))
	received := 0
	for received < len(bootstrapHosts) {
		select {
		case <-ctx.Done():
			return bootstraps, nil, probed, failures
		case r := <-results:
			received++
			probed[r.host] = true
			if r.err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", r.host, r.err))
				continue
			}
			bootstraps = append(bootstraps, r)
			if dynamicData == nil {
				domains, err := APIHostsFromStartupData(r.data)
				if err == nil && len(domains) > 0 {
					dynamicData = r.data
					// 已找到动态来源；其余仍未完成且已不可能更快的 bootstrap 请求取消。
					cancelBootstrap()
				}
			}
		}
	}
	return bootstraps, dynamicData, probed, failures
}

// rankLess 实现 tie-break：动态响应顺序优先，非动态候选排在所有动态候选之后，
// 再按固定 bootstrap 顺序。
func rankLess(aDyn, aBoot, bDyn, bBoot int) bool {
	aIsDyn, bIsDyn := aDyn >= 0, bDyn >= 0
	if aIsDyn != bIsDyn {
		return aIsDyn
	}
	if aIsDyn && aDyn != bDyn {
		return aDyn < bDyn
	}
	aIsBoot, bIsBoot := aBoot >= 0, bBoot >= 0
	if aIsBoot != bIsBoot {
		return aIsBoot
	}
	return aBoot < bBoot
}
