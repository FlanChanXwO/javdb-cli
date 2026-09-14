# goal-3 tasks

节奏:一轮一个 task,按序执行。每个 task 完成后回写四行:实际做 / 证据 / 剩余风险 / 下一步。
TDD:执行 task 一律测试先行(Red → Green → Refactor);每 task 收尾 `go test ./...` 必须全绿。

---

## T01 [x] worktree 隔离与基线验证
- 目标:按 using-git-worktrees 流程创建 `.worktrees/feat/assets-media-download`(分支 `feat/assets-media-download`,基于最新 main),`go mod download`,基线 `go test ./...` 与 `sh scripts/build.sh` 全绿。已在主 repo(main)检测过:非 worktree、非 submodule、`.worktrees` 已 gitignore、无原生工具 → git worktree fallback。
- 验收:worktree 存在、分支正确、基线测试/构建通过并记录通过用例数;goal 文件仍在主 repo 根目录(绝对路径引用)。
- 实际做:fetch 发现 origin/main 已前进 642cb06→1ad3011(PR #45 "clarify local movie asset downloads" 已合入:SDK 旧 MovieMedia* 更名为 MovieAssetDownloadOptions/Result/DownloadMovieAssets,CLI 改为单体 `assets NUMBER` + `download` alias,仍为 path-per-type 模型)。worktree 基于最新 origin/main(1ad3011)创建。
- 证据:`git worktree add` 成功;worktree 内 `go test ./...` 71 包 ok、0 FAIL;`sh scripts/build.sh` 构建成功。
- 剩余风险:探索子代理基于旧 main,plan.md A1 假设已修正(见 plan.md);PR #45 的 assets 单体命令与目标 contract 冲突,由 T03/T04 取代。
- 下一步:T02。

## T02 [x] test+refactor(assets): 最小资产契约(sdk 发现层)
- 目标:TDD 定义 `MovieAsset{Type string; URL string}` 与 `Client.MovieAssets(ctx, movieID) ([]MovieAsset, error)`:从详情 map 读取 thumb_url → cover_url(存在才纳入)→ preview_images[](large_url 回退 thumb_url)→ preview_video_url,缺失跳过;Type 只有 "image"/"video"。先写失败测试(多预览图、缺失项、fallback),再实现。
- 验收:新契约测试覆盖 thumbnail/cover/多 preview/视频/缺失/large→thumb 回退;`MovieAsset` 无任何多余字段(对照 input.md #2/#46)。
- 实际做:新增 sdk/asset.go(MovieAsset、MovieAssets、movieAssetsFromDetail、moviePreviewImageURLs)+ sdk/asset_test.go 7 个用例(顺序、缺失跳过、无 URL preview 跳过、malformed preview_images 容错、空 slice 非 nil、httptest 接线、错误传播)。
- 证据:Red 确认(undefined 符号 build fail)→ Green:`go test ./sdk/` ok;全仓库 `go test ./...` 无 FAIL;gofmt/vet 通过;commit feat(sdk) 已入分支。
- 剩余风险:cover_url 真实 API 是否返回待 E2E(T12)确认,缺失即跳过不影响。
- 下一步:T03。

## T03 [ ] feat(sdk)+chore(cli): DownloadMovieAsset 与旧 API 移除
- 目标:实现 `Client.DownloadMovieAsset(ctx, asset MovieAsset, target string) (int64, error)`(视频 target 仅 .ts/.mp4,否则 `unsupported video output format ".mkv"`;图片走 media 验证链)。删除 `MovieMediaDownloadOptions/Result/DownloadMovieMedia` 与旧顶层 `javdb download` 命令(`internal/cli/commands/download/` + root.go 注册),同步更新 sdk/contract_external_test.go、sdk/facade_test.go 及相关测试。旧命令删除先于新 assets 命令(invariant 1)。
- 验收:`go build ./...` + `go test ./...` 全绿;全仓库无 MovieMediaDownloadOptions 残留引用;无顶层 download 命令。
- 实际做:待填
- 证据:待填
- 剩余风险:待填
- 下一步:待填

## CHECK1 [x] 集中检查(每 3 task)
- 检查:需求未偏离 input.md;无死代码/调试残留;`go vet`、`go test ./...`、`sh scripts/build.sh`;文档边界是否被破坏(此时应无文档承诺);安全(旧 API 删除无残留、无凭证泄漏);回滚方案是否需调整。
- 结论:全绿。全量 `go test ./...` 无 FAIL;构建成功;`go vet ./...` 干净;全仓库无 MovieAssetDownloadOptions/Result/DownloadMovieAssets/旧单体命令残留;无 TODO/FIXME/fmt.Println 调试残留。root_test.go 中保留的 download 引用均为新 invariant 断言(顶层 download 不恢复、unknown command 不落盘),符合 input.md #86。
- 发现的问题:无。
- T01-T03 期间 commit:feat(sdk) expose type-url movie assets / feat(sdk) add DownloadMovieAsset / refactor(sdk/cli) remove path-per-type asset API。

## T04 [x] feat(cli): javdb assets list(筛选 + selector + 四种输出)
- 实际做:internal/cli/commands/assets/(assets.go 父命令、list.go、selector.go、download_test.go 同包);root.go 注册;SDK 新增 MovieAssetsFromDetail/MovieAssetDescriptions 同序映射(单次详情请求,TITY 描述不进机器输出)。selector 位置参数 args[1:] 拼接,兼容 "1-4"/"1,3-5"/"1 3 5"。--type 校验仅 image|video;JSON/NDJSON 严格只有 type/url;TTY 表格动态编号宽 + TYPE 列对齐。
- 证据:Red(undefined NewList)→ Green;9 个 list 测试 + selector 表驱动测试;全量 go test 71 包 ok。
- 剩余风险:TTY 表格列宽格式为本次定义(计划示例未精确规定),T13 文档按实现描述。
- 下一步:T05(依赖 T06 的扩展名检测,故先执行 T06)。

## T05 [x] feat(cli): javdb assets download(pipe consumer)
- 实际做:internal/cli/commands/assets/download.go + download_test.go:stdin TYPE<TAB>URL 记录流(空行跳过、坏行带行号报错、未知 type 明确拒绝);-d DIR(默认 .)/-o(仅恰好一个资产,否则 `-o requires exactly one asset`);图片自动命名走临时文件→magic 检测扩展名→冲突检查→rename 发布(image-NNN.ext);视频自动命名 video-NNN.mp4(remux 层 T10 前对 .mp4 拒绝,文案与最终契约一致,.ts 可用);绝不覆盖已有文件。SDK 新增 ImageAssetFormat(读文件头转发 media 检测)。
- 证据:Red(undefined NewDownload)→ Green;11 个 download 测试;全量 71 包 ok;build 成功;commit 已入分支。
- 剩余风险:视频自动命名端到端依赖 T10 的 MP4 remux;T05 阶段 video 默认命名会报 unsupported(计划内中间态,#18 的自动 .mp4 契约不变)。
- 下一步:CHECK2 → T07。

## T06 [x] refactor(media): 统一 MediaFetcher 与图片验证链(提前执行)
- 实际做:(执行顺序调整:图片默认命名依赖 magic→扩展名检测,该能力属本 task,故 T06 先于 T05 执行,已注明)。knownImagePayload 重构为 imagePayloadFormat 并导出 ImagePayloadFormat(格式名 jpg/png/gif/webp/avif/heic);media_image_test.go 表驱动失败 payload(HTML/JSON/Cloudflare/空/未知二进制)不落盘 + fetch 错误传播 + XOR 正反路径;安全边界固化测试:带 token 的 client 发起媒体请求不携带 Authorization/jdsignature(cookie jar 为标准 per-domain 语义,不跨 host 传播,未新增无据改动)。
- 证据:Red(undefined ImagePayloadFormat)→ Green;media 包测试全绿;全量无回归;commit refactor(media) 已入分支。
- 剩余风险:FetchMedia 仅设 UA——真实 CDN 是否要求 Referer 待 T12 E2E 确认(计划 #21 允许按真实结果内部添加)。
- 下一步:T05。

## CHECK2 [x] 集中检查(每 3 task)
- 结论:全绿。全量 71 包 ok、0 FAIL、vet/build 通过。assets 域无 envelope/schema/kind 蠕变(rg 验证);MovieAsset 结构严格只有 Type/URL 两字段;media 请求 header 白名单(UA only)且有行为固化测试,无凭证泄漏路径;selector 边界(0/-1/1-/4-1/foo/1.5/区间内空白)全部拒绝有测试。
- 发现的问题:无。
- T04-T06 commit:feat(cli) assets list / refactor(media) image validation / feat(cli) assets download。

## T07 [x] feat(media): Layer A — HLS segment 完整性与 .ts 输出
- 实际做:新增 ts_validate.go(validateTSSegment:188 对齐/0x47/单包 PSI PAT+PMT/PES 前缀/截断;cc 不校验依据 #31;两遍扫描避免 PMT 晚于首包误报)+ publishMediaFile(.part 临时文件→validate→rename,存在不覆盖);downloadHLS 逐 segment fetchValidatedSegment(失败重试同一 segment 至多 3 次,依据 #32,耗尽报 `segment N remained invalid after 3 attempts`);.ts 拼接完成后对整文件 validateTSFile 二次校验(#33)。表驱动测试覆盖 #58 全部条目 + retry success/exhausted + discontinuity + 无 .part 残留。下游 sdk/CLI fixture 更新为合法 TS。
- 证据:Red(行为缺失:garbage segment 曾被照单全收)→ Green;调试中发现并修复 PAT 项长与循环条件不匹配 bug(曾致死循环,已被 -timeout 捕获);全量 71 包 ok + vet;commit feat(media) 已入分支。
- 剩余风险:PSI 跨 packet/多 section 不支持(预览视频场景单包足够,#23 focused 范围);ctx 贯穿(#44)推迟到 T10 与 MP4 finalize 一并处理。
- 下一步:T08 spike。

## T08 [x] [spike] pure-Go remux 路线决策(零新依赖默认)
- 实际做:经 proxy.golang.org 实查三个候选库最新版本与 go.mod 依赖树(go-astits v1.16.0/2026-08、abema/go-mp4 v1.7.3/2026-09、Eyevinn/mp4ff v0.56.0/2026-08;均 MIT/纯 Go/无 CGO/活跃);评估维度对照 input.md #27。决策:自研零依赖(唯一不阻塞 #28 审批 gate 的路径;demux 半壁已在 T07 落地),Eyevinn/mp4ff 记录为后备(若自研受阻,作为独立审批项申报,不擅自引依赖)。
- 证据:goal-3/spike-remux.md 落盘;git status 确认 go.mod/go.sum 未修改。
- 剩余风险:B 帧时间戳处理(见报告 R-BFrame,Layer B/C 兜底)。
- 下一步:T09。

## T09 [x] feat(media): TS demux + Layer B 媒体完整性
- 实际做:demux.go(parseTSStreams:TS→PES 重组按 PUSI 分界、PES header PTS/DTS 提取;parsePMTTypes 直接产出 PID→stream_type);tracks.go(H.264 Annex-B→AVCC 4 字节长度前缀,SPS/PPS 剥离进参数集;AAC ADTS→raw 帧+AudioSpecificConfig,AAC-LC/频率表/通道数;splitAnnexBNALs 容忍 3/4 字节起始码);validateMediaStream = Layer B 入口(HEVC/AC-3/E-AC-3 等按 codecNames 明确拒绝、单一 H.264 轨、ParamSets≥2、DTS 单调、duration>0、timed ID3 丢弃)。.ts 发布前最终校验从 validateTSSegment 升级为 validateMediaFile(媒体模型级)。fixture 全面升级为完整媒体模型(两段拼接用时间基准参数避免 PTS 回退误报)。
- 证据:Red(HEVC 流曾被 .ts 发布)→ Green;fixture 调试修 3 处(PES header 布局、ADTS 长度位、PAT 项长循环);全量 71 包 ok + vet;commit feat(media) 81d3c78。
- 剩余风险:HLS 每帧一个 PES 的约定是样本划分前提(真实 JavDB 流待 E2E 验证);Layer B 中段缺失 SPS/PPS(仅首帧带)场景已覆盖(avcC 取自全流扫描)。
- 下一步:CHECK3。

## CHECK3 [x] 集中检查(每 3 task)
- 检查清单:需求偏离 input.md?死代码/调试残留?类型/构建/测试?安全?限制依据?
- 结论:全绿。全量 71 包 ok、vet 干净、build 成功。`go vet` 无输出;gofmt 无未格式化文件。Layer A/B 的限制均有 input.md 明文依据(retry=3→#32;PSI 单包→#23 focused;codec 拒绝→#24;ID3 丢弃→#39),无无据防御(AGENTS §8 自查通过)。spike 后 go.mod/go.sum 零改动(git diff 确认)。安全:媒体请求 header 白名单测试仍在;无 credential 日志路径。
- 发现的问题:无阻塞。一个待办确认:真实 JavDB 预览流是否每帧一个 PES(T12 E2E 验证)。
- T07-T09 commit:feat(media) 81d3c78(validate HLS segments and media stream integrity)。

## T10 [x] feat(media): MP4 mux Fast Start + Layer C + atomic publish
- 实际做:mp4.go(box 构造/mp4TrackMeta/planVideoTrack/planAudioTrack/buildMoov/buildMP4 内存便捷路径/SPS Exp-Golomb 解析宽高含 high profile scaling list 与 crop);mp4_stream.go(mp4Spooler 逐 segment spool、writeMP4Body 两阶段 ftyp→moov→mdat、copySamplesFromSpool、validateMP4File=Layer C 递归 box 校验、validateTSFileStream 分块结构校验)。avc1/avcC、mp4a/esds、stts/stss/ctts/stsc/stsz/stco;时间轴首样本零基准。ctx 贯穿:Fetch→FetchContext(transport/appapi/media/sdk 全链),取消即断网且不落盘。调试修复 6 处(PES stuffing 截断、fullbox 头序、mdatStart 数据区语义、stco count 偏移、stsd entry 偏移、spool O_RDWR/mdat 头长度)。
- 证据:Red(MP4 尚未实现,测试引用未定义符号)→ Green;media 包 22+ 用例含端到端 HLS→MP4 与 ctx 取消;全量 71 包 ok、vet、build、gofmt hook 通过;commit a171be5。
- 剩余风险:Layer C 未校验 stss 边界与 ctts(结构性跳过);真实流 E2E 待 T12。
- 下一步:T11。

## T11 [x] test: 全链路集成与核心测试收敛
- 实际做:补视频自动命名 video-001.mp4 端到端测试(T10 remux 后接通,断言 ftyp 与 moov<mdat);补 assets list 输出直连 assets download stdin 的真管道组合测试;list 测试改用可替换 server URL(契约不耦合端口)。
- 证据:`go test ./...` 71 包 ok、vet、gofmt、build 全绿;commit 97fe33a。
- 剩余风险:无。
- 下一步:T12。

## T12 [x] [E2E] 真实资产验证
- 实际做:本机已有有效认证(auth.json),E2E 全部跑通:① `assets list NUMBER` 返回 13 图片+1 视频(顺序 thumbnail→cover→preview×11→video,cover_url 真实存在,A2 假设验证);② 图片 `--type image 1-2 | download -d`:image-001/002.jpg,`file` 确认合法 JPEG;③ MP4 `--type video | download -o preview.mp4`:47.5MB,ffprobe=720x404 h264+48kHz 立体声 aac,duration 117.05s,moov<mdat(Fast Start),ffmpeg 全片解码零错误(仅 null muxer 的 B 帧 DTS 重排警告 5 行,属 ffmpeg 对 ctts 的展示性警告);④ TS `-o preview.ts`:49MB,解码零错误。E2E 暴露并修复 4 个真实流问题(PMT program descriptor 跳过、ID3 流豁免、AAC 多帧切分、spool/stco 布局语义)。
- 证据:commit 2a00dcd;产物在 /tmp/assets-e2e/;`go test ./...` 71 包 ok。
- 剩余风险:B 帧流的 null-muxer DTS 警告为 ffmpeg 展示性行为,播放器不受影响;seek 已由 moov 前置+完整 sample table 支持。
- 下一步:CHECK4。

- 目标:按 #52-#61 收敛:少量 deterministic fixture(JPEG/PNG/XOR 图 + 一个小 H.264/AAC VOD fixture,AES case 测试内加密)+ table-driven;list 契约(#53)、输出契约(#54)、download(#55)、segment(#58)、MP4(#59/#60)核对补缺;`go test ./...` + `go vet ./...` + `sh scripts/build.sh` 全绿;不加 ffprobe 依赖(#61)。
- 验收:全量绿;无重复臃肿测试;fixture 数量受控(无多个大二进制 fixture)。
- 实际做:待填
- 证据:待填
- 剩余风险:待填
- 下一步:待填

## CHECK4 [x] 集中检查(每 3 task)
- 结论:invariant #86 全部 23 条逐条对照通过(rg/代码核验):①顶层 download 无注册;②下载仅在 javdb assets download;③MovieAsset 仅 Type/URL;④无 schema/kind/role/id/index 字段;⑤编号为位置语义;⑥--type 仅 image|video;⑦⑧无 --preview/--all;⑨⑩pipe 默认 TYPE<TAB>URL 且无需 --ndjson;⑪图片零转码;⑫防盗链内部且未暴露 --media-header;⑬视频自动命名 .mp4;⑭.ts 显式保留;⑮Fast Start(E2E moov<mdat 证实);⑯⑰代码零 ffmpeg/ffprobe 引用、无转码;⑱三层完整性(publishMediaFile/validateMP4File/Layer C);⑲spool 落盘内存有界;⑳取消贯穿(FetchContext)且 E2E 无半成品;㉑go.mod/go.sum 零改动(spike 报告在案);㉒㉓无 Telegram 逻辑。全量 71 包 ok+vet+build。
- 发现的问题:无。
- T13/T14 文档范围确认:README×2、cli-reference×2、sdk×2、architecture、skills/javdb-cli/。

## T13 [x] docs: 文档更新(双 locale)
- 实际做:README.md/README.zh-CN.md 示例改为 assets list→download 管道;cli-reference 双语重写 assets 命令域(list 处理顺序/selector 语法/四种输出/download -d -o 自动命名/不覆盖已有文件);sdk.md 双语更新为 MovieAssets/DownloadMovieAsset/MovieAsset 最小契约(含破坏性变更说明);architecture.md 记录三层完整性/纯 Go remux/独立 TYPE<TAB>URL 协议。
- 证据:commit f44961f;rg 确认无旧 API 文档残留(en/zh sdk 中的"已移除"说明为有意保留)。
- 剩余风险:无。
- 下一步:T14。

## T14 [x] docs(skill)+ 终审
- 实际做:skills/javdb-cli/SKILL.md 与 references/discover.md 更新为两步工作流(先 assets list 看资产 → selector/--type 过滤 → 管道到 assets download),替换旧 path-per-type 用法。终审=最大范围总复查(见下)。
- 证据(终审清单):①input.md #86 全部 23 条 invariant 逐条 rg/代码核验通过(CHECK4);②全量 `go test ./...` 71 包 ok、`go vet ./...` 干净、gofmt 无未格式化、`sh scripts/build.sh` 成功;③E2E 冒烟复验:--json 输出仅 type/url、图片 magic 命名、视频自动命名 video-001.mp4 为 ISO BMFF 且 moov<mdat;④提交序列 13 个 commit 语义完整(feat/refactor/test/docs/fix,符合 input.md #74 精神,#75 允许逻辑合并);⑤零新依赖(go.mod/go.sum 未动,spike-remux.md 在案);⑥回滚路径:全部工作在 worktree 分支 feat/assets-media-download,main 未受影响。
- 证据:commit ac31a31。
- 剩余风险(低):真实流的 null-muxer DTS 警告为 ffmpeg 展示性行为;cover_url 依赖真实 API 返回(缺失自动跳过)。
- 下一步:goal 完成,交付 PR 由维护者决定。

## T13 旧条目(已被上方取代)
- 目标:README.md/README.zh-CN.md(用户主路径 #70)、docs/en/cli-reference.md 与 docs/zh-CN/cli-reference.md(assets 命令域)、docs/en/sdk.md 与 docs/zh-CN/sdk.md(MovieAssets/DownloadMovieAsset/MovieAsset)、docs/maintainers/architecture.md(assets 命令域 + media 分层 + 零依赖决策)。
- 验收:两 locale 内容一致;示例与真实行为一致;移除旧 download 命令的文档残留。
- 实际做:待填
- 证据:待填
- 剩余风险:待填
- 下一步:待填

## T14 [ ] docs(skill)+ 终审
- 目标:更新 skills/javdb-cli/SKILL.md 与 references/*(agent 工作流:先 assets list → 选择 → pipe 到 assets download,#70/#71);终审=最大范围总复查:从 input.md #86 的 23 条 invariant 逐条对照、代码质量、安全、错误处理、测试覆盖、构建产物、文档一致性、回滚说明;发现问题当场修复或追加修复 task;最后汇报 goal 完成状态(含阻塞项清单,如有)。
- 验收:终审清单逐条记录于本文件;全量验证绿;goal 状态明确(完成/存在阻塞项)。
- 实际做:待填
- 证据:待填
- 剩余风险:待填
- 下一步:待填

---

## 阻塞登记
(无)
