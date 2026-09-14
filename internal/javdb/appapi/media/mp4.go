package media

import (
	"fmt"
)

// MP4 mux(input.md 计划 #26/#35/#38/#39):纯容器级 remux,不转码。
// 结构固定 ftyp → moov → mdat(Fast Start);时间轴以每个 track 的首个
// 时间戳为零基准,90kHz(视频)与采样率(音频)。
// moov 是纯元数据(O(samples));样本数据本体走 spool,内存不随视频长度线性增长(#36)。

const mp4VideoTimescale = 90000

// mp4SampleMeta 记录单个样本的时间戳与它在 mdat 数据区内的位置。
type mp4SampleMeta struct {
	Offset int64
	Size   uint32
	PTS    uint64
	DTS    uint64
	Sync   bool
}

// mp4TrackMeta 是一个 track 的完整 moov 级元数据。
type mp4TrackMeta struct {
	Timescale  uint32
	Duration   uint64
	Width      uint16
	Height     uint16
	ParamSets  [][]byte // video:SPS/PPS(avcC)
	ASC        []byte   // audio:AudioSpecificConfig(esds)
	Channels   int
	SampleRate int
	Samples    []mp4SampleMeta
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
	meta := &mp4TrackMeta{
		Timescale: mp4VideoTimescale,
		Width:     width,
		Height:    height,
		ParamSets: video.ParamSets,
	}
	var offset int64
	baseDTS := video.Samples[0].DTS
	basePTS := video.Samples[0].PTS
	minStart := minU64(basePTS, baseDTS)
	var lastEnd uint64
	for _, s := range video.Samples {
		meta.Samples = append(meta.Samples, mp4SampleMeta{
			Offset: offset, Size: uint32(len(s.Data)),
			PTS: s.PTS, DTS: s.DTS, Sync: s.Sync,
		})
		offset += int64(len(s.Data))
		if end := maxU64(s.PTS, s.DTS); end > lastEnd {
			lastEnd = end
		}
	}
	meta.Duration = lastEnd - minStart
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
func buildMoov(video, audio *mp4TrackMeta, mdatStart int64) ([]byte, error) {
	if video == nil || len(video.Samples) == 0 {
		return nil, fmt.Errorf("mp4 requires a video track with samples")
	}
	var mvhdDuration uint64
	for _, s := range video.Samples {
		if end := s.PTS; end > mvhdDuration {
			mvhdDuration = end
		}
	}
	if audio != nil && audio.Duration > mvhdDuration {
		mvhdDuration = audio.Duration
	}
	if mvhdDuration == 0 {
		return nil, fmt.Errorf("mp4 duration is zero")
	}
	mvhd := mp4FullBox("mvhd", 0, 0,
		mp4U32(0), mp4U32(0), mp4U32(mp4VideoTimescale),
		mp4U32(uint32(mvhdDuration)),
		mp4U32(0x00010000), mp4U16(0x0100), mp4U16(0),
		mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0),
		mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0), mp4U32(0),
		[]byte{0, 0, 0, 0, 0, 0},
		mp4U16(3),
	)
	videoTrak := buildVideoTrak(video, mdatStart)
	traks := [][]byte{videoTrak}
	if audio != nil {
		// mdat 布局:视频样本连续在前,音频样本紧随其后;
		// 音频 stco 的基址 = mdatStart + 视频样本总字节。
		var videoTotal int64
		for _, s := range video.Samples {
			videoTotal += int64(s.Size)
		}
		traks = append(traks, buildAudioTrak(audio, mdatStart+videoTotal))
	}
	return mp4Box("moov", mvhd, flatten(traks)), nil
}

