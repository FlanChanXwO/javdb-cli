/goal  使用 [$using-git-worktrees](/Users/flanchan/.zcode/skills/using-git-worktrees/SKILL.md) 技能， [$goal-mode](/Users/flanchan/.zcode/skills/goal-mode/SKILL.md) 技能，  [$superpowers](/Users/flanchan/.zcode/skills/superpowers-skill/SKILL.md) 技能完成计划

---
以下为用户随消息附上的完整计划文档（逐字保留）：

# javdb-cli 资产发现、下载与 Progressive MP4 完整实施计划

## 0. 最终目标

重新定义 JavDB 影片资产能力，使：

```text
影片详情
   ↓
assets list
   ↓
筛选 / 选择
   ↓
shell pipeline
   ↓
assets download
   ↓
图片 / TS / Progressive MP4
```

用户不需要理解：

```text
preview_images[0]
large_url
thumb_url
preview_video_url
HLS segment
AES-128
MPEG-TS
PES
Annex-B
AVCC
moov
mdat
Referer
Origin
```

这些都属于实现细节。

最终 CLI 以：

```text
javdb assets
├── list
└── download
```

作为唯一资产命令域。

明确禁止重新增加顶层：

```bash
javdb download
```

因为：

```text
javdb download NUMBER
```

天然容易被理解成：

> 下载完整影片。

这与实际“下载影片附属媒体资产”的能力不一致。

---

# 1. 最终 CLI Contract

## 1.1 查看资产

```bash
javdb assets list NUMBER
```

TTY 输出示例：

```text
#   TYPE    DESCRIPTION
1   image   thumbnail
2   image   cover
3   image   preview 1
4   image   preview 2
5   image   preview 3
6   video   preview
```

其中：

```text
#
DESCRIPTION
```

都只是 CLI 人类输出。

它们不是 Asset 数据模型字段。

---

## 1.2 按类型筛选

只支持真正有意义的媒体类型：

```bash
javdb assets list NUMBER --type image
```

或：

```bash
javdb assets list NUMBER --type video
```

不增加：

```text
--preview
--role
--kind
--asset-kind
--preview-image
--preview-video
```

等筛选参数。

`type` 只有：

```text
image
video
```

---

# 2. Asset 数据模型必须保持极小

资产领域唯一需要长期维护的数据：

```go
type MovieAsset struct {
    Type string
    URL  string
}
```

仅此两个字段。

禁止引入：

```text
Schema
Kind
Role
ID
Index
AssetID
AssetRole
AssetKind
PreviewImage
PreviewVideo
Description
Name
Source
```

作为 Asset model 字段。

---

# 3. 为什么只有 type + url

下载器真正需要的信息只有：

```text
type
↓
决定 image / video 下载路径

url
↓
决定从哪里获取数据
```

其余：

```text
第几项
是不是 thumbnail
是不是 cover
是不是 preview
preview 的第几张
```

都可以从：

```text
MovieDetail 原始结构
+
当前列表位置
```

得到。

不应该为了这些信息长期维护新的 domain metadata。

---

# 4. 资产来源

从影片详情读取：

```text
thumb_url
cover_url
preview_images[]
preview_video_url
```

转为资产序列。

顺序固定为：

```text
thumbnail
cover
preview_images[0]
preview_images[1]
preview_images[2]
...
preview_video
```

缺失项直接跳过。

例如没有 cover：

```text
thumbnail
preview_images[0]
preview_images[1]
preview_video
```

---

# 5. Preview image URL 选择

`preview_images[]` 每项内部可能同时包含：

```text
large_url
thumb_url
```

资产列表中不要把它们拆成两个 Asset。

它们本质是同一张图片的不同来源。

内部策略：

```text
large_url available
    ↓ yes
use large_url

    ↓ no

thumb_url available
    ↓ yes
use thumb_url
```

因此：

```go
MovieAsset{
    Type: "image",
    URL:  selectedURL,
}
```

---

# 6. DESCRIPTION 只是人类渲染

TTY 可以输出：

