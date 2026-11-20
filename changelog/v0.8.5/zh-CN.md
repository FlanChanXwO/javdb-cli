# v0.8.5 — 2026-11-20

## 新增

- 可用 `--log-format text|json`、`--log-level debug|info|warning|error` 及对应的 `log_format` / `log_level` 配置选择 stderr 诊断格式和级别。日志包含时间、命令、源码位置，每个失败输入只记录一次错误。 ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))
- 帮助、交互、状态与错误提示根据系统语言显示英文或简体中文；API 的 `lang` 保留原有含义。命令语法、JSON/NDJSON 字段与素材管道格式保持稳定。 ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))

## 变更

- `auth check` 通过服务端验证已保存的 token，被拒绝时退出码为 1，`--json` 同样适用。脚本应检查退出码和 JSON 结果。 ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))

## 修复

- 演员、系列、片商、导演、番号、清单、浏览、标签与类别查询统一使用已保存的默认账号，修复部分线路在登录后仍拒绝请求的问题。 ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))
- 普通命令与管道调用在缺少或被拒绝的凭据下，给出可操作的 `javdb auth login` 提示并保留底层错误。显式开启 `auto_relogin=true` 时，同次调用共享一次重新登录，先保存新 token 再重试失败操作。 ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))
- 修正中文素材表头与数据列的对齐，补全分组帮助标题和默认值提示翻译，统一双语命令用词。 ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))

**完整变更**：[v0.8.4...v0.8.5](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.4...v0.8.5)

<!-- internal: https://github.com/FlanChanXwO/javdb-cli/pull/109 -->