func buildVideoTrak(meta *mp4TrackMeta, mdatStart int64) []byte {
	duration := meta.Duration
	avcC := buildAVCC(meta.ParamSets)
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
	stts, ctts, stss := buildVideoTimingBoxes(meta.Samples)
	stbl := mp4Box("stbl", stsd, stts, ctts, stss, buildSTSC(len(meta.Samples)), buildSTSZ(meta.Samples), buildSTCO(meta.Samples, mdatStart))
	media := mp4Box("minf",
		mp4FullBox("vmhd", 0, 1, mp4U16(0), mp4U16(0), mp4U16(0), mp4U16(0)),
		buildDINF(), stbl)
	mdhd := mp4FullBox("mdhd", 0, 0,
		mp4U32(0), mp4U32(0), mp4U32(meta.Timescale), mp4U32(uint32(duration)),
		mp4U32(0x55C40000), mp4U16(0), mp4U16(0))
	hdlr := mp4FullBox("hdlr", 0, 0, mp4U32(0), []byte("vide"), mp4U32(0), mp4U32(0), mp4U32(0), append([]byte("VideoHandler"), 0))
	md := mp4Box("mdia", mdhd, hdlr, media)
	tkhd := mp4FullBox("tkhd", 0, 3,
		mp4U32(0), mp4U32(uint32(duration)), mp4U32(1),
		mp4U32(0), mp4U32(0),
		mp4U32(0),
		mp4U16(0), mp4U16(0), mp4U16(0),
		unitMatrix(), mp4U32(uint32(meta.Width)<<16), mp4U32(uint32(meta.Height)<<16))
	return mp4Box("trak", tkhd, md)
}

func buildAudioTrak(meta *mp4TrackMeta, mdatStart int64) []byte {
	duration := meta.Duration
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
	stts := mp4FullBox("stts", 0, 0, mp4U32(1), mp4U32(uint32(len(meta.Samples))), mp4U32(1024))
	stbl := mp4Box("stbl", stsd, stts, buildSTSC(len(meta.Samples)), buildSTSZ(meta.Samples), buildSTCO(meta.Samples, mdatStart))
	media := mp4Box("minf",
		mp4FullBox("smhd", 0, 0, mp4U16(0), mp4U16(0)),
		buildDINF(), stbl)
	mdhd := mp4FullBox("mdhd", 0, 0,
		mp4U32(0), mp4U32(0), mp4U32(meta.Timescale), mp4U32(uint32(duration)),
		mp4U32(0x55C40000), mp4U16(0), mp4U16(0))
	hdlr := mp4FullBox("hdlr", 0, 0, mp4U32(0), []byte("soun"), mp4U32(0), mp4U32(0), mp4U32(0), append([]byte("SoundHandler"), 0))
	md := mp4Box("mdia", mdhd, hdlr, media)
	tkhd := mp4FullBox("tkhd", 0, 3,
		mp4U32(0), mp4U32(uint32(duration)), mp4U32(1),
		mp4U32(0), mp4U32(0),
		mp4U32(0),
		mp4U16(0), mp4U16(0), mp4U16(0x0100),
		unitMatrix(), mp4U32(0), mp4U32(0))
	return mp4Box("trak", tkhd, md)
}

// buildVideoTimingBoxes 产出 stts(DTS delta)、可选 ctts(PTS-DTS)与 stss(sync)。
// 时间戳以首样本为零基准(track 内时间从 0 开始)。
func buildVideoTimingBoxes(samples []mp4SampleMeta) (stts, ctts, stss []byte) {
	baseDTS := samples[0].DTS
	deltas := make([][]byte, 0, len(samples))
	deltas = append(deltas, mp4U32(uint32(baseDTS-baseDTS)))
	prevDTS := baseDTS
	for i := 1; i < len(samples); i++ {
		deltas = append(deltas, mp4U32(uint32(samples[i].DTS-prevDTS)))
		prevDTS = samples[i].DTS
	}
	stts = mp4FullBox("stts", 0, 0, mp4U32(uint32(len(deltas))), flatten(deltas))

	hasCTTS := false
	offs := make([][]byte, 0, len(samples))
	for _, s := range samples {
		offset := int64(s.PTS) - int64(s.DTS)
		if offset < 0 {
			offset = 0
		}
		if offset != 0 {
			hasCTTS = true
		}
		offs = append(offs, mp4U32(uint32(offset)))
	}
	if hasCTTS {
		ctts = mp4FullBox("ctts", 0, 0, mp4U32(uint32(len(offs))), flatten(offs))
	}

	syncs := make([][]byte, 0, 8)
	for i, s := range samples {
		if s.Sync {
			syncs = append(syncs, mp4U32(uint32(i+1)))
		}
	}
	if len(syncs) > 0 && len(syncs) != len(samples) {
		stss = mp4FullBox("stss", 0, 0, mp4U32(uint32(len(syncs))), flatten(syncs))
	}
	return stts, ctts, stss
}

