# PR #47 媒体验收记录

本轮验证日期为 2026-09-17，本轮容器/校验修复的代码版本为 `4e6d097`，已合入 `main` 的
`bd2798a`（#46）。后续本页的证据更新不改变被测生产代码。环境为 macOS
`darwin/arm64`、Go `go1.26.3`、ffmpeg/ffprobe `8.1.1`。
ffmpeg/ffprobe 仅是独立验收工具，不属于 CLI 运行时依赖。

番号以 `NUMBER` 表示；本页不保存媒体、签名 URL、凭据或本机工作目录。

## 列表、过滤与图片（上一轮 `8ee7b64`）

使用真实 JavDB API 分别执行：

```bash
javdb assets list NUMBER
javdb assets list NUMBER --json
javdb assets list NUMBER --ndjson
javdb assets list NUMBER --type image 1-4
javdb assets list NUMBER --type video
javdb assets list NUMBER --type image 1-2 | javdb assets download -d DIR
```

- 本轮样本包含 13 个资产，类型、顺序与过滤后 selector 对应；非 TTY 每行严格为
  `TYPE<TAB>URL`，JSON/NDJSON 每个对象严格只有 `type`、`url`。
- 上游媒体 URL 的 `sign`/`t` 会随请求改变。跨次调用比较类型、scheme、host、path
  与顺序；下载使用该次请求的完整签名 URL，不要求不同请求的签名相同。
- 图片无需手工 Referer/Origin；JPEG magic 与 `sips` 解码检查通过。
  stdout 每行只输出最终文件路径。重复写入同一路径失败，已有文件 SHA-256 不变。
- 本地 HTTP 故障 fixture 提供无效图片与 TS 数据：图片、TS、MP4 下载均非零退出，
  不发布目标文件，目标目录不遗留临时文件。真实下载后的目标目录同样无 `.part`、spool 残留。

## 真实 TS 与 Fast Start MP4

```bash
javdb assets list NUMBER --type video | javdb assets download -o preview.ts
javdb assets list NUMBER --type video | javdb assets download -o preview.mp4
ffprobe -v error -show_format -show_streams -of json preview.ts
ffprobe -v error -show_format -show_streams -of json preview.mp4
ffmpeg -v error -i preview.ts -f null -
ffmpeg -v error -i preview.mp4 -f null -
```

- TS 保留 H.264 720×404、AAC 48 kHz 双声道和输入的 timed ID3；MP4 只写入
  H.264 `avc1` 与 AAC `mp4a`。两种输出均通过完整解码，退出码 0、stderr 为空。
- TS 为 49,052,960 bytes，MP4 为 47,548,685 bytes；MP4 duration 为
  `116.916800s`。逐 box 检查确认 `ftyp → moov → mdat`，moov 在媒体数据之前。
- 视频 3,502 个 packet 的 DTS 全程非递减；PTS−DTS 为 `0..15015`（90 kHz）。
- 从 `ffprobe` duration 动态计算 10%/50%/90% 的 seek 位置，本轮为
  `11.69168s`、`58.45840s`、`105.22512s`；三处读取并解码视频帧成功，stderr 为空。

本轮另按最终文件的 box 布局读取两个 version 0 `tkhd`，matrix 均为：

```text
[65536, 0, 0, 0, 65536, 0, 0, 0, 1073741824]
```

`ffprobe -show_streams` 不再返回无效 Display Matrix side data（单位矩阵被正常省略）。

## 单 segment 与总长度资源实测（上一轮 `8ee7b64`）

所有下载均使用同一台机器的本地 HTTP server，单独以 `/usr/bin/time -l`
记录被测 CLI 的 peak RSS；ffmpeg 解码在下载退出后独立执行，不计入 CLI RSS。
使用已保存的真实 TS fixture：普通 case 将整份 TS 作为一个 HLS segment；
大单段通过 `ffmpeg -stream_loop 11 -c copy -f mpegts` 推进时间戳并重复为约 12 倍；
多段 case 再以 `-f segment -segment_time 120` 切分同一大输入，保持时间戳连续。
生成 fixture 时不转码，由 `8ee7b64` CLI 下载、解析并 remux。
本次未修改 streaming/spool 生命周期，按审查方案保留该轮资源证据，不重复 594 MB 压测。

