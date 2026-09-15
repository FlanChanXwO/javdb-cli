# Goal 2 计划：javdb-cli 资产发现、下载与 Progressive MP4

## 目标与来源

本 goal 执行用户提供的《javdb-cli 资产发现、下载与 Progressive MP4 完整实施计划》（全文逐字保存在 `goal-2/input.md`）。最终交付：

- `javdb assets list` / `javdb assets download` 作为唯一资产命令域；
- 极小 Asset 数据模型 `MovieAsset{Type, URL}`；
- 经完整性验证的图片下载（magic 检测 + XOR 解扰 + 原子发布）；
- 经完整性验证的 HLS/MPEG-TS 下载（Layer A/B/C 三层校验）；
- pure-Go TS→MP4 remux，Fast Start（ftyp→moov→mdat），无 ffmpeg 运行时依赖；
- 双语文档与 agent skill 同步。

完整 invariants 见 input.md §86（23 条不可违反项），本计划不得与之冲突。

## 工作区与分支

- 按 `using-git-worktrees` 技能：仓库主树在 `main`，存在既有 `.worktrees/` 惯例目录且已被 gitignore（已验证）。新建 worktree：`.worktrees/assets-media-download`，分支 `feat/assets-media-download`，基于最新 `origin/main`。
- goal 控制文件位于主项目根目录 `goal-2/`（`/goal-*/` 已在 .gitignore）。实现、测试与提交全部在隔离 worktree 中完成，每轮显式使用该路径。
- 旧分支 `codex/download-command-clarity`（对应计划 §72 提到的旧设计）不合并、不修改、不删除；本 goal 从 main 全新交付完整 contract。
- 不使用 `reset --hard`、force push、删除分支；不动其他 worktree 的文件。

## 仓库现状摘要（探子核verified，file:line 供执行轮复核）

1. CLI：命令注册于 `internal/cli/root.go:69-94`；多文件命令域先例 `commands/auth/`、`commands/rankings/`、`commands/lists/`；`Streams{In,Out,Err,InIsTerminal,OutIsTerminal}` 于 `internal/cli/invocation/invocation.go:16-29`；TTY 探测在 root.go:55-56（`golang.org/x/term`）。
2. Pipeline envelope：`internal/cli/pipeline/envelope.go`（Schema="javdb.pipeline/v1"，Kind 白名单 :108-111）。**本功能不加入 envelope**（input §12：不加 KindAsset，pipe 用自定义 `TYPE<TAB>URL` 文本流），因此 assets 命令不使用 `pipeline.BatchRunner` 的 envelope 输出路径，需自写 renderer（`pipeline/output.go:31` `ResolveOutputMode` 可参考 TTY/非 TTY 分流惯例）。
3. 现有下载：顶层 `javdb download NUMBER`（`internal/cli/commands/download/download.go`，flags `--thumbnail/--preview-image/--preview-video`，输出 KindDownload envelope）；SDK `sdk/movie.go:10-58`：`MovieMediaDownloadOptions{ThumbnailPath,PreviewImagePath,PreviewVideoPath}`、`DownloadMovieMedia`。注意：计划 §45 写的 `MovieAssetDownloadOptions` 是笔误，实际符号为 `MovieMediaDownloadOptions`。
4. 资产来源：详情返回 `map[string]any`（`internal/javdb/appapi/endpoint/movie/movie.go:30-47`）；URL 提取逻辑在未导出 `sdk/movie.go:101-134`（`thumb_url`/`preview_video_url`/`preview_images[].large_url||thumb_url`，目前只取 preview_images[0]）。无强类型。
5. Media 现状：`internal/javdb/appapi/media/media.go`（441 行）已有：m3u8 VOD 解析（要求 `#EXT-X-ENDLIST`，拒绝 BYTERANGE/fMP4/master）、AES-128-CBC+PKCS7 解密（sequence IV 缺省）、XOR 图片解扰 + magic 白名单（JPEG/PNG/GIF/WEBP/AVIF/HEIC-like，:73-110）、`O_EXCL` 不覆盖落盘（:403）。`FetchMedia`（`appapi/client/transport.go:133-156`）只带 UA 头、整段 `[]byte` 读入内存（非流式）。
6. 完全缺失：TS 完整性校验（0x47/PAT/PMT/continuity）、TS demux、H.264 AnnexB/AAC ADTS 解析、MP4 mux、fast start、Layer B/C、segment bounded retry、流式下载。
7. 依赖：go 1.26.3（工具链 1.27.1）；直接依赖仅 cobra、tls-client、uuid、toml、x/sys、x/term。标准库 `crypto/aes` 可用。
8. 测试惯例：内联构造二进制、httptest、表驱动；无 fixture 目录；cobra 直接 Execute 断言流输出（`commands/download/download_test.go`）；`internal/cli/root_test.go:31` 锁定根 help 字面量——**新增/删除顶层命令必须同步**。
9. 文档资产：`docs/{en,zh-CN}/cli-reference.md`、`docs/{en,zh-CN}/sdk.md`（另有 `docs/sdk.md`/`docs/sdk.zh-CN.md` 疑似副本，执行时确认同步方式）、`README{,.zh-CN}.md`、`docs/maintainers/architecture.md`、`skills/javdb-cli/SKILL.md` + references。

