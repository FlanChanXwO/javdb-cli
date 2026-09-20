package media

import (
	"errors"
	"fmt"
	"sort"
)

// MP4 mux 仅执行容器级 remux，不进行转码。
// 结构固定 ftyp → moov → mdat(Fast Start);时间轴以每个 track 的首个
// 时间戳为零基准,90kHz(视频)与采样率(音频)。
// moov 是纯元数据(O(samples));样本数据本体走 spool,内存不随视频长度线性增长(#36)。

const mp4VideoTimescale = 90000

// mp4MovieTimescale 是 movie 层的独立固定时间基;tkhd/mvhd duration
// 必须换算到它，不能直接用视频 90k 或音频采样率时间。
const mp4MovieTimescale = 1000

var errSPSTruncated = errors.New("SPS truncated")

// mp4 track ID 必须唯一，禁止音视频共用 track ID 1。
const (
	mp4VideoTrackID = 1
	mp4AudioTrackID = 2
	mp4NextTrackID  = 3
)

// mp4ChunkInterleaveSeconds 是 A/V chunk 交错的目标准：
// 约 0.5～1 秒,无需 per-sample 超细粒度。
const mp4ChunkInterleaveSeconds = 1.0

// mp4SampleMeta 记录单个样本的时间戳与它的两个位置：
// Offset 是 interleave 后样本在最终 mdat 数据区内的物理绝对位置（调试与
// 布局核对用；stco 由 mp4Chunk.offset 生成，不再从样本读取）；
// SpoolOffset 是样本在 spool 文件内的物理位置（数据拷贝使用）。
// 注意 sample ≠ chunk：一个 chunk 是同一 track 的连续一组样本。
type mp4SampleMeta struct {
	Offset      int64
	SpoolOffset int64
	Size        uint32
	PTS         uint64
	DTS         uint64
	Sync        bool
}

// mp4TrackMeta 是一个 track 的完整 moov 级元数据。
type mp4TrackMeta struct {
	Timescale       uint32
	Duration        uint64
	Width           uint16
	Height          uint16
	ParamSets       [][]byte // video:SPS/PPS(avcC)
	ASC             []byte   // audio:AudioSpecificConfig(esds)
	Channels        int
	SampleRate      int
	Samples         []mp4SampleMeta
	SampleDurations []uint32 // video stts 与 mdhd 共用的样本 delta 序列
}

// planVideoTrack 把内存中的 H.264 样本转成 track 元数据(偏移顺序累计)。
func planVideoTrack(video *h264Track) (*mp4TrackMeta, error) {
	if len(video.Samples) == 0 {
		return nil, fmt.Errorf("video track has no samples")
	}
	if len(video.ParamSets) < 2 {
		return nil, fmt.Errorf("video track is missing SPS/PPS")
	}
	width, height, err := parseSPSDimensions(video.ParamSets[0])
	if err != nil {
		return nil, fmt.Errorf("parse SPS: %w", err)
	}
	// MP4 中省略 stss 表示全部 sample 都是同步样本;完全没有可确认的
	// sync sample 时拒绝生成 MP4，不伪造全部可 seek。
	syncCount := 0
	for _, s := range video.Samples {
		if s.Sync {
			syncCount++
		}
	}
	if syncCount == 0 {
		return nil, fmt.Errorf("video track has no sync samples; refusing to write an MP4 that claims all samples are seekable")
	}
	meta := &mp4TrackMeta{
		Timescale: mp4VideoTimescale,
		Width:     width,
		Height:    height,
		ParamSets: video.ParamSets,
	}
	var offset int64
	for _, s := range video.Samples {
		meta.Samples = append(meta.Samples, mp4SampleMeta{
			Offset: offset, Size: uint32(len(s.Data)),
			PTS: s.PTS, DTS: s.DTS, Sync: s.Sync,
		})
		offset += int64(len(s.Data))
	}
	durations, duration, err := videoSampleDurations(meta.Samples)
	if err != nil {
		return nil, err
	}
	meta.SampleDurations = durations
	meta.Duration = duration
	return meta, nil
}

// planAudioTrack 把内存中的 AAC 样本转成 track 元数据。
func planAudioTrack(audio *aacTrack) (*mp4TrackMeta, error) {
	meta := &mp4TrackMeta{
		Timescale:  uint32(audio.SampleRate),
		ASC:        audio.Config,
		Channels:   audio.Channels,
		SampleRate: audio.SampleRate,
	}
	var offset int64
	for _, s := range audio.Samples {
		meta.Samples = append(meta.Samples, mp4SampleMeta{
			Offset: offset, Size: uint32(len(s.Data)), PTS: s.PTS,
		})
		offset += int64(len(s.Data))
	}
	meta.Duration = uint64(len(audio.Samples)) * 1024
	return meta, nil
}

// ---- box 构造工具 ----

