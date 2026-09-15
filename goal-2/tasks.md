# Goal 2 Tasks

状态约定：`[ ]` 未完成，`[x]` 已完成，`[!]` 阻塞。每轮只执行第一个未完成 task。代码 task 必须先 Red 再 Green（Red 必须实际运行并观察到失败）；完成后在 worktree 提交代码并回写本文件的「实际完成/验证证据/剩余风险/下一步」。工作目录：`/Users/flanchan/Developer/Projects/GithubProjects/javdb-cli/.worktrees/assets-media-download`（分支 `feat/assets-media-download`）。

集中检查 task 清单（每 3 个普通 task 一次）：需求是否偏离 input.md；bug/死代码/调试残留；类型检查；构建；测试（含未动到的相关测试）；安全性（凭证泄露、注入、scheme 校验）；数据一致性；回滚方案是否需调整；文档是否同步。

---

## Task 1 — 建立隔离 worktree 与干净基线

- 状态：[ ]
- 范围：`.worktrees/assets-media-download`（新分支 `feat/assets-media-download`，基于最新 `origin/main`）。主仓库不改业务代码。
- 验收：worktree 已建；`go mod download` 完成；基线 `go test ./...` 与 `sh scripts/build.sh` 在 worktree 内通过；报告基线结果。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 2 — Dependency spike：pure-Go TS demux / MP4 mux 评估

- 状态：[ ]
- 范围：只读评估 + `goal-2/spike-report.md` 报告，不改 go.mod、不写业务代码。
- 验收：评估候选库（Eyevinn/mp4ff、yapingcat/gomedia、bluenviron/mediacommon、abema/go-mp4 等）在 Go 版本、license、维护状态、依赖树、CGO、TS demux、H264/AAC 解析、PTS/DTS/B-frame、MP4 mux、faststart、sample table 正确性上的表现；给出「手写 vs 引库」结论；引库路径标注必须等维护者审批（input §28）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 3 — SDK：MovieAsset{type,url} 模型与 MovieAssets() 发现 API

- 状态：[ ]
- 范围：`sdk/`（新文件或 movie.go）；TDD。仅 type+url 两字段（input §2/§46）；资产序列 thumbnail→cover→preview_images[*]→preview_video，缺失跳过（§4）；preview_images 每项取 large_url||thumb_url（§5）；map[string]any 提取逻辑沿用现状风格。
- 验收：`MovieAssets(ctx, movieID) ([]MovieAsset, error)`；表驱动测试覆盖多 preview、缺失项、fallback、cover、video；JSON tag 严格 `type`/`url`；不新增 kind/role/id/index 字段。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 4 — 集中检查 1（Task 1-3）

- 状态：[ ]
- 检查清单结论（需求偏离/bug/死代码/类型/构建/测试/安全/回滚/文档）：
- 实际完成：
- 验证证据：
- 追加修复 task：
- 下一步：

## Task 5 — CLI：assets list（--type、selector、四种输出）

- 状态：[ ]
- 范围：新建 `internal/cli/commands/assets/`（assets.go + list.go，selector 较大则 selector.go）；注册进 `internal/cli/root.go`；同步 `internal/cli/root_test.go` help 字面量。不进 pipeline envelope（input §12，不加 KindAsset）；非 TTY 默认 `TYPE<TAB>URL`；`--json`/`--ndjson` 严格仅 type+url；DESCRIPTION 仅 TTY 渲染（§6）；filter 先于 selector（§9）；无 --all/--preview（§10、invariant 7/8）。
- 验收：selector `1 / 1 3 5 / 1-4 / 1,3,5 / 1,3-5 / 1 3-5` 合法，`0 / -1 / 1- / 4-1 / foo` 非法，重复区间去重按列表顺序（§8）；表驱动覆盖 TTY/pipe/JSON/NDJSON 四态与 contract test（防未来塞 schema/kind/role/index，§54）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 6 — CLI：assets download 管道消费者

