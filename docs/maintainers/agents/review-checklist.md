# Review Checklist

## 边界与兼容性

- `cmd/javdb` 是否仍只委托 CLI？
- CLI 是否只通过公开 `javdb` facade 执行远程 JavDB 操作？
- 是否把协议、签名或 HTTP 细节错误地暴露为公开 SDK 契约？
- CLI flag、JSON 字段和文本列是否保持既有脚本/agent 的兼容性？

## 凭据与状态

- 是否避免打印、记录或测试夹带密码、JWT、`auth.json` 内容？
- 是否把账号、配置、tag cache 和远程 watch/want 操作视为显式状态变化？
- 错误是否暴露真因，而非伪造成正常空结果或成功？

## 测试与文档

- 是否为行为变更补充或更新聚焦测试？
- 是否运行相关的 `go test`、race、vet、构建与脚本检查？
- 是否同步两个 locale 的 public contract、README、skill 和 changelog？
- 新文档链接是否指向 `docs/<locale>/` 或 `docs/maintainers/` 的权威路径？