func mp4Box(kind string, payload ...[]byte) []byte {
	size := 8
	for _, p := range payload {
		size += len(p)
	}
	out := make([]byte, 0, size)
	out = append(out, byte(size>>24), byte(size>>16), byte(size>>8), byte(size))
	out = append(out, kind...)
	for _, p := range payload {
		out = append(out, p...)
	}
	return out
}

func mp4FullBox(kind string, version byte, flags uint32, payload ...[]byte) []byte {
	header := []byte{version, byte(flags >> 16), byte(flags >> 8), byte(flags)}
	return mp4Box(kind, append(header, flatten(payload)...))
}

func mp4U16(v uint16) []byte { return []byte{byte(v >> 8), byte(v)} }
func mp4U32(v uint32) []byte { return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)} }

func maxU64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func minU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func flatten(parts [][]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func buildFTYP() []byte {
	return mp4Box("ftyp", []byte("isom"), mp4U32(512), []byte("isomiso2avc1mp41"))
}

// buildMoov 构造完整 moov(含 sample table)。
// mdatStart 是 mdat 数据区在最终文件中的绝对偏移,stco 直接写最终值,无需回填。
// videoChunks/audioChunks 是 interleaveSamples 产出的唯一 chunk layout:
// stsc/stco 与 mdat writer 消费同一份结果。
func buildMoov(video, audio *mp4TrackMeta, videoChunks, audioChunks []mp4Chunk, mdatStart int64) ([]byte, error) {
	if video == nil || len(video.Samples) == 0 {
		return nil, fmt.Errorf("mp4 requires a video track with samples")
	}
	if err := validateChunkLayout(video, videoChunks); err != nil {
		return nil, err
	}
	includeAudio := audio != nil && len(audio.Samples) > 0
	if includeAudio {
		if err := validateChunkLayout(audio, audioChunks); err != nil {
			return nil, err
		}
	}
	// mdat 偏移与 moov 尺寸都是 32-bit 字段;可能溢出时明确拒绝,
	// 不产生损坏的容器。
	const maxBoxOffset = int64(0xFFFFFFFF)
	// stco 偏移是 32-bit:mdat 数据区末尾(含全部样本)溢出时拒绝。
	var mdatPayload int64
	for _, s := range video.Samples {
		mdatPayload += int64(s.Size)
	}
	if audio != nil {
		for _, s := range audio.Samples {
			mdatPayload += int64(s.Size)
		}
	}
	if mdatStart > maxBoxOffset || mdatStart+mdatPayload > maxBoxOffset {
		return nil, fmt.Errorf("mdat payload exceeds 32-bit MP4 bounds (start %d, payload %d)", mdatStart, mdatPayload)
	}
	// movie duration 在 movie timescale 下取音视频时长最大值;
	// mdhd duration 使用各自的 media timescale，tkhd/mvhd 换算到 movie timescale。
	videoMovieDur := u64ScaleToMovieTimescale(video.Duration, video.Timescale)
	var audioMovieDur uint64
	if audio != nil && len(audio.Samples) > 0 {
		audioMovieDur = u64ScaleToMovieTimescale(audio.Duration, audio.Timescale)
	}
	movieDuration := maxU64(videoMovieDur, audioMovieDur)
	if movieDuration == 0 {
		return nil, fmt.Errorf("mp4 duration is zero")
	}
	if movieDuration > uint64(^uint32(0)) || video.Duration > uint64(^uint32(0)) || (audio != nil && audio.Duration > uint64(^uint32(0))) {
		return nil, fmt.Errorf("MP4 version-0 duration exceeds 32-bit bounds")
	}
	mvhd := buildMovieHeader(uint32(movieDuration))
	videoTrak, err := buildVideoTrak(video, videoChunks, mdatStart)
	if err != nil {
		return nil, err
	}
	traks := [][]byte{videoTrak}
	if includeAudio {
		audioTrak, err := buildAudioTrak(audio, audioChunks, mdatStart)
		if err != nil {
			return nil, err
		}
		traks = append(traks, audioTrak)
	}
	return mp4Box("moov", mvhd, flatten(traks)), nil
}

// validateChunkLayout 只校验 chunk 非空且覆盖 track 的全部样本；
// 最终 MP4 的 stsc/stco 与 mdat 字节布局由 Layer C 重新解析校验。
func validateChunkLayout(meta *mp4TrackMeta, chunks []mp4Chunk) error {
	if len(chunks) == 0 {
		return fmt.Errorf("track has no mdat chunks")
	}
	covered := 0
	for _, chunk := range chunks {
		if len(chunk.samples) == 0 {
			return fmt.Errorf("chunk layout contains an empty chunk")
		}
		covered += len(chunk.samples)
	}
	if covered != len(meta.Samples) {
		return fmt.Errorf("chunk layout covers %d samples, track has %d", covered, len(meta.Samples))
	}
	return nil
}

// buildMovieHeader 构造 ISO BMFF version-0 mvhd。
// 每个字段显式对应规范布局，避免通过“多塞一个整数”掩盖偏移错误。
func buildMovieHeader(duration uint32) []byte {
	return mp4FullBox("mvhd", 0, 0,
		mp4U32(0), mp4U32(0), mp4U32(mp4MovieTimescale), mp4U32(duration),
		mp4U32(0x00010000), mp4U16(0x0100), mp4U16(0),
		mp4U32(0), mp4U32(0),
		mp4U32(0x00010000), mp4U32(0), mp4U32(0),
		mp4U32(0), mp4U32(0x00010000), mp4U32(0),
		mp4U32(0), mp4U32(0), mp4U32(0x40000000),
		mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0),
		mp4U32(mp4NextTrackID),
	)
}

