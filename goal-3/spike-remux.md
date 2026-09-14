# spike-remux:TS→MP4 remux 路线决策(goal-3/T08)

日期:2026-09-13。范围:input.md 计划 #25-#28(无 ffmpeg、纯 Go、依赖审批 gate)。

## 结论

**默认路线:纯标准库自研 demux + remux,零新增依赖**(已获计划约束支持:input.md #28
要求任何新增 Go dependency 必须先报维护者认可;goal-mode 无人值守下无法取得该认可,
自研是唯一不阻塞路径)。若 T09/T10 证明自研遇到硬阻塞,后备方案为引入
`github.com/Eyevinn/mp4ff`,届时作为独立审批项向维护者申报。

## 候选库核实(2026-09-13,经 proxy.golang.org 实查)

| 候选 | 最新版 | 发布时间 | Go 要求 | 传递依赖 |
|---|---|---|---|---|
| asticode/go-astits | v1.16.0 | 2026-08-14 | go 1.20 | go-astikit、pkg/profile(非 test!)、testify(test) |
| abema/go-mp4 | v1.7.3 | 2026-09-08 | go 1.14 | uuid、writerseeker、bufseekio、x/term、go-billy、testify |
| Eyevinn/mp4ff | v0.56.0 | 2026-08-22 | 待核(v0 系) | 未深入(已止步) |

三者均为 MIT、纯 Go、无 CGO、活跃维护(截至查询日)。

## 维度评估(input.md #27)

- **MPEG-TS demux**:astits 成熟;但本项目 T07 已实现 188 对齐/0x47/PSI(PAT/PMT)解析,
  剩余仅 PES 重组(跨 packet 拼接),自研成本低。
- **H.264 parse / AAC ADTS parse**:仅需要 Annex-B NALU 长度前缀转换(AVCC)与
  ADTS→AudioSpecificConfig 提取,均为确定性字节操作,无解码。
- **PTS/DTS、B-frame**:PES header 携带 PTS/DTS;B 帧需要按 DTS 排序 sample 并写
  ctts(composition offset)。是自研的真正复杂点,但输入是本工具自己下载并
  Layer A 校验过的流,不存在任意输入的防御面。
- **MP4 mux + Fast Start**:核心工作(mvhd/tkhd/mdhd/hdlr/stbl:stsd+avcC/esds、
  stts/stss/stsc/stsz/stco/ctts)。计划 #35 的两阶段写(fix moov 先于 mdat)与
  自研 box 写天然契合。
- **依赖树/审批**:go-astits 拖入 astikit+pkg/profile;go-mp4 依赖达 5 个非 test 包;
  mp4ff 单独但 v0.x。全部触发 input.md #28 的维护者审批 gate。
- **单二进制/跨平台**:三条路线均无差异(纯 Go)。

## 自研风险与后备

- R-BFrame:B 帧时间戳处理错误 → Layer B(时间戳单调、duration>0)与 Layer C
  (ffprobe 可选 smoke + 自研 box 校验)兜底;fixture 含乱序 DTS 用例。
- R-Codec:遇到 HEVC/AC-3 直接明确失败(input.md #24,不转码)。
- 后备:若自研在 T09/T10 被证不可行(非预期复杂度失控),以本报告为依据向维护者
  申报引入 Eyevinn/mp4ff(v0.56.0,MIT),阻塞等待批准,不擅自加依赖。

## 执行记录

- go.mod/go.sum 未修改(全仓库 `git status` 干净,工作全部在测试与实现代码)。
- 本报告落盘于主仓库 goal-3/(goal 状态文件,不进入 go.mod/代码树)。
