# goal-3 plan:javdb-cli 资产发现、下载与 Progressive MP4

## 0. 权威来源与优先级

- `goal-3/input.md`:用户提供的完整实施计划(2684 行原文 + 命令行),是唯一 contract 权威。本 plan 与其冲突时,以 input.md 为准。
- 本 plan 只负责:现状映射(真实代码位置)、计划与现实的差异假设、执行编排、风险与回滚。
- 执行采用 superpowers 方法的 TDD 原则(Red → Green → Refactor,每 task 测试先行),受 goal-mode 无人值守节奏约束(一轮一个 task,不能提问,假设必须落盘)。

## 1. 现状分析(2026-09-13 探索结论,file:line)

### CLI(internal/cli)
- 根命令 `internal/cli/root.go:46` `New(stdin, stdout, stderr)`;子命令注册于 root.go:69-94。
- **已存在顶层 `download` 命令**:`internal/cli/commands/download/download.go:20`,`Use: "download NUMBER"`,flags `--id/-i`、`--thumbnail`、`--preview-image`、`--preview-video`、`--json`、`--ndjson`;依赖 SDK `MovieMediaDownloadOptions`。`assets` 命令不存在。
- TTY 探测:root.go:50-56(`x/term.IsTerminal` → `streams.InIsTerminal/OutIsTerminal`,定义于 internal/cli/invocation/invocation.go:23,28)。
- 输出模式:`internal/cli/pipeline/output.go:31` `ResolveOutputMode(flagNDJSON, flagJSON, outIsTerminal)`;模式 OutputAuto/Human/Text/NDJSON/JSON。
- envelope:`internal/cli/pipeline/envelope.go:16` Schema="javdb.pipeline/v1",已有 `KindDownload="download"` 等常量。input.md #12 明确:assets 域不加入该协议,pipe 用 `TYPE<TAB>URL` 极简文本流。

### SDK(sdk/,package javdb)
- `sdk/movie.go:10-14` `MovieMediaDownloadOptions{ThumbnailPath, PreviewImagePath, PreviewImagePath, PreviewVideoPath}`;:17-24 `MovieMediaDownloadResult{...Bytes}`;:47 `DownloadMovieMedia(ctx, movieID, opt)`。
- `sdk/movie.go:101` `movieMediaURLs`:取 `thumb_url`、`preview_video_url`、`preview_images[0].large_url`(回退 `thumb_url`),仅第一张预览图。
- 影片详情返回 `map[string]any`(internal/javdb/appapi/endpoint/movie/movie.go:25),**无 typed struct**;`cover_url` 在全仓库从未被引用。
- 公开 API 签名被 `sdk/contract_external_test.go:50` 与 `sdk/facade_test.go` 冻结,改动需同步更新。

### media(internal/javdb/appapi/media/media.go)
- `type Fetch func(string) ([]byte, error)`(:113);`MediaEndpoint{fetch}`(:18-25)。
- HLS:`parseHLSMediaPlaylist`(:182-245)支持 EXTM3U/ENDLIST/MEDIA-SEQUENCE/EXT-X-KEY,显式拒绝 BYTERANGE、EXT-X-MAP(fMP4)、master playlist;AES-128 完整(KEY 解析 :247-275、IV :328-344、缺省 sequence IV :362-366、CBC+PKCS7 :368-401、key 缓存 :142-159)。
- **无 MPEG-TS 解析**:解密后 segment 原样拼接落盘。
- 图片:`decodeImagePayload`(:73-91)已实现 XOR 解包(首字节 key)+ `knownImagePayload`(:93-110)魔数校验(JPEG/PNG/GIF/WEBP/AVIF/HEIC)。
- `writeNewMediaFile`(:403-441)`O_EXCL` 不覆盖已有文件,失败清理。
- Fetch 实现于 `internal/javdb/appapi/client/transport.go:133-156` `FetchMedia`:只设 `user-agent`(`model.UserAgent = "Dart/3.4 (dart:io)"`);**无 Referer/Origin**;认证 header(jdsignature/Bearer)只进签名 API 请求(:158-169, :218),不进媒体请求。URL 校验:transport.go:134-140、media.go:61-70 `validateMediaURL`、:346-360 `resolveHLSURI`。
- 注意:httpx.Client 共享全局 CookieJar(internal/javdb/protocol/httpx/client.go:35-40),API Set-Cookie 理论上可能被重放到媒体 host——实现 media fetcher 时保持媒体请求 header 白名单化。

### 测试与工作区
- 测试与包同目录 `*_test.go`,httptest 广泛使用,无 testdata/golden(全内联 fixture + `t.TempDir()`);media_test.go 5 例(XOR、非 2xx、AES sequence IV、未完成 playlist、坏 PKCS7、不覆盖)。
- E2E:`e2e/run.sh`,需 `JAVDB_E2E_USERNAME/PASSWORD`,缺省 SKIP。
- 本机 git:main@642cb06;`.worktrees/` 已存在且被 gitignore;旧分支 `codex/download-command-clarity` / `origin/refactor/download-command-clarity` 存在,其设计已被 input.md #72 取代(不合并)。

## 2. 计划与现实的差异及默认假设(无人值守,不可再问)