// buildMediaHeader 构造 ISO BMFF version-0 mdhd。
func buildMediaHeader(timescale, duration uint32) []byte {
	return mp4FullBox("mdhd", 0, 0,
		mp4U32(0), mp4U32(0), mp4U32(timescale), mp4U32(duration),
		mp4U16(0x55C4), mp4U16(0),
	)
}

// buildTrackHeader 构造 ISO BMFF version-0 tkhd。
// width/height 是规范要求的 16.16 定点字段，音频轨道传入零值。
func buildTrackHeader(trackID, duration, width, height uint32, volume uint16) []byte {
	return mp4FullBox("tkhd", 0, 3,
		mp4U32(0), mp4U32(0),
		mp4U32(trackID), mp4U32(0), mp4U32(duration),
		mp4U32(0), mp4U32(0),
		mp4U16(0), mp4U16(0), mp4U16(volume), mp4U16(0),
		unitMatrix(), mp4U32(width), mp4U32(height),
	)
}

// u64ScaleToMovieTimescale 把 media timescale 下的 duration 换算到 movie timescale。
func u64ScaleToMovieTimescale(duration uint64, mediaTimescale uint32) uint64 {
	if mediaTimescale == 0 {
		return 0
	}
	return duration * mp4MovieTimescale / uint64(mediaTimescale)
}

func buildVideoTrak(meta *mp4TrackMeta, chunks []mp4Chunk, mdatStart int64) ([]byte, error) {
	duration := u64ScaleToMovieTimescale(meta.Duration, meta.Timescale)
	avcC, err := buildAVCC(meta.ParamSets)
	if err != nil {
		return nil, err
	}
	avc1 := mp4Box("avc1",
		make([]byte, 6), mp4U16(1),
		mp4U16(0), mp4U16(0),
		make([]byte, 12),
		mp4U16(meta.Width), mp4U16(meta.Height),
		mp4U32(0x00480000), mp4U32(0x00480000),
		mp4U32(0), mp4U16(1),
		make([]byte, 32),
		mp4U16(0x0018), mp4U16(0xFFFF),
		avcC,
	)
	stsd := mp4FullBox("stsd", 0, 0, mp4U32(1), avc1)
	durations := meta.SampleDurations
	if len(durations) == 0 {
		var err error
		durations, _, err = videoSampleDurations(meta.Samples)
		if err != nil {
			return nil, err
		}
	}
	if len(durations) != len(meta.Samples) {
		return nil, fmt.Errorf("video sample duration count %d != sample count %d", len(durations), len(meta.Samples))
	}
	stts, ctts, stss, err := buildVideoTimingBoxesWithDeltas(meta.Samples, durations)
	if err != nil {
		return nil, err
	}
	boxes := [][]byte{stsd, stts, stss}
	if ctts != nil {
		boxes = append(boxes, ctts)
	}
	boxes = append(boxes, buildSTSC(chunks), buildSTSZ(meta.Samples), buildSTCO(chunks))
	stbl := mp4Box("stbl", boxes...)
	media := mp4Box("minf",
		mp4FullBox("vmhd", 0, 1, mp4U16(0), mp4U16(0), mp4U16(0), mp4U16(0)),
		buildDINF(), stbl)
	mdhd := buildMediaHeader(meta.Timescale, uint32(meta.Duration))
	hdlr := mp4FullBox("hdlr", 0, 0, mp4U32(0), []byte("vide"), mp4U32(0), mp4U32(0), mp4U32(0), append([]byte("VideoHandler"), 0))
	md := mp4Box("mdia", mdhd, hdlr, media)
	tkhd := buildTrackHeader(mp4VideoTrackID, uint32(duration), uint32(meta.Width)<<16, uint32(meta.Height)<<16, 0)
	return mp4Box("trak", tkhd, md), nil
}