## 需求拆解（对照 input.md）

- 数据模型：`MovieAsset{Type,URL}` 仅两字段（§2/§46/§77）；`type ∈ {image, video}`（§1.2/§6 invariant 6）。
- 资产序列：thumbnail → cover → preview_images[*] → preview_video，缺失跳过（§4）；preview_images 每项取 `large_url || thumb_url`，不拆成两个 Asset（§5）。
- CLI `assets list`：`--type image|video`、selector `N / N-M / N,M / 混合（空格或逗号分隔）`（§7-8）；filter 先于 selector 编号（§9）；无 `--all`（§10）；DESCRIPTION 仅 TTY 渲染（§6）。
- 四种输出：TTY 表格（§11.1）；非 TTY 默认 `TYPE<TAB>URL`（§11.2）；`--json` 数组（§11.3）；`--ndjson` 行（§11.4）；绝不加入 pipeline envelope/kind（§12）。
- `assets download`：读 stdin 的 `TYPE<TAB>URL` 行；`-d DIR`（默认 `.`）多资产；`-o PATH` 仅允许恰好 1 个资产（§13-14）；默认命名 `image-NNN.<ext>`（ext 由 magic 检测定）/`video-001.mp4`（§15-16）；视频仅 `.ts`/`.mp4` 两种输出，其他报 `unsupported video output format`（§17）；无 `--format` 等参数（§18）。
- 图片链：download → detect → XOR unwrap if needed → detect → validate → atomic publish（§19-20）；不转码不增强（invariant 11）。
- 视频链：HLS download → AES-128 decrypt → Layer A segment integrity（含 bounded retry=3，§32）→ `.ts` 或 demux → H.264/AAC samples → MP4 remux fast start → Layer B/C → atomic publish（§29-§40）；仅 H.264/AAC，其他 codec 明确失败（§24）；无 ffmpeg（§25）；流式 + bounded buffer，RAM 不随文件线性增长（§36）；临时磁盘 ≤2× 输出（§37）。
- SDK：`MovieAssets(ctx, movieID)` 与 `DownloadMovieAsset(ctx, asset, target)`（§45/§47/§48）；删除旧 `MovieMediaDownloadOptions/Result/DownloadMovieMedia`，不加 deprecated wrapper（§49）。
- 下载安全：MediaFetcher 统一处理防盗链头（§21）；仅 http/https、cookie/authorization 不跨 host 传播、日志无 credential（§22）；HLS 范围保持 single-media VOD + MPEG-TS + AES-128（§23）；context 取消贯穿全阶段并清理 temp（§44）。
- 不覆盖已存在文件 + 多文件 preflight（§42-43）；失败无半成品（invariant 20）。
- 排除项（§84）：完整影片下载、磁力/BT、转码、画质增强、live HLS、master playlist、fMP4、Telegram/Hermes 逻辑（§65-68 属独立仓库独立 PR）。
- 文档（§69-71）：双语 README、cli-reference、sdk docs、architecture、skills/javdb-cli；按 workspace AGENTS.md 变更路由执行。changelog 不在本次范围（仅授权 release-prep PR）。

## 依赖 Spike 与审批 Gate（input §27-28）

- 正式实现 MP4 remux 前先做 dependency spike：评估候选 pure-Go 库（至少包括：`Eyevinn/mp4ff`（MIT，MP4 mux/demux + faststart）、`yapingcat/gomedia`（mp4 mux + mpegts demux）、`bluenviron/mediacommon`（mpegts/fmp4/h264/aac）、`abema/go-mp4`（box 读写）），维度：Go 版本、license、维护状态、依赖树、CGO、TS demux/H264/AAC/PTS-DTS/B-frame/MP4 mux/faststart/sample table 正确性。评估结论写入 `goal-2/spike-report.md`。
- 审批 gate：新增任何 Go 依赖必须先报告并获维护者认可（§28）。goal-mode 无人值守、禁止提问，因此：
  - **默认路径**：不引入新依赖，TS demux 与 MP4 mux 用标准库手写（同样是 pure-Go remux，满足 §25/§26，且零审批阻塞）。spike 报告同时产出候选库评估，供维护者后续决定是否切换到库实现。
  - 若 spike 明确证明手写规模不可控（如 B-frame/DTS 重排复杂度超预期），该 task 标记「阻塞」并写明原因，跳到后续非阻塞 task，终审统一汇报。

