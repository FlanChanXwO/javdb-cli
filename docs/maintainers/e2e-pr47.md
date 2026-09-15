# PR #47 真实 E2E 实测报告

本页保留 PR #47 媒体链路的真实 JavDB E2E 记录。番号统一以 `NUMBER` 占位，
不记录具体影片；媒体 URL 域名属于公开 CDN/镜像地址。

实测环境：macOS darwin-arm64、Go 1.26，历史测量 commit `e96202b`；
`ffprobe`/`ffmpeg` 仅用于独立验证，不是运行时依赖。当前契约的资产机器输出
严格只有 `type` 与 `url`，不会在 list 阶段读取媒体内容。

## E2E #1/#2：`assets list NUMBER`

- `javdb assets list NUMBER` 返回 13 个资产（thumbnail、cover、10 张 preview 图与 preview video），
  管道输出为 `TYPE<TAB>URL`，无装饰。
- `javdb assets list NUMBER --json` 返回 JSON 数组；每个对象严格只有 `type` 与 `url`。

## E2E #3：pipe → download

```bash
javdb assets list NUMBER --type image 1-2 | javdb assets download -d DIR
```

- stdout 只输出最终路径（`DIR/image-001.jpg`、`DIR/image-002.jpg`），无 `saved ...` 装饰。
- `file` 确认产物为合法 JPEG；无残留临时文件，已有目标不会被覆盖。

## E2E #4：video → TS

- `javdb assets list NUMBER --type video | javdb assets download -o preview.ts`
  生成 MPEG-TS；`ffprobe` 确认 h264 720×404 + aac，`ffmpeg -f null -` 完整
  decode 无错误（exit 0）。

## E2E #5：video → MP4（Fast Start remux）

- `javdb assets list NUMBER --type video | javdb assets download -o preview.mp4`
  生成 ISO MP4。
- **Fast Start**：`ftyp → moov → mdat` 顺序确认。
- **独立验证**：`ffprobe` 确认 h264 720×404、aac 48000 Hz 立体声、时长与上游
  preview 时间轴一致；`ffmpeg -v error -i preview.mp4 -f null -` 无错误输出。
- 在 10s、60s、110s 随机点执行 seek 均成功。

## E2E #6：资源占用实测

历史测量使用同一 segment 重复引用 120 次、输出约 569 MB：

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