func buildSTSC(count int) []byte {
	return mp4FullBox("stsc", 0, 0, mp4U32(1), mp4U32(1), mp4U32(1), mp4U32(1))
}

func buildSTSZ(samples []mp4SampleMeta) []byte {
	entries := make([][]byte, 0, len(samples))
	for _, s := range samples {
		entries = append(entries, mp4U32(s.Size))
	}
	return mp4FullBox("stsz", 0, 0, mp4U32(0), mp4U32(uint32(len(samples))), flatten(entries))
}

// buildSTCO 的 chunk 偏移指向各 track 在 mdat 数据区内的位置:
// mdat 按 track 分区、样本按 Samples 顺序连续存放,偏移 = 之前样本 Size 之和。
// mp4SampleMeta.Offset 是 spool 内的物理位置,只用于数据拷贝,不用于 stco。
func buildSTCO(samples []mp4SampleMeta, mdatStart int64) []byte {
	entries := make([][]byte, 0, len(samples))
	var off int64
	for _, s := range samples {
		entries = append(entries, mp4U32(uint32(mdatStart+off)))
		off += int64(s.Size)
	}
	return mp4FullBox("stco", 0, 0, mp4U32(uint32(len(samples))), flatten(entries))
}

func buildDINF() []byte {
	dref := mp4FullBox("dref", 0, 0, mp4U32(1), mp4FullBox("url ", 0, 1))
	return mp4Box("dinf", dref)
}

// unitMatrix 是 transform matrix 的单位阵(36 字节)。
func unitMatrix() []byte {
	m := make([]byte, 36)
	m[0], m[1], m[2], m[3] = 0, 0x01, 0x00, 0x00        // 0x00010000
	m[32], m[33], m[34], m[35] = 0x40, 0x00, 0x00, 0x00 // 0x40000000
	return m
}

// buildAVCC 把 SPS/PPS 参数集打包成 avcC(configuration record)。
func buildAVCC(paramSets [][]byte) []byte {
	payload := []byte{0x01, paramSets[0][1], paramSets[0][2], paramSets[0][3], 0xFF, 0xE1}
	for i, ps := range paramSets {
		if i == 1 {
			payload = append(payload, 0x01)
		}
		payload = append(payload, byte(len(ps)>>8), byte(len(ps)))
		payload = append(payload, ps...)
	}
	return mp4Box("avcC", payload)
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
		// seq_scaling_matrix_present_flag 是 1 bit,不是 Exp-Golomb(计划 #16);
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
// 生产大文件路径走 writeMP4Stream(spool),两者共享 buildMoov。
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
	// 先用占位 mdatStart 求 moov 尺寸,再用真实偏移重建。
	// mdatStart 指向 mdat 数据区(box 起始 + 8 字节头),即第一个样本所在文件偏移。
	moov, err := buildMoov(videoMeta, audioMeta, int64(len(ftyp))+1<<20)
	if err != nil {
		return nil, err
	}
	mdatStart := int64(len(ftyp) + len(moov) + 8)
	moov, err = buildMoov(videoMeta, audioMeta, mdatStart)
	if err != nil {
		return nil, err
	}
	out := append([]byte{}, ftyp...)
	out = append(out, moov...)
	var mdatPayload []byte
	for _, s := range video.Samples {
		mdatPayload = append(mdatPayload, s.Data...)
	}
	if audio != nil {
		for _, s := range audio.Samples {
			mdatPayload = append(mdatPayload, s.Data...)
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
		return 0, fmt.Errorf("SPS truncated")
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