```text
thumbnail
cover
preview 1
preview 2
preview
```

但这些文本不得进入：

```text
MovieAsset
SDK JSON
pipe record
download input
持久化状态
```

它们只在：

```text
assets list TTY renderer
```

生成。

不要建立：

```go
Role string
Description string
```

再传来传去。

---

# 7. Selector 语法

`assets list` 支持：

```text
1
1 3 5
1-4
1,3,5
1-3,5
1 3-5
```

例如：

```bash
javdb assets list NUMBER 1-4
```

或：

```bash
javdb assets list NUMBER 1,3-5
```

---

# 8. Selector 解析规则

必须满足：

```text
1       ✓
1-4     ✓
1,3,5   ✓
1,3-5   ✓

0       ✗
-1      ✗
1-      ✗
4-1     ✗
foo     ✗
```

重复项：

```text
1-3,2-4
```

最终得到：

```text
1
2
3
4
```

按照列表顺序输出。

---

# 9. Filter 与 Selector 顺序

顺序固定：

```text
获取所有资产
    ↓
--type filter
    ↓
生成当前列表编号 1..N
    ↓
selector
```

因此：

```bash
javdb assets list NUMBER --type image
```

假设得到：

```text
1 image thumbnail
2 image cover
3 image preview 1
4 image preview 2
5 image preview 3
```

那么：

```bash
javdb assets list NUMBER --type image 1-4
```

就是：

```text
当前 image 结果中的前四项
```

编号不是长期 Asset ID。

过滤结果变化后编号允许变化。

这正是为什么 Asset model 不需要：

```go
ID int
```

---

# 10. 不增加 `--all`

没有 selector：

```bash
javdb assets list NUMBER --type image
```

本身就表示：

```text
所有 image
```

因此没有必要增加：

```text
--all
```

---

# 11. 四种资产列表输出

## 11.1 TTY

```bash
javdb assets list NUMBER
```

示例：

```text
#   TYPE    DESCRIPTION
1   image   thumbnail
2   image   cover
3   image   preview 1
4   image   preview 2
5   video   preview
```

---

## 11.2 默认 pipeline 输出

当 stdout 非 TTY：

```bash
javdb assets list NUMBER --type image |
    javdb assets download
```

producer 输出：

```text
image<TAB>https://...
image<TAB>https://...
image<TAB>https://...
```

即：

```text
TYPE<TAB>URL
```

不输出：

```text
schema
kind
role
id
index
```

---

## 11.3 `--json`

```bash
javdb assets list NUMBER --json
```

输出：

```json
[
  {
    "type": "image",
    "url": "https://..."
  },
  {
    "type": "video",
    "url": "https://..."
  }
]
```

只有：

```text
type
url
```

---

## 11.4 `--ndjson`

如果保留项目现有显式 NDJSON 使用习惯：

```bash
javdb assets list NUMBER --ndjson
```

输出：

```json
{"type":"image","url":"https://..."}
{"type":"image","url":"https://..."}
{"type":"video","url":"https://..."}
```

仍然没有 envelope。

---

# 12. Assets 不使用 javdb.pipeline/v1 Envelope

现有项目中的：

```text
schema
kind
Envelope
```

属于已有 pipeline 基础设施。

本功能不删除整个项目已有协议。

但是：

> Asset domain 不加入该协议。

因此不得新增：

```go
KindAsset
```

不得生成：

```json
{
  "schema": "javdb.pipeline/v1",
  "kind": "asset",
  ...
}
```

`assets list | assets download` 使用自己的极简文本流：

```text
TYPE<TAB>URL
```

这避免为了两个字段引入整套 envelope metadata。

---

# 13. Download Consumer

真正落盘：

```bash
javdb assets list NUMBER --type image 1-4 |
    javdb assets download
```

指定目录：

```bash
javdb assets list NUMBER --type image 1-4 |
    javdb assets download -d ./media
```

单文件指定完整路径：

```bash
javdb assets list NUMBER --type video |
    javdb assets download -o ./preview.mp4
```

---

# 14. `-o` 与 `-d`