- 状态：[ ]
- 范围：`commands/assets/download.go`；读 stdin `TYPE<TAB>URL` 行；`-d DIR`（默认 `.`）、`-o PATH`（仅恰好 1 个资产，否则 `assets download: -o requires exactly one asset`）；默认命名 image-001.<magic ext> / video-001.mp4（§15-16）；目标已存在 fail 不覆盖；多文件 target preflight（§42-43）；无效行/未知 type 显式失败（§55）。
- 验收：单图/多图/单视频、-d/-o、-o 多输入失败、已存在失败、invalid line 失败、unsupported type 失败；不需要 --ndjson 即可 pipe（§76）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 7 — media：图片下载校验链与原子发布

- 状态：[ ]
- 范围：`internal/javdb/appapi/media/`；download → detect → XOR unwrap if needed → detect again → validate → target.tmp → atomic rename（§19/§41）；magic 白名单沿用（JPEG/PNG/GIF/WEBP/AVIF/HEIC-like ftyp，§20）；HTML/JSON/403 页/Cloudflare 页/空响应/坏 XOR/未知二进制必须失败，HTTP 200+非图片不得落盘成 .jpg（§20）；扩展名由检测结果决定（§15）。
- 验收：表驱动覆盖各 magic、XOR 还原、坏 payload 拒绝、原子发布、不覆盖已存在；fixture 内联构造（§56）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 8 — 集中检查 2（Task 5-7）

- 状态：[ ]
- 检查清单结论：
- 实际完成：
- 验证证据：
- 追加修复 task：
- 下一步：

## Task 9 — media：统一 MediaFetcher 与 TS segment 完整性（Layer A）

- 状态：[ ]
- 范围：`internal/javdb/appapi/media/` + `appapi/client/transport.go`（FetchMedia 扩展：仅 http/https、防盗链头内部处理（§21）、cookie/authorization 不跨 host 传播、日志无 credential（§22）、context 贯穿）；TS 校验：0x47 sync、188 对齐、PAT/PMT parseable、PID 一致、PES 可解析、明显截断（§30）；continuity 校验容忍 segment 边界/DISCONTINUITY/合法 reset（§31）；坏 segment bounded retry=3 后整资产失败（§32，计划明确要求）。
- 验收：表驱动 valid TS/truncated/invalid sync/missing PAT-PMT/broken PES/合法 discontinuity/retry 成功/重试耗尽（§58）；fixture 内联构造最小 TS。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 10 — media：流式 HLS 获取与 .ts 全链输出

- 状态：[ ]
- 范围：`internal/javdb/appapi/media/`；segment 下载→validate transport→decrypt→validate TS→commit 流水线（§30）；HLS 范围保持 single-media VOD + MPEG-TS + AES-128（§23，不扩 master/live/BYTERANGE/fMP4）；不整载内存：流式 + bounded segment buffer（§36）；.ts 经 Layer A + final media validation + atomic publish（§33）；context 取消停止网络/清理 temp/返回 context error（§44）。
- 验收：VOD 全链 httptest 端到端、AES 成功/失败、取消清理 temp、内存不随长度线性（结构层面断言 + 必要的基准观测）、失败无半成品。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 11 — media：pure-Go TS→MP4 remux、Fast Start、Layer B/C

- 状态：[ ]
- 范围：`internal/javdb/appapi/media/`；TS→PES→H.264 AnnexB + AAC ADTS→MP4 samples（avc1/mp4a）（§26）；仅 H.264/AAC，其他 codec 明确失败 `unsupported video codec`（§24）；不转码、像素/PCM 原样；两阶段 fast start：spool+sample table → ftyp→moov→mdat（§35）；临时磁盘 ≤2× 输出（§37）；timed ID3 丢弃（§39）；Layer B media integrity（track/samples/SPS-PPS/AAC config/duration/timestamps/sync samples，§34）与 Layer C container integrity（重开解析 ftyp/moov/mdat/moov<mdat/offsets bounds/sample table coherent，§40）通过后才发布（§41）。
- 验收：真实 MP4 断言（§59：ftyp/moov/mdat/moov<mdat/avc1/mp4a/duration>0/samples>0/valid offsets）；video-only、unsupported codec、malformed TS、remux 失败、final validation 拒绝损坏 temp（§60）；ffprobe 仅可选冒烟（§61）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 12 — 集中检查 3（Task 9-11）

