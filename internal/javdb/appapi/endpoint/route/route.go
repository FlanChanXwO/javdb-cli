package route

import "context"

// RouteEndpoint 是自动线路选择 capability。它不使用共享 transport（每个 probe 独立构建
// 零重试 transport），因此不持有状态；经根 Client 嵌入后，方法可经 method promotion 访问。
type RouteEndpoint struct{}

// NewRoute 构造 route capability。
func NewRoute() *RouteEndpoint { return &RouteEndpoint{} }

// Select 执行并发动态线路选择，委托包级 Select。
func (*RouteEndpoint) Select(ctx context.Context, opts SelectorOptions, probe Probe) (Result, error) {
	return Select(ctx, opts, probe)
}
