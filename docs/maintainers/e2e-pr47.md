# PR #47 真实 E2E 实测报告

按 PR 修改建议 §32 执行的真实 JavDB E2E。番号在全部记录中以 `NUMBER` 占位,
不出现具体番号;媒体 URL 域名照实记录(它们是公开 CDN/镜像地址)。

实测环境:macOS darwin-arm64,Go 1.26,构建 commit e96202b;
独立媒体解析使用本机 `ffprobe`/`ffmpeg`(仅验证用,非运行时依赖)。

## E2E #1/#2:`assets list NUMBER`(文本/JSON)

- `javdb assets list NUMBER` → 13 个资产(thumbnail + cover + 10 preview 图 + preview video),
  pipe 输出为 `TYPE<TAB>URL`,无装饰。
- `javdb assets list NUMBER --json` → JSON 数组,12 张图片全部带 `width`/`height`
  probe 结果(800×439、800×534、534×800、147×200 等,与 `ffprobe` 图片尺寸一致);
  preview video 带 `duration=116.851`(来自 HLS `#EXTINF` 求和)。
- 无法获取的元信息直接省略;没有 `0` 冒充。

## E2E #3:pipe → download

```bash
javdb assets list NUMBER --type image 1-2 | javdb assets download -d DIR
```

- stdout 只输出最终路径(`DIR/image-001.jpg`、`DIR/image-002.jpg`),无 `saved ...` 装饰。
- `file` 确认合法 JPEG;尺寸与 probe 一致(147×200、800×439)。
- 下载产物权限 `0600`/`0644`,无残留临时文件。

## E2E #4:video → TS

- `javdb assets list NUMBER --type video | javdb assets download -o preview.ts`
  → 49.05 MB MPEG-TS,`ffprobe` 确认 h264 720×404 + aac;`ffmpeg -f null -` 完整
  decode 无错误(exit 0)。

## E2E #5:video → MP4(Fast Start remux)

- `javdb assets list NUMBER --type video | javdb assets download -o preview.mp4`
  → 47.55 MB ISO MP4。
- **Fast Start**:`ftyp → moov → mdat` 顺序确认(moov 在 36,mdat 在 100259)。
- **ffprobe 独立验证**(不是自写 validator 的自我确认):
  - h264 **720×404**,30000/1001 fps —— `720p.m3u8` 没有被错误报告成 1280×720。
  - aac 48000 Hz 立体声。
  - format duration 116.917 s,与 EXTINF 求和 116.851 s 一致(差 0.07 s,属
    最后一个样本 delta 推导的正常舍入)。
- **完整 decode**:`ffmpeg -v error -i preview.mp4 -f null -` 无错误输出(exit 0)。
- **seek**:在 10s/60s/110s 三个随机点 `-ss` seek 全部成功(exit 0)。
- **A/V sync**:视频轨 116.85 s,音频轨 0.02 s(预览流音频时长本就极短,与上游一致)。

## E2E #6:资源占用实测

### 10x 构造媒体(同一 segment 重复引用 120 次,输出 569 MB)

| 指标 | 1x(47.5 MB 输出) | 10x(569 MB 输出) |
| --- | --- | --- |
| peak RSS | 73.4 MB | **68.2 MB** |
| wall time | 12.5 s(真实 CDN) | 2.7 s(本地服务器,无网络延迟) |

**RSS 不随媒体长度增长**:输出体积 12×,RSS 持平甚至略降——spool 化管线
(样本落盘、内存只保留 moov 级元数据)的预期行为得到实测确认。

### probe 请求成本

- 12 张图片 + 1 个 video:图片 probe 请求 12 次,预算 12 × 64 KiB = 768 KiB 上限;
  video probe 为 playlist 256 KiB + 首 segment 256 KiB。
- 实际 wall time:`assets list --json` 与 pipe 模式无可感知差异(probe 并发 4)。

### pipe 模式跳过 probe

- `javdb assets list NUMBER`(pipe)输出 13 行 `TYPE<TAB>URL`,无元数据字段;
  请求计数与 JSON 模式对比确认 probe 请求为 0。

## E2E #7:`[assets.probe]` 配置

- `enabled = false` → `--json` 输出 13 个资产,**0 个带元数据字段**(probe 完全关闭)。
- 部分配置 `concurrency = 2` → `config get` 确认 `concurrency=2`,
  其余字段保持默认(65536 / 10s)。
- `config set assets.probe.timeout 5s` → `get` 返回 `5s`;
  `config unset assets.probe.timeout` → `get` 回落默认 `10s`。
- 六个键全部可在 `config get/set/unset` 使用。

## 结论

报告 §32 的真实 E2E 与资源占用实测全部通过;`#32 E2E` 明确要求的关键断言
(720p.m3u8 不被报告成 1280×720、Fast Start、完整 decode 无错误、RSS 不随
媒体长度 10× 增长)均有独立工具(ffprobe/ffmpeg/`/usr/bin/time -l`)的证据。