- 状态：[ ]
- 检查清单结论：
- 实际完成：
- 验证证据：
- 追加修复 task：
- 下一步：

## Task 13 — SDK：DownloadMovieAsset + 移除旧媒体 API 与顶层 download 命令

- 状态：[ ]
- 范围：`sdk/movie.go`：`DownloadMovieAsset(ctx, asset, target) (int64, error)`（video target 仅 .ts/.mp4，否则 `unsupported video output format ".mkv"`，§47；image 走 Task 7 链，§48）；删除 `MovieMediaDownloadOptions/MovieMediaDownloadResult/DownloadMovieMedia` 及未导出 helpers，不加 deprecated wrapper（§49）；CLI `commands/download/` 移除（含 KindDownload envelope 路径，rg 全部引用逐一处理）、`assets download` 接管；`commands/assets/download.go` 经 sdk facade 调用（cli 只经 sdk 触达远程，架构边界）；同步 root.go/root_test.go。
- 验收：`go build ./...` + 全部引用清理；facade 测试（`TestClientAPIReturnsCompatibleFacade`）更新后通过；不新增额外 metadata 字段（§46）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 14 — 文档与 agent skill 同步

- 状态：[ ]
- 范围（workspace AGENTS.md 路由 + input §69-71）：`README.md`/`README.zh-CN.md`（主路径示例 §70）；`docs/en/cli-reference.md`/`docs/zh-CN/cli-reference.md`；`docs/en/sdk.md`/`docs/zh-CN/sdk.md`（及 `docs/sdk.md`/`docs/sdk.zh-CN.md` 若为副本）；`docs/maintainers/architecture.md`；`skills/javdb-cli/SKILL.md` + `references/*`（agent 先 list→选择→pipe download 的用法，§71）。
- 验收：双语一致；示例与最终 CLI contract 逐字吻合；无对已删命令/类型的引用（rg 校验 `MovieMediaDownloadOptions`、`javdb download` 等）；不含 Telegram 逻辑描述（invariant 22）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 15 — 集中检查 4（全量门禁）

- 状态：[ ]
- 范围：`go test ./...`、`sh scripts/build.sh`、`gofmt -l`、`go vet ./...`；文档核对；安全复查（凭证日志、scheme 校验、临时文件清理）；invariants §86 全 23 条逐条对照。
- 实际完成：
- 验证证据：
- 追加修复 task：
- 下一步：

## Task 16 — Real E2E（真实 API）

- 状态：[ ]
- 范围：`javdb assets list NUMBER`（数量/类型）；`--type image 1-2 | assets download -d /tmp/assets`（图片可打开）；`--type video | assets download -o /tmp/preview.mp4`（H264/AAC/duration/seek/moov before mdat）；`-o /tmp/preview.ts`（真实 TS 可播放，§62-64）。需真实网络与本机认证；认证缺失则标 [!] 并写明原因。
- 验收：实测命令输出与文件校验证据（ffprobe 若存在仅作辅助）。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：

## Task 17 — 终审（最大范围复查与 goal 完成）

- 状态：[ ]
- 范围：从 C 端体验、代码质量、安全、数据一致性、错误处理、测试覆盖、构建产物、文档、回滚方案全角度复查；对照 input §76-86 DoD 逐条核验；汇总所有阻塞项与低风险遗留。
- 实际完成：
- 验证证据：
- 剩余风险：
- 下一步：
