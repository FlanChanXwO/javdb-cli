package search

import (
	"bytes"
	"encoding/json"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewHelpListsExpectedFlags(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"--zone", "--sort", "--filter-by", "--type", "--has-magnets", "--json", "--page", "--limit"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s:\n%s", want, out.String())
		}
	}
}

func TestNewRequiresKeyword(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "keyword or an image") {
		t.Fatalf("expected keyword/image error, got %v", err)
	}
}

func TestSearchTypeKey(t *testing.T) {
	if searchTypeKey("actor") != "actors" || searchTypeKey("list") != "lists" || searchTypeKey("") != "movies" {
		t.Fatal("search type key mapping changed")
	}
}

// TestExecuteMoviesJSONHasMagnetsFilter 覆盖 movie 分支 JSON + has-magnets 过滤。
func TestExecuteMoviesJSONHasMagnetsFilter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[
			{"number":"SSIS-589","id":"x","title":"T","magnets_count":2},
			{"number":"ZERO","id":"y","title":"Z","magnets_count":0}
		]}}`))
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"kw", "--json", "--has-magnets"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	arr, _ := got["movies"].([]any)
	if len(arr) != 1 || arr[0].(map[string]any)["number"] != "SSIS-589" {
		t.Fatalf("has-magnets filter result = %v", got)
	}
}

// TestExecuteNamedText 覆盖 --type actor 命名分支文本输出。
func TestExecuteNamedText(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"actors":[
			{"id":"9Dqpw","name":"山手梨愛","videos_count":10}
		]}}`))
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	// 人类命名实体文本只在 TTY stdout 下验证；非 TTY 默认是稳定记录管道。
	streams.OutIsTerminal = true
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"kw", "--type", "actor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	s := streams.Out.(*bytes.Buffer).String()
	if !strings.Contains(s, "9Dqpw") || !strings.Contains(s, "山手梨愛") {
		t.Fatalf("named output = %q", s)
	}
}
