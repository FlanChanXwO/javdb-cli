package comments

import (
	"fmt"
	"io"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
)

// writeComments 以可读的逐条形式输出评论；保留完整内容与字段降级，不新增截断。
func writeComments(w, errW io.Writer, reviews []map[string]any) error {
	if len(reviews) == 0 {
		_, err := errW.Write([]byte("(无评论)\n"))
		return err
	}
	for _, review := range reviews {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			display(review["id"]), reviewAuthor(review), reviewValue(review["score"]), display(review["created_at"])); err != nil {
			return err
		}
		content := display(review["content"])
		if content == "" {
			content = display(review["comment"])
		}
		if content == "" {
			content = display(review["body"])
		}
		if content != "" {
			if _, err := fmt.Fprintln(w, content); err != nil {
				return err
			}
		}
	}
	return nil
}

func reviewValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func reviewAuthor(review map[string]any) string {
	for _, key := range []string{"user_name", "username", "user_nickname"} {
		if name := display(review[key]); name != "" {
			return name
		}
	}
	user, _ := review["user"].(map[string]any)
	for _, key := range []string{"name", "username", "nickname"} {
		if name := display(user[key]); name != "" {
			return name
		}
	}
	return ""
}

func display(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return scalar.String(t)
	}
}