## `-d DIR`

适用于：

```text
一个资产
多个资产
```

默认：

```text
.
```

## `-o PATH`

只允许：

```text
恰好一个输入资产
```

如果多个输入：

```text
assets download: -o requires exactly one asset
```

---

# 15. 默认文件命名

由于 pipeline 不携带：

```text
number
role
index
```

不要为了生成漂亮文件名再次增加 metadata。

默认使用当前输入顺序生成。

图片：

```text
image-001.jpg
image-002.jpg
image-003.png
```

扩展名由实际图片 magic/type 检测结果决定。

视频：

```text
video-001.mp4
video-002.mp4
```

第一版 JavDB 通常只会有一个 preview video。

---

# 16. 视频默认输出

用户直接：

```bash
javdb assets list NUMBER --type video |
    javdb assets download
```

默认产生：

```text
video-001.mp4
```

也就是默认提供：

> 最适合直接播放和发送的输出。

如果明确想保留 TS：

```bash
javdb assets list NUMBER --type video |
    javdb assets download -o preview.ts
```

如果明确要 MP4：

```bash
javdb assets list NUMBER --type video |
    javdb assets download -o preview.mp4
```

---

# 17. 视频输出格式契约

只有：

```text
.ts
.mp4
```

## `.ts`

```text
HLS
→ download
→ AES-128 decrypt
→ validate
→ MPEG-TS
```

## `.mp4`

```text
HLS
→ download
→ AES-128 decrypt
→ validate
→ TS demux
→ H.264/AAC samples
→ MP4 remux
→ Fast Start
→ validate
```

其他扩展名：

```text
unsupported video output format
```

---

# 18. 不增加格式参数

禁止：

```text
--format
--container
--faststart
--telegram
--transcode
```

单文件时由：

```text
-o filename.ext
```

决定格式。

自动命名时默认：

```text
.mp4
```

---

# 19. 图片下载目标

图片不进行：

```text
resize
upscale
JPEG re-encode
PNG re-encode
quality enhancement
```

流程：

```text
URL
 ↓
download
 ↓
detect normal image
 ↓
if needed: JavDB XOR unwrap
 ↓
detect again
 ↓
validate
 ↓
atomic publish
```

---

# 20. 图片完整性要求

支持现有格式：

```text
JPEG
PNG
GIF
WEBP
AVIF
HEIC-like ftyp
```

以下 payload 必须失败：

```text
HTML error page
JSON error
403 response page
Cloudflare page
empty response
broken XOR payload
unknown binary
```

不得：

```text
HTTP 200
+
body != image
```

仍然保存成 `.jpg`。

---

# 21. 防盗链全部内部处理

普通用户不应该知道：

```text
Referer
Origin
User-Agent
CDN host
AES key request header
```

统一内部 MediaFetcher 处理：

```text
image
playlist
segment
AES key
```

第一版不要新增：

```text
--media-header
```

如果未来确实出现需要用户 override 的实际案例，再单独设计。

不要预先暴露高级参数。

---

# 22. MediaFetcher 安全边界

必须：

```text
只接受 http / https
```

并确保：

```text
Cookie
Authorization
其他敏感 header
```

不会因为 playlist 引用了另一个 CDN host 就无条件传播。

日志不得打印 credential。

---

# 23. 当前 HLS 支持范围

继续保持 focused：

```text
single-media playlist
finished VOD
MPEG-TS segments
AES-128
```

不要顺手扩展：

```text
master playlist
live playlist
EXT-X-BYTERANGE
EXT-X-MAP / fMP4 input
```

这些不是本 PR 的目标。

---

# 24. 视频 Codec Contract

MP4 第一版支持：

```text
H.264 / AVC
AAC
H.264-only
```

不支持：

```text
HEVC
AC-3
E-AC-3
其他未知 codec
```

发现不支持时明确失败。

例如：

```text
unsupported video codec: HEVC
```

不自动转码。

---

# 25. 禁止 ffmpeg runtime dependency

不得在运行时执行：

```bash
ffmpeg
ffprobe
```

