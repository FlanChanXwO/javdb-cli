package magnets

import (
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
)

// writeMagnets 用 magnet 投影写出磁力行文本；空列表输出 (无磁力链)。
func writeMagnets(w, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, i18n.T("cli.message"))
		return
	}
	for _, row := range display.ProjectMagnets(items) {
		fmt.Fprintln(w, row.Line())
		fmt.Fprintln(w, row.HashLine())
	}
}

// writeJSON 以 jsonx.MarshalLine 写出紧凑 JSON 并传播编码与写入错误。
func writeJSON(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