func buildAudioTrak(meta *mp4TrackMeta, chunks []mp4Chunk, mdatStart int64) ([]byte, error) {
	duration := u64ScaleToMovieTimescale(meta.Duration, meta.Timescale)
	esds := buildESDS(meta.ASC)
	mp4a := mp4Box("mp4a",
		make([]byte, 6), mp4U16(1),
		mp4U16(0), mp4U16(0), mp4U32(0),
		mp4U16(uint16(meta.Channels)), mp4U16(16),
		mp4U16(0), mp4U16(0),
		mp4U32(uint32(meta.SampleRate)<<16),
		esds,
	)
	stsd := mp4FullBox("stsd", 0, 0, mp4U32(1), mp4a)
	// 音频 sample duration 固定 1024(AAC-LC frame):所有样本 RLE 成单 entry。
	stts := buildRLE32Constant(uint32(len(meta.Samples)), 1024)
	stbl := mp4Box("stbl", stsd, stts, buildSTSC(chunks), buildSTSZ(meta.Samples), buildSTCO(chunks))
	media := mp4Box("minf",
		mp4FullBox("smhd", 0, 0, mp4U16(0), mp4U16(0)),
		buildDINF(), stbl)
	mdhd := buildMediaHeader(meta.Timescale, uint32(meta.Duration))
	hdlr := mp4FullBox("hdlr", 0, 0, mp4U32(0), []byte("soun"), mp4U32(0), mp4U32(0), mp4U32(0), append([]byte("SoundHandler"), 0))
	md := mp4Box("mdia", mdhd, hdlr, media)
	tkhd := buildTrackHeader(mp4AudioTrackID, uint32(duration), 0, 0, 0x0100)
	return mp4Box("trak", tkhd, md), nil
}

// buildVideoTimingBoxes 产出 stts(DTS delta RLE)、可选 ctts(PTS-DTS signed RLE)
// 与可选 stss(sync)。
// 时间戳以首样本为零基准(track 内时间从 0 开始)。
// ISO BMFF 每个 stts/ctts entry 必须为 {sample_count, sample_delta/offset};
// 末样本 delta 由倒数第二样本 delta 推导。
func buildVideoTimingBoxes(samples []mp4SampleMeta) (stts, ctts, stss []byte, err error) {
	deltas, _, err := videoSampleDurations(samples)
	if err != nil {
		return nil, nil, nil, err
	}
	stts, ctts, stss, err = buildVideoTimingBoxesWithDeltas(samples, deltas)
	return stts, ctts, stss, err
}

// videoSampleDurations 计算视频样本的 stts delta，并返回与之相同来源的 mdhd duration。
// 末样本没有下一个 DTS，因此沿用倒数第二个 delta；所有 delta 必须可编码为 uint32。
func videoSampleDurations(samples []mp4SampleMeta) ([]uint32, uint64, error) {
	if len(samples) < 2 {
		return nil, 0, fmt.Errorf("video track requires at least two timestamped samples")
	}
	deltas := make([]uint32, len(samples))
	for i := 1; i < len(samples); i++ {
		previous := samples[i-1].DTS
		current := samples[i].DTS
		if current < previous {
			return nil, 0, fmt.Errorf("video timestamp regression at sample %d: DTS %d < previous DTS %d", i, current, previous)
		}
		delta := current - previous
		if delta > uint64(^uint32(0)) {
			return nil, 0, fmt.Errorf("video sample delta %d at sample %d exceeds uint32", delta, i)
		}
		deltas[i-1] = uint32(delta)
	}
	deltas[len(deltas)-1] = deltas[len(deltas)-2]

	var duration uint64
	for _, delta := range deltas {
		if ^uint64(0)-duration < uint64(delta) {
			return nil, 0, fmt.Errorf("video track duration overflows uint64")
		}
		duration += uint64(delta)
	}
	if duration == 0 {
		return nil, 0, fmt.Errorf("video track duration is zero")
	}
	return deltas, duration, nil
}

// buildVideoTimingBoxesWithDeltas 使用已经校验并保存的 delta，确保 stts 与 mdhd 不再各算一遍。
func buildVideoTimingBoxesWithDeltas(samples []mp4SampleMeta, deltas []uint32) (stts, ctts, stss []byte, err error) {
	stts = buildRLE32(deltas)

	// ctts:offset = PTS - DTS(RLE);存在负 offset 时使用 version 1 signed。
	hasNonZero := false
	hasNegative := false
	offsets := make([]int64, len(samples))
	for i, s := range samples {
		offsets[i], err = compositionOffset(s.PTS, s.DTS)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("sample %d: %w", i, err)
		}
		if offsets[i] != 0 {
			hasNonZero = true
		}
		if offsets[i] < 0 {
			hasNegative = true
		}
	}
	if hasNonZero {
		ctts, err = buildRLE32Signed(offsets, hasNegative)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	syncs := make([][]byte, 0, 8)
	for i, s := range samples {
		if s.Sync {
			syncs = append(syncs, mp4U32(uint32(i+1)))
		}
	}
	// 存在部分 keyframe 时列出真正同步样本;全部都是 sync 时省略 stss
	// (省略语义 = 全部同步,与事实一致)。
	if len(syncs) > 0 && len(syncs) != len(samples) {
		stss = mp4FullBox("stss", 0, 0, mp4U32(uint32(len(syncs))), flatten(syncs))
	}
	return stts, ctts, stss, nil
}

