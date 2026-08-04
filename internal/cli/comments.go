package cli

import (
	"fmt"
	"io"
)

// PrintMovieComments 以可读的逐条形式输出评论；JSON 模式保留上游所有字段。
func PrintMovieComments(w io.Writer, errW io.Writer, reviews []map[string]any) {
	if len(reviews) == 0 {
		fmt.Fprintln(errW, "(无评论)")
		return
	}
	for _, review := range reviews {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", anyString(review["id"]), reviewAuthor(review), reviewValue(review["score"]), anyString(review["created_at"]))
		content := anyString(review["content"])
		if content == "" {
			content = anyString(review["comment"])
		}
		if content == "" {
			content = anyString(review["body"])
		}
		if content != "" {
			fmt.Fprintln(w, content)
		}
	}
}

func reviewValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func reviewAuthor(review map[string]any) string {
	for _, key := range []string{"user_name", "username", "user_nickname"} {
		if name := anyString(review[key]); name != "" {
			return name
		}
	}
	user, _ := review["user"].(map[string]any)
	for _, key := range []string{"name", "username", "nickname"} {
		if name := anyString(user[key]); name != "" {
			return name
		}
	}
	return ""
}