不得要求用户安装：

```text
ffmpeg
libavcodec
native shared libraries
```

目标仍然是：

```text
single Go binary
```

---

# 26. Pure-Go Remux

目标不是 codec 转码。

实现只负责：

```text
MPEG-TS
 ↓
PES
 ↓
H.264 Annex-B
AAC ADTS
 ↓
MP4 samples
 ↓
avc1 / mp4a
```

视频像素保持原样。

音频 PCM 不 decode。

---

# 27. Dependency Spike 必须先做

正式实现 MP4 前，先验证候选 pure-Go media library。

验证：

```text
Go version
License
Maintenance status
Dependency tree
Cross-platform
CGO dependency
MPEG-TS demux
H264 parse
AAC ADTS parse
PTS/DTS
B-frame support
MP4 mux
Fast Start
Sample table correctness
```

没有通过 compatibility spike：

> 不进入正式实现。

---

# 28. 依赖审批 Gate

任何新增 Go dependency：

```text
不得直接修改 go.mod
```

必须先报告：

```text
为什么需要
候选库
Go version 要求
license
维护状态
transitive dependencies
是否 CGO
替代方案
```

得到维护者认可后才能添加。

尤其不能为了方便直接把 ffmpeg wrapper 引进来。

---

# 29. 视频下载必须先验证，再转换

核心 invariant：

> 未验证的 segment 不允许进入最终输出。

禁止：

```text
坏 segment
↓
照样 demux
↓
照样封装成 MP4
```

因为这只是在：

> 把损坏内容包装进另一个 container。

---

# 30. Layer A — Segment Integrity

每个 HLS segment：

```text
download
 ↓
validate transport
 ↓
decrypt
 ↓
validate TS
 ↓
commit
```

检查至少包括：

```text
HTTP 2xx
body non-empty
Content-Length（若可靠）
AES-128 decrypt valid
188-byte TS alignment
0x47 sync
PAT parseable
PMT parseable
PID consistency
PES parseability
obvious truncation
```

---

# 31. Continuity 校验不要过严

可以检查：

```text
continuity counter
```

但必须正确处理：

```text
segment boundary
EXT-X-DISCONTINUITY
合法 reset
```

不能因为 segment 切换就误报损坏。

---

# 32. Segment bounded retry

损坏 segment：

```text
discard
↓
retry same segment
```

建议：

```text
max attempts = 3
```

例如：

```text
segment 17 attempt 1 invalid
segment 17 attempt 2 invalid
segment 17 attempt 3 invalid
```

最终：

```text
segment 17 remained invalid after 3 attempts
```

然后整个资产下载失败。

---

# 33. `.ts` 同样执行完整性验证

`.ts` 不能继续：

```text
download
→ blindly concatenate
```

正确：

```text
download
→ decrypt
→ segment validate
→ temp TS
→ final media validation
→ atomic publish
```

所以：

```text
.ts
.mp4
```

共享 Layer A。

---

# 34. Layer B — Media Integrity

所有 segment 下载完成后，必须验证整个媒体模型。

检查：

```text
video track exists
video samples > 0
SPS/PPS valid
supported codec
AAC config valid if audio exists
duration > 0
timestamps usable
DTS sensible
PTS usable
sync sample/keyframe known
no fatal PES truncation
no impossible timeline regression
```

只有通过后才能进入 MP4 finalize。

---

# 35. Fast Start 两阶段策略

不要把完整视频载入 RAM。

## Phase 1

```text
HLS
 ↓
download/decrypt
 ↓
validate
 ↓
demux
 ↓
temporary media spool
+
sample metadata
```

## Phase 2

已知完整 sample table 后：

```text
write ftyp
write moov
write mdat
```

形成：

```text
ftyp
moov
mdat
```

---

# 36. 内存目标

不允许：

```text
100 MB video
≈
100+ MB []byte RAM
```

网络、解密和解析必须：

```text
streaming
+
bounded segment buffer
```

内存消耗不应该随整个视频长度近似线性增长。

---

# 37. 临时磁盘

MP4 finalize 时可能同时存在：