func compositionOffset(pts, dts uint64) (int64, error) {
	if pts >= dts {
		delta := pts - dts
		if delta > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("composition timestamp difference cannot be represented")
		}
		return int64(delta), nil
	}
	delta := dts - pts
	if delta > uint64(1)<<63 {
		return 0, fmt.Errorf("composition timestamp difference cannot be represented")
	}
	if delta == uint64(1)<<63 {
		return -1 << 63, nil
	}
	return -int64(delta), nil
}

// buildRLE32 把 uint32 序列 RLE 成 {sample_count, value} entry。
func buildRLE32(values []uint32) []byte {
	type entry struct {
		count uint32
		value uint32
	}
	entries := make([]entry, 0, 8)
	for _, v := range values {
		if len(entries) > 0 && entries[len(entries)-1].value == v {
			entries[len(entries)-1].count++
			continue
		}
		entries = append(entries, entry{count: 1, value: v})
	}
	payload := make([][]byte, 0, len(entries)*2)
	for _, e := range entries {
		payload = append(payload, mp4U32(e.count), mp4U32(e.value))
	}
	return mp4FullBox("stts", 0, 0, mp4U32(uint32(len(entries))), flatten(payload))
}

// buildRLE32Constant 直接构造只有一个连续值的 RLE 表，避免把 sample_count
// 错当成待编码的样本值。
func buildRLE32Constant(sampleCount, value uint32) []byte {
	return mp4FullBox("stts", 0, 0, mp4U32(1), mp4U32(sampleCount), mp4U32(value))
}

// buildRLE32Signed 把 int64 序列 RLE 成 {sample_count, offset} entry;
// 存在负 offset 时使用 version 1 的 signed 32-bit 字段。
func buildRLE32Signed(values []int64, signed bool) ([]byte, error) {
	type entry struct {
		count uint32
		value int64
	}
	entries := make([]entry, 0, 8)
	for i, v := range values {
		if signed {
			if v < -1<<31 || v > 1<<31-1 {
				return nil, fmt.Errorf("composition offset %d at sample %d exceeds ctts version 1 range", v, i)
			}
		} else if v < 0 || uint64(v) > uint64(^uint32(0)) {
			return nil, fmt.Errorf("composition offset %d at sample %d exceeds ctts version 0 range", v, i)
		}
		if len(entries) > 0 && entries[len(entries)-1].value == v {
			entries[len(entries)-1].count++
			continue
		}
		entries = append(entries, entry{count: 1, value: v})
	}
	payload := make([][]byte, 0, len(entries)*2)
	for _, e := range entries {
		payload = append(payload, mp4U32(e.count))
		if signed {
			// version 1 使用 signed 32-bit offset。
			payload = append(payload, mp4U32(uint32(int32(e.value))))
		} else {
			payload = append(payload, mp4U32(uint32(e.value)))
		}
	}
	version := byte(0)
	if signed {
		version = 1
	}
	return mp4FullBox("ctts", version, 0, mp4U32(uint32(len(entries))), flatten(payload)), nil
}

// buildSTSC 把真实 chunk layout 压缩为 samples_per_chunk 的 RLE entry:
// first_chunk 指向 RLE 区间起点,samples_per_chunk 是该区间每个 chunk 的样本数
// (来自 len(chunk.samples),而不是 1)。sample_description_index 恒为 1。
func buildSTSC(chunks []mp4Chunk) []byte {
	type entry struct {
		firstChunk      uint32
		samplesPerChunk uint32
	}
	var entries []entry
	for index, chunk := range chunks {
		count := uint32(len(chunk.samples))
		if len(entries) > 0 && entries[len(entries)-1].samplesPerChunk == count {
			continue
		}
		entries = append(entries, entry{firstChunk: uint32(index + 1), samplesPerChunk: count})
	}
	payload := make([][]byte, 0, len(entries))
	for _, e := range entries {
		payload = append(payload, mp4U32(e.firstChunk), mp4U32(e.samplesPerChunk), mp4U32(1))
	}
	return mp4FullBox("stsc", 0, 0, mp4U32(uint32(len(entries))), flatten(payload))
}

// buildSTSZ 是 per-sample 表:样本 ≠ chunk,这里仍逐样本列出大小。
func buildSTSZ(samples []mp4SampleMeta) []byte {
	entries := make([][]byte, 0, len(samples))
	for _, s := range samples {
		entries = append(entries, mp4U32(s.Size))
	}
	return mp4FullBox("stsz", 0, 0, mp4U32(0), mp4U32(uint32(len(samples))), flatten(entries))
}