| 输入 | MP4 输出（MB） | peak RSS（MB） | wall time（s） | full decode |
| --- | ---: | ---: | ---: | --- |
| 普通单段（49.05 MB TS） | 47.55 | 27.48 | 13.20 | PASS |
| 大单段（593.92 MB TS） | 570.57 | 54.89 | 10.82 | PASS |
| 相同大输入切为 12 段 | 570.57 | 59.51 | 10.31 | PASS |

MB 使用十进制 1,000,000 bytes。

三组 MP4 全量解码退出码均为 0，stderr 为空。输出约增长 12 倍，RSS 约增长
2 倍；大单段与相同总长度多段的 RSS 接近，没有观察到 RAM 与单个 segment payload
近似 1:1 增长。RSS 仍包含当前各 PID 的未完成 PES、codec 配置和 MP4 样本元数据；
样本越多，metadata 仍会增长，本结果不表示与样本数无关的常量内存。

这些 wall time 仅记录运行环境中的观测值，不作为吞吐对比或性能承诺。旧记录的
47.5 MB / 569 MB 约为 12 倍而非 10 倍，且旧 CDN 与本地服务器耗时不可横比；
本轮结果取代基于 `e96202b` 的旧资源证据。

## 回归与质量门禁

以下检查在上述代码版本完成：

- `go test ./... -count=1`
- `go test -race ./... -count=1`
- `go vet ./...`
- `sh scripts/build.sh`
- `sh scripts/test-package-release.sh`
- `sh scripts/test-homebrew-formula.sh`
- `sh scripts/test-workflows.sh`
- `python3 -m pre_commit run --all-files`（pre-commit 4.6.0）
- tracked Go files 的 gofmt、`git diff --check`、受影响 Go 文件的 LSP 诊断。

本轮五项回归均先实际执行 Red，再执行 Green：修正已有单位矩阵测试，并检查最终
`tkhd`；同配置的多 H.264 / 多 AAC PID 被 MP4 拒绝；Layer A 合法但 Annex-B/ADTS
损坏的 TS 被拒绝，目标与临时文件无残留；AAC-LC 单 raw data block 通过，其他
profile / 多 block 的 MP4 明确失败且无残留，`channel_configuration=0` 的 MP4
明确失败而 1/2 通过，TS 仍原样保存。

支持边界来自当前 muxer 的单轨、ASC=AAC-LC、mono/stereo channel configuration
和每帧 1024 samples 假设，目的是避免静默合轨与错误 codec 配置/时长；只收窄不支持
的 MP4 输入，不增加大小或数量配额。
旧的 4,200 segment 数量回归夹具重复时间戳；启用 TS Layer B 后已改为每段递增，
段数和字节数断言保持不变。该校验不替代完整 codec 解码，真实媒体仍由 ffmpeg 独立验收。

上一轮补充回归继续保留：segment 未读完时真实 spool 已有 payload；
多 PES 样本数、时间戳与 MP4 容器；大 sample 的有界复制与内容一致；PES 内 AAC
配置变化拒绝；分别传输的 SPS/PPS；按 PID 拒绝声明后无帧的音轨。原有跨 segment
DTS、SPS/PPS 连续性、TS 结构、AES-128 与 MP4 Layer C 测试继续通过。

生产路径不再提供整段 `[]demuxedFrame` 聚合；H.264 每 PES 处理后立即写 spool，
AAC 每 ADTS 帧直接消费借用的 payload，最终 mdat 以 32 KiB I/O 缓冲读取 spool
区间。没有新增媒体总量、segment 大小/数量、样本数量等硬配额；既有格式与协议
表达边界继续显式报错。

最终 PR HEAD 的 GitHub Actions 状态见 PR 检查页；仅以该 HEAD 的 Quality gate、
Real API e2e、Container image smoke 与 Platform packaged binary smoke 为合并依据，
不沿用旧 base/HEAD 的绿色结果。