```text
media spool
video.tmp
```

因此允许短时间：

```text
temporary disk ≈ 2 × output size
```

这是为了换取：

```text
bounded RAM
+
Fast Start
+
strong validation
```

---

# 38. Progressive MP4 invariant

所有 `.mp4` 必须满足：

```text
ISO BMFF
ftyp exists
moov exists
mdat exists
moov before mdat
```

即：

```text
ftyp → moov → mdat
```

不是：

```text
ftyp → mdat → moov
```

---

# 39. Track Contract

视频：

```text
H.264 → avc1
```

音频：

```text
AAC → mp4a
```

H.264-only：

```text
合法支持
```

timed ID3：

```text
默认丢弃
```

不要创建无意义 MP4 track。

---

# 40. Layer C — Container Integrity

最终 MP4 先写：

```text
video.tmp
```

然后重新打开解析。

检查：

```text
ftyp
moov
mdat
moov < mdat
avc1 track
mp4a track if audio
duration > 0
samples > 0
sample offsets in file bounds
sample sizes in file bounds
sample table coherent
sync samples valid
timestamps parseable
```

通过后才发布。

---

# 41. Atomic Publish

所有资产：

```text
image
TS
MP4
```

统一：

```text
target.tmp
 ↓
write
 ↓
validate
 ↓
close
 ↓
atomic rename
 ↓
target
```

失败时：

```text
no final target
```

---

# 42. 不覆盖已有文件

目标路径已存在：

```text
fail
```

不要：

```text
overwrite silently
```

多文件下载开始前应尽可能进行：

```text
target preflight
```

避免：

```text
已经下载前三个
第四个才发现文件冲突
```

---

# 43. Download 默认命名预检查

对于：

```bash
assets download -d ./media
```

先根据输入数量生成：

```text
image-001.*
image-002.*
video-001.mp4
```

图片真实扩展名在下载校验后才完全确定，因此：

```text
temp path
↓
detect format
↓
resolve final extension
↓
collision check
```

如果最终路径已有文件：

```text
fail without overwriting
```

---

# 44. Context cancellation

所有阶段响应：

```go
context.Context
```

包括：

```text
HTTP fetch
retry
AES key fetch
decrypt
TS parse
demux
spool
MP4 finalize
validation
```

取消后：

```text
stop network
stop work
cleanup temp
return context error
```

---

# 45. SDK Contract 重构

现有：

```go
MovieAssetDownloadOptions{
    ThumbnailPath: ...
    PreviewImagePath: ...
    PreviewVideoPath: ...
}
```

已经不适合多预览图。

应改为真正的 Asset API。

建议：

```go
type MovieAsset struct {
    Type string
    URL  string
}
```

发现：

```go
func (c *Client) MovieAssets(
    ctx context.Context,
    movieID string,
) ([]MovieAsset, error)
```

下载单个：

```go
func (c *Client) DownloadMovieAsset(
    ctx context.Context,
    asset MovieAsset,
    target string,
) (int64, error)
```

---

# 46. SDK 不增加额外 metadata

禁止：

```go
MovieAsset{
    ID: ...
    Kind: ...
    Role: ...
    Index: ...
    Schema: ...
}
```

SDK 与 CLI 共享：

```text
type
url
```

这个最小 contract。

---

# 47. SDK 视频 target 契约

调用：

```go
DownloadMovieAsset(ctx, asset, "preview.ts")
```

得到 TS。

调用：

```go
DownloadMovieAsset(ctx, asset, "preview.mp4")
```

得到 Fast Start MP4。

如果：

```go
asset.Type == "video"
```

且 target 是：

```text
preview.mkv
```

返回：

```text
unsupported video output format ".mkv"
```

---

# 48. SDK 图片行为

对于：

```go
asset.Type == "image"
```

执行：

```text
download
unwrap if necessary
validate
atomic publish
```

不进行格式转换。

---

# 49. 移除旧 Path-per-type SDK

完成迁移后删除：