// buildSTCO 为每个真实 chunk 写一个 entry:entry N 是第 N 个当前 track chunk
// 在最终 mdat 数据区内的起始绝对偏移。禁止每 sample 一个 entry。
func buildSTCO(chunks []mp4Chunk) []byte {
	entries := make([][]byte, 0, len(chunks))
	for _, chunk := range chunks {
		entries = append(entries, mp4U32(uint32(chunk.offset)))
	}
	return mp4FullBox("stco", 0, 0, mp4U32(uint32(len(chunks))), flatten(entries))
}

func buildDINF() []byte {
	dref := mp4FullBox("dref", 0, 0, mp4U32(1), mp4FullBox("url ", 0, 1))
	return mp4Box("dinf", dref)
}

// unitMatrix 是 transform matrix 的单位阵(36 字节);
// 中间对角项 d = 0x00010000 必须存在。
func unitMatrix() []byte {
	m := make([]byte, 36)
	m[0], m[1], m[2], m[3] = 0, 0x01, 0x00, 0x00        // a = 0x00010000
	m[16], m[17], m[18], m[19] = 0, 0x01, 0x00, 0x00    // d = 0x00010000(中间对角项)
	m[32], m[33], m[34], m[35] = 0x40, 0x00, 0x00, 0x00 // w = 0x40000000
	return m
}

// buildAVCC 把参数集打包成 avcC(configuration record)。
// 不假定 paramSets[0]=唯一 SPS、paramSets[1]=唯一 PPS:分别收集
// SPS[] 与 PPS[],正确填写 numOfSequenceParameterSets/
// numOfPictureParameterSets。
func buildAVCC(paramSets [][]byte) ([]byte, error) {
	var spsList, ppsList [][]byte
	for _, ps := range paramSets {
		if len(ps) == 0 {
			continue
		}
		switch ps[0] & 0x1F {
		case 7:
			spsList = append(spsList, ps)
		case 8:
			ppsList = append(ppsList, ps)
		}
	}
	if len(spsList) == 0 || len(ppsList) == 0 {
		return nil, fmt.Errorf("avcC requires at least one SPS and one PPS")
	}
	// configurationVersion + AVCProfileIndication + compat + level +
	// 0xFF(6bit reserved + NALULengthSize-1 = 4)。
	payload := []byte{0x01, spsList[0][1], spsList[0][2], spsList[0][3], 0xFF}
	payload = append(payload, 0xE0|byte(len(spsList)&0x1F))
	for _, sps := range spsList {
		payload = append(payload, byte(len(sps)>>8), byte(len(sps)))
		payload = append(payload, sps...)
	}
	payload = append(payload, byte(len(ppsList)))
	for _, pps := range ppsList {
		payload = append(payload, byte(len(pps)>>8), byte(len(pps)))
		payload = append(payload, pps...)
	}
	return mp4Box("avcC", payload), nil
}

// buildESDS 用 ASC 构造 esds(ES_Descriptor → DecoderConfigDescriptor → DecoderSpecificInfo)。
func buildESDS(asc []byte) []byte {
	decoderSpecific := append([]byte{0x05, byte(len(asc))}, asc...)
	decoderConfig := append([]byte{
		0x04, byte(13 + 2 + len(asc)),
		0x40,             // objectTypeIndication: MPEG-4 AAC
		0x15,             // streamType audio(5)<<2 | reserved 1
		0x00, 0x00, 0x00, // bufferSizeDB
		0x00, 0x01, 0x77, 0x00, // maxBitrate
		0x00, 0x01, 0x77, 0x00, // avgBitrate
	}, decoderSpecific...)
	slConfig := []byte{0x06, 0x02, 0x01}
	es := append([]byte{
		0x03, byte(3 + len(decoderConfig) + len(slConfig)),
		0x00, 0x01, 0x00,
	}, decoderConfig...)
	es = append(es, slConfig...)
	return mp4Box("esds", es)
}

// ---- SPS 解析(宽高,Exp-Golomb) ----

