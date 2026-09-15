# PR #47 修改建议实施计划

按报告 #1–#33 实施。模块间无循环依赖,按以下批次执行,每批先 Red 后 Green。

## 批次 A:媒体解析修复(纯函数,独立)

| # | 内容 | 文件 |
|---|------|------|
| 13 | PES PTS-only `HasDTS=false`;PES_packet_length > 重组长度报 truncated | demux.go |
| 14 | ADTS freqIdx 13..15 越界检查;channel bit `(data[2]&1)<<2`;channel>2 拒绝 | tracks.go |
| 15 | PSI/PMT 边界证明(section[10]/[11]);Layer A seen = PUSI+合法 PES prefix | ts_validate.go, demux.go |
| 16 | SPS `seq_scaling_matrix_present_flag` 是 1 bit(readBit 非 readUE) | mp4.go |

## 批次 B:MP4 结构修复

| # | 内容 | 文件 |
|---|------|------|
| 18 | video-only 不 panic(音轨 nil 检查) | mp4_stream.go |
| 19 | stts {sample_count, sample_delta} RLE;末样本 delta 推导 | mp4.go |
| 20 | ctts RLE + version 1 signed offset;全零省略 | mp4.go |
| 21 | mvhd/tkhd/mdhd 按 ISO BMFF 布局;unitMatrix 中间对角项 | mp4.go |
| 22 | track ID 1/2;movie timescale 1000;duration 换算 | mp4.go |
| 23 | 无 sync sample 拒绝生成 MP4 | mp4.go / mp4_stream.go |
| 24 | avcC 分 SPS[]/PPS[] 收集 | mp4.go |
| 25 | mdat A/V chunk interleave(~1s);stsc/stco 对应 | mp4.go, mp4_stream.go |
| 26 | 32-bit 溢出明确拒绝 | mp4.go |
| 17 | 跨段 codec configuration 校验 | mp4_stream.go |
| 30 | validator 独立检查 stts/ctts/stsc/stco/track ID/interleave | mp4_stream.go |

## 批次 C:资源处理与原子发布

| # | 内容 | 文件 |
|---|------|------|
| 11 | 删除无依据的 playlist/segment/image/总 payload/segment 数/sample 数固定上限;保留协议/格式边界与逐 segment 处理 | media.go, mp4_stream.go, client/transport.go |
| 12 | 保持并发 1;segment payload 及时释放;一次拆流复用 | 现状检查 |
| 27 | os.CreateTemp 唯一临时文件 | media.go, mp4_stream.go |
| 28 | 以共享 `internal/common/atomicfile.LinkNoReplace` 实现图片、TS、MP4 真 no-replace 发布 | media.go, download.go, internal/common/atomicfile |
| 29 | cleanup 错误不静默吞 | media.go, download.go |

## 批次 D:资产契约收敛

| # | 内容 | 文件 |
|---|------|------|
| 1 | MovieAsset 严格只保留 Type/URL;删除 Width/Height/Duration 与所有 metadata probe | sdk/asset.go, internal/javdb/appapi/media/probe.go |
| 2/3 | assets list:filter→selector;TTY 仅渲染描述,JSON/NDJSON 与 pipe 严格只输出 type/url | internal/cli/commands/assets/list.go |
| 4/6 | 删除 `[assets.probe]` 配置 schema、config 白名单、transport bounded probe 与 probe 测试 | internal/config/settings, config.go, client/transport.go |

## 批次 E:assets download 改造

| # | 内容 | 文件 |
|---|------|------|
| 7 | stdout 只输出最终路径;去掉 `saved ... (bytes)`;写错误传播 | internal/cli/commands/assets/download.go |
| 8 | stdin 流式逐条;`-o` 只需读两条判断;行上限 64 KiB | 同上 |
| 9 | selector 解析 start/end 后按资产数校验再展开 | selector.go |
| 10 | movieString 只接受 string | sdk/asset.go |

## 批次 F:文档 + skill

README×2、cli-reference×2、sdk×2、architecture.md、skills/javdb-cli。

## 验证

- 每批:`go test ./...`(Red→Green)
- 收尾:`sh scripts/build.sh`、`go vet`、真实 E2E(需用户授权联网)