```text
MovieAssetDownloadOptions
MovieAssetDownloadResult
PreviewImagePath
PreviewVideoPath
ThumbnailPath
```

如果项目此次版本允许 breaking change：

> 不增加 deprecated wrapper。

避免同时维护两套资产 API。

---

# 50. CLI 实现位置建议

建议：

```text
internal/cli/commands/assets/
    assets.go
    list.go
    download.go
```

selector 如果逻辑较大：

```text
selector.go
```

否则直接留在 `list.go`。

不要为了几个 helper 拆十几个文件。

---

# 51. Media 实现位置

底层复杂性继续封装：

```text
internal/javdb/appapi/media/
```

职责大致：

```text
fetch
image validation
HLS acquisition
AES decrypt
TS validation
demux
MP4 mux
final validation
```

不要让：

```text
CLI
SDK
```

接触：

```text
TS packet
PES
NALU
ADTS
MP4 box
```

---

# 52. 测试策略：只保留核心链路

不要为每个细小 branch 新建一份大型 test suite。

目标：

```text
少量 deterministic fixtures
+
table-driven tests
+
核心 failure path
```

避免：

```text
为了一个行为新增数百行重复测试
```

---

# 53. Asset list 核心测试

覆盖：

```text
thumbnail
cover
multiple preview_images
preview video
missing items
large_url fallback thumb_url
--type image
--type video
1
1-4
1,3-5
invalid selector
out-of-range
```

无需每个排列组合都写独立测试。

---

# 54. Asset 输出测试

核心断言：

## TTY

有：

```text
编号
type
描述
```

## Pipe

严格：

```text
TYPE<TAB>URL
```

## JSON

严格只有：

```text
type
url
```

## NDJSON

严格只有：

```text
type
url
```

增加 contract test 防止未来又塞入：

```text
schema
kind
role
index
```

---

# 55. Asset download 测试

核心：

```text
single image
multiple images
single video
-d
-o
-o + multiple input fails
existing output fails
invalid line fails
unsupported type fails
```

---

# 56. Image fixture

保留很小 fixture：

```text
JPEG
PNG
XOR-wrapped image
```

损坏场景可以直接：

```text
truncate bytes
replace magic
```

不需要专门保存大量坏文件。

---

# 57. HLS fixture

准备一个非常小的 deterministic VOD fixture：

```text
H.264
AAC
multiple TS packets
multiple samples
```

另一个 AES case 可以：

```text
在 test setup 中加密同一个 fixture
```

尽量避免多个大二进制 fixture。

---

# 58. Segment integrity 核心测试

覆盖：

```text
valid TS
truncated packet
invalid sync
missing PAT/PMT
broken PES
valid discontinuity
retry success
retry exhausted
AES success
AES failure
```

优先 table-driven。

---

# 59. MP4 核心测试

必须验证：

```text
real MP4
ftyp
moov
mdat
moov < mdat
avc1
mp4a
duration > 0
samples > 0
valid sample offsets
```

并覆盖：

```text
video-only
unsupported codec
malformed TS
remux failure
```

---

# 60. Final validation failure test

故意损坏临时 MP4：

```text
truncate
corrupt offset
```

validator 必须：

```text
reject
delete temp
not publish target
```

---

# 61. ffprobe 仅可选

如果开发/CI 环境存在：

```text
ffprobe
```

可以增加 optional smoke check。

但测试 correctness 不能依赖：

```text
ffmpeg
ffprobe
```

Go test 必须能够独立证明核心 contract。

---

# 62. Real E2E

完成 unit/integration 后，执行真实：

```bash
javdb assets list NUMBER
```

确认真实资产数量与类型。

然后：

```bash
javdb assets list NUMBER --type image 1-2 |
    javdb assets download -d /tmp/assets
```

确认图片可以直接打开。

---

# 63. Real Video E2E — MP4

执行：

```bash
javdb assets list NUMBER --type video |
    javdb assets download -o /tmp/preview.mp4
```

验证：

```text
真实 MP4
H264
AAC if available
duration
seek
moov before mdat
```

---

# 64. Real Video E2E — TS

执行：