// parseSPSDimensions 从 SPS NALU(裸字节,无起始码)提取编码宽高。
func parseSPSDimensions(sps []byte) (uint16, uint16, error) {
	if len(sps) < 4 || sps[0]&0x1F != 7 {
		return 0, 0, fmt.Errorf("not an SPS NALU")
	}
	r := &bitReader{data: sps[1:]}
	profileIDC, err := r.readBits(8)
	if err != nil {
		return 0, 0, err
	}
	if err := r.skipBits(16); err != nil { // constraint flags + level_idc
		return 0, 0, err
	}
	if _, err := r.readUE(); err != nil { // seq_parameter_set_id
		return 0, 0, err
	}
	highProfile := profileIDC == 100 || profileIDC == 110 || profileIDC == 122 ||
		profileIDC == 244 || profileIDC == 44 || profileIDC == 83 || profileIDC == 86 ||
		profileIDC == 118 || profileIDC == 128 || profileIDC == 138
	chromaFormatIDC := uint64(1)
	if highProfile {
		if chromaFormatIDC, err = r.readUE(); err != nil {
			return 0, 0, err
		}
		if chromaFormatIDC == 3 {
			if err := r.skipBits(1); err != nil {
				return 0, 0, err
			}
		}
		if _, err := r.readUE(); err != nil { // bit_depth_luma_minus8
			return 0, 0, err
		}
		if _, err := r.readUE(); err != nil { // bit_depth_chroma_minus8
			return 0, 0, err
		}
		if err := r.skipBits(1); err != nil { // qpprime_y_zero_transform_bypass
			return 0, 0, err
		}
		// seq_scaling_matrix_present_flag 是 1 bit，不是 Exp-Golomb；
		// 每一个 seq_scaling_list_present_flag 同样是 1 bit,只有 flag=1
		// 时才进入对应 scaling list parser。
		scalingPresent, err := r.readBit()
		if err != nil {
			return 0, 0, err
		}
		if scalingPresent == 1 {
			count := 8
			if chromaFormatIDC == 3 {
				count = 12
			}
			for i := 0; i < count; i++ {
				listPresent, err := r.readBit()
				if err != nil {
					return 0, 0, err
				}
				if listPresent == 1 {
					if err := skipScalingList(r, i < 6); err != nil {
						return 0, 0, err
					}
				}
			}
		}
	}
	if _, err := r.readUE(); err != nil { // log2_max_frame_num_minus4
		return 0, 0, err
	}
	picOrderType, err := r.readUE()
	if err != nil {
		return 0, 0, err
	}
	switch picOrderType {
	case 0:
		if _, err := r.readUE(); err != nil {
			return 0, 0, err
		}
	case 1:
		if err := r.skipBits(1); err != nil {
			return 0, 0, err
		}
		if _, err := r.readSE(); err != nil {
			return 0, 0, err
		}
		if _, err := r.readSE(); err != nil {
			return 0, 0, err
		}
		positiveRefs, err := r.readUE()
		if err != nil {
			return 0, 0, err
		}
		for i := 0; i < int(positiveRefs); i++ {
			if _, err := r.readSE(); err != nil {
				return 0, 0, err
			}
		}
	}
	if _, err := r.readUE(); err != nil { // max_num_ref_frames
		return 0, 0, err
	}
	if err := r.skipBits(1); err != nil { // gaps_in_frame_num_value_allowed
		return 0, 0, err
	}
	widthMBS, err := r.readUE()
	if err != nil {
		return 0, 0, err
	}
	heightMapUnits, err := r.readUE()
	if err != nil {
		return 0, 0, err
	}
	frameMBSOnly, err := r.readBit()
	if err != nil {
		return 0, 0, err
	}
	if frameMBSOnly == 0 {
		if err := r.skipBits(1); err != nil { // mb_adaptive_frame_field
			return 0, 0, err
		}
	}
	if err := r.skipBits(1); err != nil { // direct_8x8_inference
		return 0, 0, err
	}
	cropFlag, err := r.readBit()
	if err != nil {
		return 0, 0, err
	}
	width := (uint64(widthMBS) + 1) * 16
	height := (uint64(heightMapUnits) + 1) * 16 * uint64(2-frameMBSOnly)
	if cropFlag == 1 {
		cropLeft, err := r.readUE()
		if err != nil {
			return 0, 0, err
		}
		cropRight, err := r.readUE()
		if err != nil {
			return 0, 0, err
		}
		cropTop, err := r.readUE()
		if err != nil {
			return 0, 0, err
		}
		cropBottom, err := r.readUE()
		if err != nil {
			return 0, 0, err
		}
		subW, subH := chromaCropUnits(chromaFormatIDC, frameMBSOnly)
		width -= (cropLeft + cropRight) * subW
		height -= (cropTop + cropBottom) * subH
	}
	if width == 0 || height == 0 || width > 65535 || height > 65535 {
		return 0, 0, fmt.Errorf("implausible SPS dimensions %dx%d", width, height)
	}
	return uint16(width), uint16(height), nil
}

// skipScalingList 跳过一个 scaling_list 结构。
func skipScalingList(r *bitReader, sizeIs16 bool) error {
	size := 64
	if sizeIs16 {
		size = 16
	}
	lastScale, nextScale := uint64(8), uint64(8)
	for j := 0; j < size; j++ {
		if nextScale != 0 {
			delta, err := r.readSE()
			if err != nil {
				return err
			}
			nextScale = (lastScale + uint64(int64(delta)+256)) % 256
		}
		if nextScale != 0 {
			lastScale = nextScale
		}
	}
	return nil
}

// chromaCropUnits 返回裁剪的水平/垂直子采样单位。
func chromaCropUnits(chromaFormatIDC uint64, frameMBSOnly uint64) (uint64, uint64) {
	switch chromaFormatIDC {
	case 0:
		return 1, 1
	case 2:
		return 2, 2 * (2 - frameMBSOnly)
	case 3:
		return 1, 1 * (2 - frameMBSOnly)
	default: // 1:4:2:0
		return 2, 2 * (2 - frameMBSOnly)
	}
}

