// Package scalar 提供跨领域复用的基础标量转换。
package scalar

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// String 将可选值转换为字符串；nil 保持为空字符串。
//
// 浮点数的展示格式沿用 fmt.Sprint。CLI 若需要兼容旧输出中的浮点截断，
// 应在 CLI 自己的 wrapper 中处理，不在共享包中引入展示语义。
func String(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// Int64 将常见的 JSON/Go 数值及十进制整数字符串转换为 int64。
// 浮点数沿用现有数据层的截断行为；超出 int64 可表达范围的值返回 false。
func Int64(v any) (int64, bool) {
	switch t := v.(type) {
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case int:
		return int64(t), true
	case int8:
		return int64(t), true
	case int16:
		return int64(t), true
	case int32:
		return int64(t), true
	case int64:
		return t, true
	case uint:
		return uint64ToInt64(uint64(t))
	case uint8:
		return int64(t), true
	case uint16:
		return int64(t), true
	case uint32:
		return int64(t), true
	case uint64:
		return uint64ToInt64(t)
	case uintptr:
		return uint64ToInt64(uint64(t))
	case float32:
		return float64ToInt64(float64(t))
	case float64:
		return float64ToInt64(t)
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func uint64ToInt64(v uint64) (int64, bool) {
	if v > math.MaxInt64 {
		return 0, false
	}
	return int64(v), true
}

func float64ToInt64(v float64) (int64, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < math.MinInt64 || v >= math.MaxInt64+1.0 {
		return 0, false
	}
	return int64(v), true
}