```bash
javdb assets list NUMBER --type video |
    javdb assets download -o /tmp/preview.ts
```

验证：

```text
真实 MPEG-TS
完整
可以播放
```

---

# 65. Telegram 修复独立

`javdb-cli` 只负责：

> 产生正确 Progressive MP4。

Telegram Adapter 负责：

> 正确告诉 Telegram 该视频支持 streaming。

不要把：

```text
Telegram
supports_streaming
```

写进 javdb-cli。

---

# 66. Hermes 独立 PR

另开：

```text
fix/telegram-video-streaming
```

所有：

```python
bot.send_video(...)
```

统一：

```python
supports_streaming=True
```

覆盖：

```text
adapter
send_message tool
retry
thread fallback
caption retry
```

---

# 67. Telegram 不负责转封装

禁止：

```text
Telegram adapter
↓
发现 TS
↓
调用 ffmpeg
↓
修 MP4
```

正确：

```text
javdb-cli
↓
valid Progressive MP4
↓
Telegram adapter
↓
sendVideo + supports_streaming=true
```

---

# 68. Telegram Real E2E

最终：

```text
JavDB HLS
↓
javdb-cli
↓
Fast Start MP4
↓
Hermes
↓
Telegram sendVideo
↓
Telegram client
```

验收：

```text
内联 video player
上传结束前可开始播放
seek 正常
duration 正常
A/V sync 正常
```

---

# 69. 文档更新

更新：

```text
README.md
README.zh-CN.md

docs/en/cli-reference.md
docs/zh-CN/cli-reference.md

docs/en/sdk.md
docs/zh-CN/sdk.md

docs/maintainers/architecture.md

skills/javdb-cli/SKILL.md
skills/javdb-cli/references/*
```

---

# 70. 文档主路径

普通用户只需要看到：

```bash
javdb assets list NUMBER
```

然后：

```bash
javdb assets list NUMBER 3-4 |
    javdb assets download
```

或者：

```bash
javdb assets list NUMBER --type image 1-4 |
    javdb assets download -d ./images
```

视频：

```bash
javdb assets list NUMBER --type video |
    javdb assets download -o preview.mp4
```

---

# 71. Agent Skill

Agent 应优先：

```text
先 assets list
↓
看用户需要什么
↓
选择
↓
pipe 到 assets download
```

例如：

```bash
javdb assets list NUMBER --type image 1-2 |
    javdb assets download -d ./images
```

不要求 Agent：

```text
自己读取 preview_images JSON
自己写 Referer
自己下载 HLS segments
自己调用 ffmpeg
```

---

# 72. 分支策略

现有：

```text
refactor/download-command-clarity
```

原计划中的：

```text
assets 直接承担下载
download 作为 alias
```

已经不再是最终 contract。

因此：

> 不要原样合并旧计划。

应该修改/取代该设计。

---

# 73. 推荐实施分支

等当前有冲突的既有工作稳定后，从最新 main 创建：

```text
feat/assets-media-download
```

这一分支完整交付：

```text
assets list
assets download
minimal Asset model
image handling
validated TS
Fast Start MP4
docs
skill
```

避免先发布半套 contract，再第二个 PR 修改行为。

---

# 74. 建议 Commit 顺序

```text
test(assets): define minimal asset list contract

refactor(assets): expose type-url movie assets

feat(cli): add assets list filtering and selectors

feat(cli): add assets download pipe consumer

test(media): define validated download contract

refactor(media): centralize asset fetching and image validation

feat(media): validate and retry HLS segments

test(media): define progressive MP4 contract

feat(media): add pure-Go TS to MP4 remux

feat(media): validate and atomically publish outputs

docs: document asset list-download workflow

docs(skill): update asset automation workflow
```

---

# 75. 不要为了 commit 数量拆过度

如果：

```text
test + implementation
```

非常紧密，可以合并为一个逻辑 commit。

重点是：

```text
每个 commit 都可理解
```

而不是人为追求很多 commit。

---

# 76. Definition of Done — CLI

必须满足：