// buildMP4 在内存中构造完整 MP4(测试与小文件便捷路径)。
// 生产大文件路径走 writeMP4Stream(spool),两者共享 buildMoov 与交错逻辑。
func buildMP4(video *h264Track, audio *aacTrack) ([]byte, error) {
	videoMeta, err := planVideoTrack(video)
	if err != nil {
		return nil, err
	}
	var audioMeta *mp4TrackMeta
	if audio != nil {
		audioMeta, err = planAudioTrack(audio)
		if err != nil {
			return nil, err
		}
	}
	ftyp := buildFTYP()
	// 两阶段构造：chunk grouping(样本分组)不依赖 mdatStart，只有 stco 数值依赖它。
	// 因此第一遍用占位偏移即可得到最终 moov 尺寸，第二遍再写真实偏移。
	placeholderStart := int64(len(ftyp)) + 1<<20
	videoChunks, audioChunks := interleaveSamples(videoMeta, audioMeta, placeholderStart)
	moov, err := buildMoov(videoMeta, audioMeta, videoChunks, audioChunks, placeholderStart)
	if err != nil {
		return nil, err
	}
	mdatStart := int64(len(ftyp) + len(moov) + 8)
	// 交错 chunk 表与内存 mdat 使用与 spool 路径相同的交错逻辑。
	videoChunks, audioChunks = interleaveSamples(videoMeta, audioMeta, mdatStart)
	moov, err = buildMoov(videoMeta, audioMeta, videoChunks, audioChunks, mdatStart)
	if err != nil {
		return nil, err
	}
	// 样本物理数据在内存中:按 chunk 的样本顺序从 h264Track/aacTrack
	// 取样本数据与 Metadata 序列一致(planVideoTrack/planAudioTrack 顺序)。
	var mdatPayload []byte
	out := append([]byte{}, ftyp...)
	out = append(out, moov...)
	// mdat 数据区按 chunk Offset 顺序写出：视频 chunk 与音频 chunk
	// 各自按 Metadata 序列从内存样本取数据。
	allChunks := append(videoChunks, audioChunks...)
	sort.Slice(allChunks, func(i, j int) bool { return allChunks[i].offset < allChunks[j].offset })
	videoIdx, audioIdx := 0, 0
	mdatPayload = make([]byte, 0)
	for _, chunk := range allChunks {
		if chunk.offset != mdatStart+int64(len(mdatPayload)) {
			return nil, fmt.Errorf("chunk offset %d does not match mdat payload position %d", chunk.offset, mdatStart+int64(len(mdatPayload)))
		}
		if chunk.video {
			for range chunk.samples {
				mdatPayload = append(mdatPayload, video.Samples[videoIdx].Data...)
				videoIdx++
			}
		} else {
			for range chunk.samples {
				mdatPayload = append(mdatPayload, audio.Samples[audioIdx].Data...)
				audioIdx++
			}
		}
	}
	out = append(out, mp4Box("mdat", mdatPayload)...)
	return out, nil
}

type bitReader struct {
	data []byte
	pos  int
}

func (r *bitReader) readBit() (uint64, error) {
	bytePos := r.pos / 8
	if bytePos >= len(r.data) {
		return 0, errSPSTruncated
	}
	bit := (r.data[bytePos] >> (7 - uint(r.pos%8))) & 1
	r.pos++
	return uint64(bit), nil
}

func (r *bitReader) readBits(n int) (uint64, error) {
	var v uint64
	for i := 0; i < n; i++ {
		b, err := r.readBit()
		if err != nil {
			return 0, err
		}
		v = v<<1 | b
	}
	return v, nil
}

// readUE 读取 Exp-Golomb 无符号整数。
func (r *bitReader) readUE() (uint64, error) {
	zeros := 0
	for {
		b, err := r.readBit()
		if err != nil {
			return 0, err
		}
		if b == 1 {
			break
		}
		zeros++
		if zeros > 31 {
			return 0, fmt.Errorf("invalid Exp-Golomb prefix")
		}
	}
	if zeros == 0 {
		return 0, nil
	}
	suffix, err := r.readBits(zeros)
	if err != nil {
		return 0, err
	}
	return (1 << uint(zeros)) - 1 + suffix, nil
}

// readSE 读取 Exp-Golomb 有符号整数。
func (r *bitReader) readSE() (int64, error) {
	ue, err := r.readUE()
	if err != nil {
		return 0, err
	}
	if ue%2 == 0 {
		return -int64(ue / 2), nil
	}
	return int64(ue+1) / 2, nil
}

func (r *bitReader) skipBits(n int) error {
	for i := 0; i < n; i++ {
		if _, err := r.readBit(); err != nil {
			return err
		}
	}
	return nil
}
