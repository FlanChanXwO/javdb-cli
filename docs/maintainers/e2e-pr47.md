# PR #47 真实 E2E 实测报告

本页保留 PR #47 媒体链路的真实 JavDB E2E 记录。番号统一以 `NUMBER` 占位，
不记录具体影片；媒体 URL 域名属于公开 CDN/镜像地址。

实测环境：macOS darwin-arm64、Go `go1.26.3`，本次 E2E 验证代码 commit `699505d`；
`ffprobe`/`ffmpeg` `8.1.1`，仅用于独立验证，不是运行时依赖。当前契约的资产机器
输出严格只有 `type` 与 `url`，不会在 list 阶段读取媒体内容。

## E2E #1/#2：`assets list NUMBER`

- `javdb assets list NUMBER` 返回 13 个资产（thumbnail、cover、10 张 preview 图与 preview video），
  管道输出为 `TYPE<TAB>URL`，无装饰；视频筛选结果为单行 `video<TAB>URL`。
- `javdb assets list NUMBER --json` 返回 JSON 数组；逐项校验确认对象严格只有 `type` 与 `url`，
  `type` 仅为 `image`/`video`，URL 为 HTTP(S)。

## E2E #3：pipe → download

```bash
javdb assets list NUMBER --type image 1-2 | javdb assets download -d DIR
```

- stdout 只输出最终路径（`DIR/image-001.jpg`、`DIR/image-002.jpg`），无 `saved ...` 装饰。
- `file` 与 macOS `sips` 确认产物可识别且可读取（本轮样本为 147×200 与 800×439 JPEG）；
  无残留 `.part`/`.spool` 临时文件，已有目标重复下载会失败且 SHA-256 不变。

## E2E #4：video → TS

- `javdb assets list NUMBER --type video | javdb assets download -o preview.ts`
  生成 MPEG-TS；`ffprobe` 确认 H.264 720×404、AAC 48 kHz 双声道，
  `ffmpeg -v error -f null -` 完整 decode 退出码为 0。

## E2E #5：video → MP4（Fast Start remux）

- `javdb assets list NUMBER --type video | javdb assets download -o preview.mp4`
  生成 ISO MP4。
- **Fast Start**：`ftyp → moov → mdat` 顺序确认。
- **独立验证**：`ffprobe` 记录 format duration `116.916800s`；视频为 H.264
  720×404、`30000/1001`、`116.850067s`；音频为 AAC、48000 Hz、双声道、
  `116.885333s`。`ffmpeg -v error -i preview.mp4 -f null -` 完整解码退出码为 0，
  stderr 为空。
- **packet 级 B-frame 检查**：视频共 3502 个 packet，DTS 从 `0` 到 `10513503`
  全程非递减、未发现 regression；PTS-DTS composition offset 范围为 `0..15015`
  （90 kHz 时间基）。因此本次没有修改 mux 时间戳逻辑，也没有复现 null muxer 诊断。
- 按探测到的总时长动态计算 10%、50%、90% 三个位置执行 seek，三处均成功；未使用
  固定秒数（本次位置约为 `11.692s`、`58.458s`、`105.225s`）。

## E2E #7：最新 HEAD 的流式资源处理

- 媒体 transport 将响应 body 交给下载层消费；playlist 按行读取，segment、解密结果和
  图片中间结果使用临时文件流转，不把普通媒体响应无条件读入内存。只有协议明确要求
  固定 16 字节的 AES-128 key 使用有界读取，并通过实际受限 reader 测试确认在超限后停止。
- MP4 spool 使用 target 目录内的唯一临时文件；预先存在的 `preview.mp4.spool` 不会
  阻塞下载。图片、TS、MP4 下载均未留下 `.part` 或 `.spool` 文件。
- `ctts` composition offset 的 32-bit 表达边界、Layer C 的 `ctts` 版本/边界/样本数
  一致性均在本次提交的 contract tests 中覆盖；未新增无依据的媒体总量或样本数硬上限。

## E2E #6：资源占用实测

历史测量（非本轮重跑）使用同一 segment 重复引用 120 次、输出约 569 MB，基于
`e96202b`：

| 指标 | 1x（约 47.5 MB 输出） | 10x（约 569 MB 输出） |
| --- | --- | --- |
| peak RSS | 73.4 MB | **68.2 MB** |
| wall time | 12.5 s（真实 CDN） | 2.7 s（本地服务器） |

RSS 不随媒体长度线性增长：样本落盘，内存只保留 moov 级元数据。

## 结论

媒体 E2E 覆盖资产顺序、`TYPE<TAB>URL` 管道、图片 magic 校验、no-replace 发布、
TS 完整性、MP4 Fast Start、独立解码和 spool 资源特征。固定的 playlist、segment、
image、总 payload 与 segment 数硬上限不属于当前契约；保留协议/格式边界和明确的
segment 重试语义。