- **A1 符号名(T01 修正)**:PR #45(1ad3011)已合入最新 main:旧 `MovieMedia*` 已更名为 `MovieAssetDownloadOptions/MovieAssetDownloadResult/DownloadMovieAssets`(sdk/movie.go),CLI 已有单体 `assets NUMBER` 命令(alias `download`,internal/cli/commands/download/,path-per-type flags --thumbnail/--preview-image/--preview-video)。input.md #45 所指的旧 API 即这套。迁移 = 删除该 path-per-type API 与单体 assets 命令,新增 `MovieAsset{Type,URL}` + `MovieAssets` + `DownloadMovieAsset`(新的 `javdb assets list|download` 子命令域)。
- **A2 cover_url**:JavDB 详情 map 中从未出现。资产读取顺序按 input.md #4:thumbnail(`thumb_url`)→ cover(`cover_url`,map 存在才纳入)→ `preview_images[]`(large_url 回退 thumb_url)→ `preview_video_url`;缺失项直接跳过。真实字段形态以 E2E 输出为准。
- **A3 防盗链 header**:现状仅 UA 已能拉取部分媒体。MediaFetcher 统一内部管理 image/playlist/segment/key 请求 header(input.md #21),实际 header 组合在真实 E2E 阶段按结果确定;不预先暴露 `--media-header`。
- **A4 旧 `javdb download` 与单体 assets 命令**:input.md invariant 1/2 要求下载只存在于 `javdb assets download` 子命令。PR #45 的单体 `assets NUMBER`(alias `download`,目录 internal/cli/commands/download/)与 path-per-type SDK 在 T03 一并删除,保持每 task 结束时编译+测试全绿。
- **A5 依赖策略(input.md #27/#28)**:默认**零新依赖**,TS demux/PES/NALU/ADTS/MP4 mux 用标准库自研(encoding/binary + 已有 crypto/aes)。spike task 仍完成候选库(go-astits、go-mp4 等)调研并写 `goal-3/spike-remux.md` 供维护者审阅;若自研被证不可行,后续 task 标阻塞,不擅自引依赖。
- **A6 分支与隔离**:已检测主 repo(main,GIT_DIR==GIT_COMMON,非 submodule),无原生 worktree 工具 → git worktree fallback:`.worktrees/feat/assets-media-download`(分支 `feat/assets-media-download`,基于最新 main)。不合并、不 rebase 旧分支(input.md #72/#73)。
- **A7 E2E 凭证**:若本机无有效 JavDB 认证,E2E task 标阻塞并记录所需输入,fixture 测试先行覆盖核心契约,不阻塞其他 task。
- **A8 commit 规范**:遵循 input.md #74 的英文 conventional commit 序列;test 与实现紧耦合时可合并为逻辑 commit(#75)。

## 3. 执行方案

- 隔离:using-git-worktrees 流程(检测结果见 A6;`.worktrees` 已被 gitignore,无需追加)。
- 每轮只做 tasks.md 第一个未完成 task;每 3 个执行 task 后接一个集中检查 task(CHECK)。
- 实现 locus:CLI 于 `internal/cli/commands/assets/`(assets.go/list.go/download.go/selector.go);media 于 `internal/javdb/appapi/media/`(TS/PES/NALU/ADTS/MP4 不出该包,input.md #51);SDK 于 `sdk/movie.go`(或新 `sdk/asset.go`)。
- PR #45 的单体 assets 命令(目录 `internal/cli/commands/download/`)在 T03 删除;新 `assets` 命令域(list/download 子命令)T04/T05 新建于 `internal/cli/commands/assets/`。envelope/protocol 基础设施不动(input.md #12)。

## 4. 验证方式

- 每 task 收尾:`go test ./...`(仓库 AGENTS.md 默认离线验证)+ 受影响包 `go vet`;构建 `sh scripts/build.sh`。全量门禁留给 CHECK/终审轮,避免 fail-fast 重复跑。
- MP4 契约校验用自研 Go 校验器(Layer C,input.md #61:测试 correctness 不依赖 ffprobe);若环境存在 ffprobe 仅做可选 smoke。
- E2E(input.md #62-64):`e2e/run.sh` 或手工命令;验证真实资产数量/类型、图片可打开、TS 完整、MP4 moov<mdat、duration>0、可 seek。

## 5. 风险与回滚

- **R1 remux 正确性**(B-frame、PTS/DTS 重排、AVCC 转换):Layer B 时间戳校验 + Layer C 容器校验兜底;fixture 覆盖多 sample、乱序 DTS、video-only。
- **R2 内存有界**(input.md #36/#81):网络流式 + 有界 segment 缓冲 + spool 落盘 + sample metadata;禁止整视频进 RAM。
- **R3 凭证泄漏**(input.md #22):媒体请求 header 白名单化,Authorization/Cookie 不随 playlist 跨 host 传播;日志不打印 credential。
- **R4 E2E 凭证缺失**:A7 阻塞路径,记录后跳到下一未阻塞 task。
- **R5 与既有 pipeline 协议冲突**:assets 域完全独立于 javdb.pipeline/v1,不新增 KindAsset。
- **回滚**:全部工作在 worktree 分支,main 不受影响;整体放弃 = 移除 worktree + 删除分支;单 task 失败 = worktree 内 revert 对应 commit。

## 6. 明确不做(遵循 input.md #84 与 #86)

完整影片下载、磁力/BT、转码画质增强、live HLS、master playlist、fMP4 input、HEVC、Telegram 逻辑(input.md #65-68 属 Hermes 独立仓库,不在本 goal)、任何新 Go 依赖、`javdb download` 顶层命令、`--all`/`--preview`/`--format` 等参数。
