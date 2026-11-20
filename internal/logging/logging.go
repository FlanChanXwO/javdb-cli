// Package logging owns stderr diagnostics independently of command result streams.
package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/term"

	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
)

// New creates an invocation-local logger. It never changes slog's process default.
func New(w io.Writer, level, format string) (*slog.Logger, error) {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "info":
		lv = slog.LevelInfo
	case "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		return nil, fmt.Errorf(i18n.T("log.invalid_level"), level)
	}
	switch format {
	case "json":
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: true, Level: lv, ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey && a.Value.Any() == slog.LevelWarn {
				return slog.String(slog.LevelKey, "WARNING")
			}
			return a
		}})), nil
	case "text":
		color := false
		if f, ok := w.(*os.File); ok {
			_, disabled := os.LookupEnv("NO_COLOR")
			color = term.IsTerminal(int(f.Fd())) && !disabled
		}
		return slog.New(&textHandler{out: &output{w: w, color: color}, level: lv}), nil
	default:
		return nil, fmt.Errorf(i18n.T("log.invalid_format"), format)
	}
}

type output struct {
	mu    sync.Mutex
	w     io.Writer
	color bool
}
type textHandler struct {
	out   *output
	level slog.Level
	attrs []slog.Attr
	group string
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.level }
func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copy := *h
	copy.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &copy
}
func (h *textHandler) WithGroup(name string) slog.Handler {
	copy := *h
	if name != "" {
		copy.group += name + "."
	}
	return &copy
}
func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := append([]slog.Attr(nil), h.attrs...)
	r.Attrs(func(a slog.Attr) bool { attrs = append(attrs, a); return true })
	command := "javdb"
	for _, a := range attrs {
		if a.Key == "command" {
			command = a.Value.String()
		}
	}
	level := r.Level.String()
	if r.Level == slog.LevelWarn {
		level = "WARNING"
	}
	if h.out.color {
		color := "\x1b[36m"
		switch r.Level {
		case slog.LevelError:
			color = "\x1b[31m"
		case slog.LevelWarn:
			color = "\x1b[33m"
		case slog.LevelDebug:
			color = "\x1b[34m"
		}
		level = color + level + "\x1b[0m"
	}
	source := ""
	if r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := frames.Next()
		source = filepath.Base(filepath.Dir(frame.File)) + "/" + filepath.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
	}
	var b strings.Builder
	message := r.Message
	if strings.IndexFunc(message, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		message = strconv.Quote(message)
	}
	fmt.Fprintf(&b, "[%s] [%s] [%s] [%s]: %s", r.Time.Local().Format("15:04:05.000"), command, level, source, message)
	for _, a := range attrs {
		if a.Key == "command" {
			continue
		}
		a.Value = a.Value.Resolve()
		fmt.Fprintf(&b, " %s=%s", h.group+a.Key, strconv.Quote(a.Value.String()))
	}
	b.WriteByte('\n')
	h.out.mu.Lock()
	defer h.out.mu.Unlock()
	_, err := io.WriteString(h.out.w, b.String())
	return err
}

// Emit preserves write failures instead of discarding handler errors.
func Emit(logger *slog.Logger, level slog.Level, message string, attrs ...slog.Attr) error {
	return emitAt(logger, level, message, attrs...)
}

func emitAt(logger *slog.Logger, level slog.Level, message string, attrs ...slog.Attr) error {
	ctx := context.Background()
	h := logger.Handler()
	if !h.Enabled(ctx, level) {
		return nil
	}
	var pc [1]uintptr
	runtime.Callers(3, pc[:])
	record := slog.NewRecord(time.Now(), level, message, pc[0])
	record.AddAttrs(attrs...)
	return h.Handle(ctx, record)
}

// Reported marks a batch whose individual errors were already emitted.
type Reported struct {
	Failed, Total int
	Cause         error
}

func (e *Reported) Error() string {
	return fmt.Sprintf(i18n.T("batch.failed_count"), e.Failed, e.Total)
}
func (e *Reported) Unwrap() error { return e.Cause }

// Failure reports a command failure once, with upstream diagnostics separate from user guidance.
func Failure(logger *slog.Logger, err error, contextAttrs ...slog.Attr) error {
	if len(contextAttrs) > 0 {
		logger = slog.New(logger.Handler().WithAttrs(contextAttrs))
	}
	var batch *Reported
	if errors.As(err, &batch) {
		if batch.Total <= 1 {
			return nil
		}
		return emitAt(logger, slog.LevelError, i18n.T("batch.failed"), slog.Int("failed", batch.Failed), slog.Int("total", batch.Total))
	}
	var user interface {
		MessageKey() string
		Unwrap() error
	}
	message := i18n.T("error.failed")
	cause := err
	if errors.As(err, &user) {
		message = i18n.T(user.MessageKey())
		cause = user.Unwrap()
	}
	var attrs []slog.Attr
	if cause != nil {
		attrs = append(attrs, slog.String("error", cause.Error()))
	}
	return emitAt(logger, slog.LevelError, message, attrs...)
}

// Writer adapts existing stderr notes to the invocation's diagnostic format.
// Command result writers are never routed through this adapter.
func Writer(logger *slog.Logger, level slog.Level) io.Writer {
	return messageWriter{logger: logger, level: level}
}

type messageWriter struct {
	logger *slog.Logger
	level  slog.Level
}

func (w messageWriter) Write(data []byte) (int, error) {
	ctx := context.Background()
	handler := w.logger.Handler()
	if !handler.Enabled(ctx, w.level) {
		return len(data), nil
	}
	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var pc uintptr
	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "fmt.") && !strings.Contains(frame.Function, "/internal/logging.") {
			pc = frame.PC
			break
		}
		if !more {
			break
		}
	}
	record := slog.NewRecord(time.Now(), w.level, strings.TrimSuffix(string(data), "\n"), pc)
	if err := handler.Handle(ctx, record); err != nil {
		return 0, err
	}
	return len(data), nil
}