## 实施方案与 Task 映射

见 `goal-2/tasks.md`（17 个 task，每 3 个普通 task 后一个集中检查 task）。映射 input §74 的 commit 顺序，但允许 test+impl 合并为单个可理解 commit（§75）。执行方式遵循 `superpowers` 的 TDD（Red→Green→Refactor，Red 必须真实运行失败）与逐 task 复核；`using-git-worktrees` 提供隔离。

## 验证方式

- 每 task：先写失败测试（Red）→ 实现（Green）→ 重构；focused test 优先：`go test ./internal/...` 对应包。
- 门禁（检查 task 与终审）：`go test ./...`、`sh scripts/build.sh`、`gofmt -l`、`go vet ./...`。
- 行为验证：cobra 直接 Execute 断言流输出（现有惯例）；httptest 模拟 HLS/图片服务；内联二进制 fixture（不新增大型 fixture 文件，§56-57）。
- Real E2E（§62-64）：`javdb assets list NUMBER`、图片落盘、`-o preview.mp4`、`-o preview.ts`。需真实网络与本机认证；若认证缺失则该 task 标阻塞并汇报，不伪造结果。
- ffprobe 仅作可选冒烟，correctness 不依赖它（§61）。

## 风险与缓解

1. **MP4 remux 规模**（B-frame DTS、sample table、faststart 两阶段）：风险最高。缓解：spike 先行；手写路径按 §35 两阶段（temp spool + sample table → ftyp/moov/mdat）；Layer B/C 测试先行锁定契约；必要时阻塞上报。
2. **breaking SDK 变更**（删除旧 Media API、移除顶层 download 命令）：破坏现有用户脚本。缓解：input §49 明确允许 breaking 且禁止 wrapper；文档同步 + README 迁移示例；在 commit message 中显式标注 breaking。
3. **内存目标**（现有 FetchMedia 整段读入）：需重构为流式。缓解：T9/T10 专门处理；测试用大 segment 断言 bounded buffer。
4. **防盗链头未知**（现有 FetchMedia 无 Referer/Origin，真实 CDN 行未知）：缓解：T16 real E2E 验证；失败时按真实错误补 Referer（内部处理，不暴露 flag，§21）。
5. **root help 字面测试**：删/增顶层命令必须同步 `root_test.go`，否则门禁红。已列入 T5/T13 验收。

## 回滚方案

- 全部工作在隔离 worktree + 独立分支 `feat/assets-media-download`，主仓库 `main` 不受影响；按 task 提交，任一 task 可 `git revert` 单独回退。
- 不覆盖/删除主仓库既有文件；`goal-2/` 目录已 gitignore。
- 若 T11（MP4 remux）阻塞，`.ts` 路径与图片路径可独立交付，MP4 task 标阻塞待维护者决策，不拖着半成品继续。

## 显式假设（无人值守默认值，终审汇报）

1. 计划 §45 的 `MovieAssetDownloadOptions` 实为 `MovieMediaDownloadOptions`（探子确认），按实际符号处理。
2. 依据 invariant 1/2 与 §72（"assets 承担下载、download 作为 alias"不再是 contract），顶层 `javdb download NUMBER` 命令在 `assets download` 能力齐备后**删除**（含其 KindDownload envelope 输出路径与 `--thumbnail/--preview-image/--preview-video` flags），不做 alias、不加 deprecation wrapper。删除前 rg `KindDownload` 全部引用并逐一处理。
3. 依赖默认路径 = 零新依赖手写 remux；引入任何第三方库前必须阻塞等维护者审批（§28），goal 内不得自行添加。
4. Segment retry 上限 3（§32）为计划明确要求，非无依据防御（AGENTS.md 8.2 条款 1）。
5. Real E2E 使用本机现有认证与真实网络；无认证则标阻塞，不伪造。
6. Hermes/Telegram 相关（§65-68）完全不在本仓库执行，仅文档层面确保 javdb-cli 不含 Telegram 逻辑。
7. `docs/sdk.md`/`docs/sdk.zh-CN.md` 若为软链接/副本，随 en/zh 一并同步，方式执行时确认。
8. 版本号不变更、不发版、不写 changelog（workspace AGENTS.md：仅授权 release-prep PR）。