```bash
javdb assets list NUMBER
javdb assets list NUMBER --type image
javdb assets list NUMBER --type video
javdb assets list NUMBER 1-4
javdb assets list NUMBER 1,3-5
```

全部正确。

并且：

```bash
javdb assets list ... | javdb assets download
```

不需要：

```text
--ndjson
```

---

# 77. Definition of Done — Data Model

Asset domain 最终只有：

```text
type
url
```

不得出现：

```text
schema
kind
role
id
index
```

作为 Asset 数据模型。

---

# 78. Definition of Done — Image

图片：

```text
下载成功
直接可打开
格式正确
错误 payload 不保存
XOR payload 正确还原
失败不留最终文件
```

---

# 79. Definition of Done — TS

`.ts`：

```text
真实 MPEG-TS
segment 全部通过 integrity gate
AES 正确
失败无半成品
```

---

# 80. Definition of Done — MP4

`.mp4`：

```text
真实 ISO BMFF
H.264 = avc1
AAC = mp4a
duration > 0
sample table valid
moov before mdat
无需 ffmpeg
无转码
失败无半成品
```

---

# 81. Definition of Done — Memory

对于大预览：

```text
RAM 使用不随完整文件大小线性增长
```

采用：

```text
bounded network buffer
+
temporary spool
+
sample metadata
```

---

# 82. Definition of Done — Integrity

必须完整经过：

```text
Layer A
Segment Integrity

Layer B
Media Integrity

Layer C
Container Integrity
```

任意一层失败：

```text
最终文件不得发布
```

---

# 83. Definition of Done — Platform

Hermes 独立满足：

```text
native sendVideo
supports_streaming=True
```

真实 Telegram client：

```text
progressive playback
+
seek
```

---

# 84. 明确排除项

本工作不得扩张成：

```text
完整影片下载
磁力下载器
BT client
aria2
qBittorrent
115
视频转码
画质增强
图片超分
live HLS
master playlist
fMP4 HLS input
HEVC 转 AVC
Telegram integration in javdb-cli
```

---

# 85. 最终用户链路

图片：

```text
javdb assets list NUMBER --type image
               │
               ▼
       选择 1 / 1-4 / 1,3
               │
               ▼
        javdb assets download
               │
               ▼
        validated local images
```

视频：

```text
javdb assets list NUMBER --type video
               │
               ▼
        javdb assets download
               │
               ▼
        HLS acquisition
               │
        AES-128 decrypt
               │
        Segment Integrity
               │
               ▼
            MPEG-TS
          ┌─────┴─────┐
          ▼           ▼
        .ts          demux
                      │
                 H264 + AAC
                      │
                Media Integrity
                      │
                      ▼
                   MP4 mux
                      │
               ftyp → moov → mdat
                      │
              Container Integrity
                      │
                      ▼
                  final .mp4
```

---

# 86. 最终不可违反的 Invariants

```text
1. 顶层 javdb download 不恢复。

2. 资产下载只存在于：
   javdb assets download

3. Asset model 只有：
   type + url

4. 不新增 asset schema/kind/role/id/index。

5. 编号只是当前 list 结果的位置。

6. --type 只有 image / video。

7. 不增加 --preview。

8. 不增加 --all。

9. 正常 pipe 不要求 --ndjson。

10. pipe 默认协议只使用：
    TYPE<TAB>URL

11. 图片不转码、不增强。

12. 普通用户不用配置防盗链。

13. 视频默认自动下载为 MP4。

14. 显式 .ts 保留 Transport Stream。

15. .mp4 必须是真 Fast Start MP4。

16. 不使用 ffmpeg runtime。

17. 不进行视频/音频转码。

18. 未通过完整性验证的数据不得发布。

19. 不把完整视频加载进 RAM。

20. 失败不留下最终半成品。

21. 新增 dependency 前必须先完成 spike 并说明理由。

22. javdb-cli 不包含 Telegram 专用逻辑。

23. Telegram Adapter 不负责媒体修复。
```

满足以上全部条件后，本次资产下载与视频输出优化才算真正完成。
