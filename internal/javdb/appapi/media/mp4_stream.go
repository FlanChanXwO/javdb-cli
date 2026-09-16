package media

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

// MP4 生产管线(input.md 计划 #35/#36/#37/#40):
// Phase1:逐 segment 下载/解密/Layer A 校验 → demux → 样本写入 spool 文件,
//        内存只保留 moov 级元数据(O(samples) 的紧凑记录);
// Phase2:样本表齐备后按 ftyp → moov → mdat 顺序写出,mdat 从 spool 流式拷贝。
// 临时磁盘峰值 ≈ spool(≈1×输出)+ video.tmp(≈1×输出),换取有界内存与 Fast Start。

// mp4Spooler 把样本数据顺序写入 spool 文件并累积元数据。
// 视频/音频样本交错追加(段内顺序不定),每个样本的 Offset 都记录
// spool 内的全局物理位置,保证读回与写入严格一致。
type mp4Spooler struct {
	file       *os.File
	video      *mp4TrackMeta
	audio      *mp4TrackMeta
	fileOffset int64
}

func newMP4Spooler(dir string) (*mp4Spooler, error) {
	// O_RDWR:Phase2 需要从 spool 读回样本写入 mdat。CreateTemp 避免 stale
	// spool 或同一 target 的并发下载在固定文件名处永久冲突。
	file, err := os.CreateTemp(dir, ".javdb-mp4-spool-*")
	if err != nil {
		return nil, fmt.Errorf("create spool file: %w", err)
	}
	return &mp4Spooler{file: file}, nil
}

// addSegment 解析单个 segment 并把样本追加进 spool。
// 每段都做 codec 检查;首个 segment 建立 codec configuration(H.264 SPS/PPS、
// AAC ASC、宽高、采样率、通道),后续 segment 再次出现配置时必须与首段一致;
// 检测到 change 直接拒绝 remux(计划 #17)。
func (s *mp4Spooler) addSegment(data []byte) error {
	return s.addSegmentReader(bytes.NewReader(data))
}

// addSegmentFile 从 segment 临时文件逐包解析;文件关闭由本方法负责,调用方只需在
// 成功或失败后删除文件。样本 payload 仍按帧写入 spool,不会保留完整 segment。
func (s *mp4Spooler) addSegmentFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	addErr := s.addSegmentReader(file)
	closeErr := file.Close()
	return errors.Join(addErr, closeErr)
}

func (s *mp4Spooler) addSegmentReader(reader io.ReadSeeker) error {
	if err := validateSegmentCodecsReader(reader); err != nil {
		return err
	}
	_, err := walkTSFrames(reader, func(kind byte, frame demuxedFrame) error {
		switch kind {
		case streamTypeH264:
			track, err := parseH264Frame(frame)
			if err != nil {
				return fmt.Errorf("parse H.264 track: %w", err)
			}
			if s.video == nil {
				s.video = &mp4TrackMeta{Timescale: mp4VideoTimescale}
			}
			if err := s.checkVideoConfig(track); err != nil {
				return err
			}
			if err := s.checkVideoTimestamps(track.Samples); err != nil {
				return err
			}
			for _, sample := range track.Samples {
				n, err := s.file.Write(sample.Data)
				if err != nil {
					return err
				}
				if n != len(sample.Data) {
					return io.ErrShortWrite
				}
				s.video.Samples = append(s.video.Samples, mp4SampleMeta{
					SpoolOffset: s.fileOffset, Size: uint32(len(sample.Data)),
					PTS: sample.PTS, DTS: sample.DTS, Sync: sample.Sync,
				})
				s.fileOffset += int64(len(sample.Data))
			}
		case streamTypeAAC:
			return walkAACFrame(frame, func(track *aacTrack, sample aacSample) error {
				if s.audio == nil {
					s.audio = &mp4TrackMeta{
						Timescale:  uint32(track.SampleRate),
						ASC:        track.Config,
						Channels:   track.Channels,
						SampleRate: track.SampleRate,
					}
				} else {
					// 跨 segment codec configuration 校验(计划 #17):
					// 后续 segment 的 AAC ASC/采样率/通道必须与首段一致。
					if err := s.checkAudioConfig(track); err != nil {
						return err
					}
				}

				n, err := s.file.Write(sample.Data)
				if err != nil {
					return err
				}
				if n != len(sample.Data) {
					return io.ErrShortWrite
				}
				s.audio.Samples = append(s.audio.Samples, mp4SampleMeta{
					SpoolOffset: s.fileOffset, Size: uint32(len(sample.Data)), PTS: sample.PTS,
				})
				s.fileOffset += int64(len(sample.Data))
				return nil
			})
		}
		return nil
	})
	return err
}

// checkVideoConfig 校验后续 segment 的 H.264 SPS/PPS 与首段一致(计划 #17)。
func (s *mp4Spooler) checkVideoConfig(track *h264Track) error {
	for _, nalType := range []byte{7, 8} {
		actual := h264ParameterSets(track.ParamSets, nalType)
		if len(actual) == 0 {
			// 某些 segment 只重复一种参数集；缺失本身不代表配置变化。
			continue
		}
		expected := h264ParameterSets(s.video.ParamSets, nalType)
		// SPS/PPS 可以分别出现在不同 PES；第一次出现时建立该类配置。
		if len(expected) == 0 {
			if nalType == 7 {
				w, h, err := parseSPSDimensions(actual[0])
				if err != nil {
					return fmt.Errorf("parse SPS: %w", err)
				}
				s.video.Width, s.video.Height = w, h
			}
			s.video.ParamSets = append(s.video.ParamSets, actual...)
			continue
		}
		if equalH264ParameterSets(expected, actual) {
			continue
		}
		if nalType == 7 {
			w, h, err := parseSPSDimensions(actual[0])
			if err != nil {
				return fmt.Errorf("parse SPS in later segment: %w", err)
			}
			return fmt.Errorf("resolution change detected: SPS differs between segments (first %dx%d, later %dx%d)", s.video.Width, s.video.Height, w, h)
		}
		return fmt.Errorf("PPS changed between segments")
	}
	return nil
}

// checkVideoTimestamps 在写入 spool 前校验新 segment 的 DTS 不回退。
// Layer A 只能发现单个 segment 内的问题，这里补上跨 segment 的全局时间轴约束。
func (s *mp4Spooler) checkVideoTimestamps(samples []h264Sample) error {
	var previous uint64
	hasPrevious := false
	if s.video != nil && len(s.video.Samples) > 0 {
		previous = s.video.Samples[len(s.video.Samples)-1].DTS
		hasPrevious = true
	}
	for _, sample := range samples {
		if hasPrevious && sample.DTS < previous {
			return fmt.Errorf("video timestamp regression across segments: DTS %d < previous DTS %d", sample.DTS, previous)
		}
		previous = sample.DTS
		hasPrevious = true
	}
	return nil
}

// h264ParameterSets 按 NAL 类型筛选参数集，并复制后排序以消除传输顺序差异。
func h264ParameterSets(paramSets [][]byte, nalType byte) [][]byte {
	filtered := make([][]byte, 0, len(paramSets))
	for _, paramSet := range paramSets {
		if len(paramSet) > 0 && paramSet[0]&0x1F == nalType {
			filtered = append(filtered, append([]byte(nil), paramSet...))
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return bytes.Compare(filtered[i], filtered[j]) < 0
	})
	return filtered
}

func equalH264ParameterSets(expected, actual [][]byte) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !bytes.Equal(expected[i], actual[i]) {
			return false
		}
	}
	return true
}

// checkAudioConfig 校验后续 segment 的 AAC ASC/采样率/通道与首段一致(计划 #17)。
func (s *mp4Spooler) checkAudioConfig(track *aacTrack) error {
	if !bytes.Equal(track.Config, s.audio.ASC) {
		return fmt.Errorf("AAC ASC change detected between segments")
	}
	if track.SampleRate != s.audio.SampleRate {
		return fmt.Errorf("audio sample rate change detected: first %d, later %d", s.audio.SampleRate, track.SampleRate)
	}
	if track.Channels != s.audio.Channels {
		return fmt.Errorf("audio channel config change detected: first %d, later %d", s.audio.Channels, track.Channels)
	}
	return nil
}

// finalize 计算 track 时长;文件保持打开,writeMP4Body 需要 ReadAt 读回样本。
// 文件关闭由调用方 downloadMP4 的 defer 负责。
func (s *mp4Spooler) finalize() error {
	if s.video == nil || len(h264ParameterSets(s.video.ParamSets, 7)) == 0 || len(h264ParameterSets(s.video.ParamSets, 8)) == 0 {
		return fmt.Errorf("video track is missing SPS/PPS")
	}

	if s.video != nil {
		durations, duration, err := videoSampleDurations(s.video.Samples)
		if err != nil {
			return err
		}
		s.video.SampleDurations = durations
		s.video.Duration = duration
	}
	if s.audio != nil {
		s.audio.Duration = uint64(len(s.audio.Samples)) * 1024
	}
	return nil
}

// writeMP4Body 完成 Phase2:ftyp → moov → mdat(从 spool 拷贝样本)。
// mdat 使用 chunk 级 A/V interleave(计划 #25):按时间顺序交锳视频/音频
// chunk(约 1 秒)，样本从 spool 按交错顺序拷出，stco 指向交错后的绝对位置。
func writeMP4Body(spool *mp4Spooler, out io.Writer) (int64, error) {
	ftyp := buildFTYP()
	// moov 尺寸与 stco 取值无关:先用占位求尺寸，再用真实 mdatStart 重建。
	moov, err := buildMoov(spool.video, spool.audio, int64(len(ftyp))+1<<20)
	if err != nil {
		return 0, err
	}
	mdatStart := int64(len(ftyp) + len(moov) + 8)
	// 交错 chunk 表：重分配每个样本在 mdat 数据区内的绝对 Offset。
	videoChunks, audioChunks := interleaveSamples(spool.video, spool.audio, mdatStart)
	moov, err = buildMoov(spool.video, spool.audio, mdatStart)
	if err != nil {
		return 0, err
	}
	var mdatPayloadSize int64
	for _, chunk := range append(videoChunks, audioChunks...) {
		for _, sample := range chunk.samples {
			mdatPayloadSize += int64(sample.Size)
		}
	}
	var written int64
	mdatHeader := append(mp4U32(uint32(8+mdatPayloadSize)), "mdat"...)
	for _, chunk := range [][]byte{ftyp, moov, mdatHeader} {
		n, err := out.Write(chunk)
		written += int64(n)
		if err != nil {
			return written, err
		}
		if n != len(chunk) {
			return written, io.ErrShortWrite
		}
	}
	// mdat 数据区按交错顺序写出：先写 videoChunks 与 audioChunks 合并后
	// 按 Offset 排序的顺序，保证 stco 偏移与物理位置一致。
	allChunks := append(videoChunks, audioChunks...)
	sort.Slice(allChunks, func(i, j int) bool { return allChunks[i].offset < allChunks[j].offset })
	for _, chunk := range allChunks {
		n, err := copySamplesFromSpool(spool, chunk.samples, out)
		written += n
		if err != nil {
			return written, fmt.Errorf("copy chunk samples: %w", err)
		}
	}
	if written != int64(len(ftyp)+len(moov)+8)+mdatPayloadSize {
		return written, fmt.Errorf("mp4 body size %d != expected %d", written, int64(len(ftyp)+len(moov)+8)+mdatPayloadSize)
	}
	return written, nil
}

// mp4Chunk 是 mdat 数据区内的一个连续 chunk:同一 track 的一组样本。
type mp4Chunk struct {
	offset  int64
	video   bool
	samples []mp4SampleMeta
}

// interleaveSamples 按 chunk 级 A/V interleave(计划 #25)重分配样本的
// mdat Offset：视频按 GOP(以 sync sample 边界)、音频按约 1 秒时间窗口，
// 按时间顺序交锳。stco 指向交错后的绝对位置，样本顺序由 chunk 表决定。
func interleaveSamples(video, audio *mp4TrackMeta, mdatStart int64) (videoChunks, audioChunks []mp4Chunk) {
	if video == nil {
		return nil, nil
	}
	// chunk 记录样本在 track.Samples 中的下标,便于回写 Offset。
	type pending struct {
		indexes []int
	}
	// 视频 chunk 边界:每个 sync sample 开一个新 chunk(GOP 级)。
	var videoPending []pending
	for i, s := range video.Samples {
		if s.Sync || len(videoPending) == 0 {
			videoPending = append(videoPending, pending{})
		}
		videoPending[len(videoPending)-1].indexes = append(videoPending[len(videoPending)-1].indexes, i)
	}
	// 音频 chunk 边界:约 1 秒时间窗口(以 90kHz 时间轴的 PTS 推进为准)。
	var audioPending []pending
	if audio != nil {
		windowNS := uint64(mp4ChunkInterleaveSeconds * float64(mp4VideoTimescale))
		var chunkStartPTS uint64
		for i, s := range audio.Samples {
			if len(audioPending) == 0 || s.PTS >= chunkStartPTS+windowNS {
				audioPending = append(audioPending, pending{})
				chunkStartPTS = s.PTS
			}
			audioPending[len(audioPending)-1].indexes = append(audioPending[len(audioPending)-1].indexes, i)
		}
	}
	// 按 chunk 首 PTS 交错:视频 chunk 与音频 chunk 依时间顺序交错分配 mdat Offset。
	type chunkRef struct {
		isVideo  bool
		index    int
		startPTS uint64
	}
	var refs []chunkRef
	for i, p := range videoPending {
		if len(p.indexes) == 0 {
			continue
		}
		refs = append(refs, chunkRef{isVideo: true, index: i, startPTS: video.Samples[p.indexes[0]].PTS})
	}
	for i, p := range audioPending {
		if len(p.indexes) == 0 {
			continue
		}
		refs = append(refs, chunkRef{isVideo: false, index: i, startPTS: audio.Samples[p.indexes[0]].PTS})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].startPTS < refs[j].startPTS })
	offset := mdatStart
	for _, ref := range refs {
		var meta *mp4TrackMeta
		var p *pending
		if ref.isVideo {
			meta, p = video, &videoPending[ref.index]
		} else {
			meta, p = audio, &audioPending[ref.index]
		}
		chunk := mp4Chunk{offset: offset, video: ref.isVideo}
		// 重分配每个样本的 Offset 为交错后的绝对位置(写回 meta.Samples,
		// buildSTCO 从 meta.Samples 读取)。
		for _, idx := range p.indexes {
			meta.Samples[idx].Offset = offset
			chunk.samples = append(chunk.samples, meta.Samples[idx])
			offset += int64(meta.Samples[idx].Size)
		}
		if ref.isVideo {
			videoChunks = append(videoChunks, chunk)
		} else {
			audioChunks = append(audioChunks, chunk)
		}
	}
	return videoChunks, audioChunks
}

// copySamplesFromSpool 以固定 I/O 缓冲从 spool 区间复制，避免按 sample 大小分配。
func copySamplesFromSpool(spool *mp4Spooler, samples []mp4SampleMeta, out io.Writer) (int64, error) {
	var written int64
	buffer := make([]byte, 32*1024)
	for _, sample := range samples {
		section := io.NewSectionReader(spool.file, sample.SpoolOffset, int64(sample.Size))
		// 只暴露 Write，避免目的端 ReaderFrom 绕过指定缓冲策略。
		n, err := io.CopyBuffer(struct{ io.Writer }{out}, section, buffer)
		written += n
		if err != nil {
			return written, fmt.Errorf("copy spool at %d: %w", sample.SpoolOffset, err)
		}
		if n != int64(sample.Size) {
			return written, io.ErrUnexpectedEOF
		}
	}
	return written, nil
}

// ---- Layer C:容器完整性(重新解析最终文件,O(moov) 内存) ----

// validateMP4File 重新打开最终文件解析 box 树:
// ftyp → moov → mdat 顺序、avc1/mp4a 轨、duration>0、样本>0、
// stco/stss 越界、stsz 总和与 mdat 一致。通过才允许发布(计划 #40)。
func validateMP4File(path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close MP4 validation file: %w", closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := info.Size()

	var boxes []struct {
		kind string
		off  int64
		size int64
	}
	off := int64(0)
	for {
		var header [8]byte
		if _, err := file.ReadAt(header[:], off); err != nil {
			return fmt.Errorf("read box header at %d: %w", off, err)
		}
		size := int64(uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3]))
		if size < 8 || off+size > fileSize {
			return fmt.Errorf("invalid box size %d at offset %d", size, off)
		}
		boxes = append(boxes, struct {
			kind string
			off  int64
			size int64
		}{kind: string(header[4:8]), off: off, size: size})
		off += size
		if off == fileSize {
			break
		}
	}
	if len(boxes) < 3 || boxes[0].kind != "ftyp" || boxes[1].kind != "moov" || boxes[len(boxes)-1].kind != "mdat" {
		return fmt.Errorf("mp4 box order %v is not ftyp→moov→mdat (Fast Start)", boxKinds(boxes))
	}
	for _, b := range boxes[1 : len(boxes)-1] {
		if b.kind != "moov" {
			return fmt.Errorf("unexpected box %q before mdat", b.kind)
		}
	}

	moov := make([]byte, boxes[1].size)
	if _, err := file.ReadAt(moov, boxes[1].off); err != nil {
		return fmt.Errorf("read moov: %w", err)
	}
	mdat := boxes[len(boxes)-1]
	mdatPayload := mdat.size - 8

	videoOK, audioOK := false, false
	type sampleTable struct {
		trackID         uint32
		mdhdSeen        bool
		mdhdDuration    uint64
		sttsSeen        bool
		sttsSampleCount uint64
		sttsDuration    uint64
		cttsSeen        bool
		cttsSampleCount uint64
		stszSeen        bool
		stszOff         int64
		stszCnt         int64
		stszSampleSize  uint32
		stcoSeen        bool
		stcoOff         int64
		stcoCnt         int64
	}
	var tables []*sampleTable
	var current *sampleTable
	seenTrackIDs := map[uint32]bool{}
	sampleEntryKinds := map[string]bool{}
	if err := walkBoxes(moov, 8, int64(len(moov)), func(ref boxWalker) error {
		switch ref.kind {
		case "trak":
			current = &sampleTable{}
			tables = append(tables, current)
		case "tkhd":
			if current == nil || ref.off+24 > int64(len(moov)) {
				return fmt.Errorf("tkhd is outside a track or truncated")
			}
			// track_ID 唯一(计划 #30):禁止音视频共用 track ID。
			trackID := beU32(moov[ref.off+20 : ref.off+24])
			if trackID == 0 {
				return fmt.Errorf("tkhd track_ID is zero")
			}
			if seenTrackIDs[trackID] {
				return fmt.Errorf("duplicate track ID %d", trackID)
			}
			seenTrackIDs[trackID] = true
			current.trackID = trackID
		case "mdhd":
			if current == nil || ref.off+28 > int64(len(moov)) {
				return fmt.Errorf("mdhd is truncated")
			}
			if current.mdhdSeen {
				return fmt.Errorf("track %d has duplicate mdhd tables", current.trackID)
			}
			timescale := beU32(moov[ref.off+20 : ref.off+24])
			duration := beU32(moov[ref.off+24 : ref.off+28])
			if timescale == 0 || duration == 0 {
				return fmt.Errorf("track duration %d (timescale %d) is not positive", duration, timescale)
			}
			current.mdhdSeen = true
			current.mdhdDuration = uint64(duration)
		case "stts":
			if current == nil {
				return fmt.Errorf("stts is outside a track")
			}
			// stts:[size kind][ver/flags][entry_count][entries(count,delta)]:
			// entries 越出 box 是损坏容器,显式拒绝(计划 #30)。
			// entryCount 可能是损坏的超大值,乘法前先证明上限,避免 int64 溢出。
			tsBoxSize := int64(beU32(moov[ref.off : ref.off+4]))
			if tsBoxSize < 16 || ref.off+tsBoxSize > int64(len(moov)) {
				return fmt.Errorf("stts box is truncated")
			}
			tsEntryCount := uint64(beU32(moov[ref.off+12 : ref.off+16]))
			if tsEntryCount > uint64((tsBoxSize-16)/8) {
				return fmt.Errorf("stts entries (%d) exceed box bounds", tsEntryCount)
			}
			entryEnd := ref.off + 16 + int64(tsEntryCount)*8
			if entryEnd > ref.off+tsBoxSize {
				return fmt.Errorf("stts entries (%d) exceed box bounds", tsEntryCount)
			}
			if current.sttsSeen {
				return fmt.Errorf("track %d has duplicate stts tables", current.trackID)
			}
			var sampleCount uint64
			var timingDuration uint64
			for pos := ref.off + 16; pos < entryEnd; pos += 8 {
				count := uint64(beU32(moov[pos : pos+4]))
				delta := uint64(beU32(moov[pos+4 : pos+8]))
				if ^uint64(0)-sampleCount < count {
					return fmt.Errorf("track %d stts sample count overflows", current.trackID)
				}
				if count != 0 && delta > (^uint64(0)-timingDuration)/count {
					return fmt.Errorf("track %d stts duration overflows", current.trackID)
				}
				sampleCount += count
				timingDuration += count * delta
			}
			current.sttsSeen = true
			current.sttsSampleCount = sampleCount
			current.sttsDuration = timingDuration
		case "ctts":
			if current == nil {
				return fmt.Errorf("ctts is outside a track")
			}
			// ctts:[size kind][version/flags][entry_count][entries(count,offset)]。
			cttsBoxSize := int64(beU32(moov[ref.off : ref.off+4]))
			if cttsBoxSize < 16 || ref.off+cttsBoxSize > int64(len(moov)) {
				return fmt.Errorf("ctts box is truncated")
			}
			version := moov[ref.off+8]
			if version != 0 && version != 1 {
				return fmt.Errorf("track %d has unsupported ctts version %d", current.trackID, version)
			}
			if current.cttsSeen {
				return fmt.Errorf("track %d has duplicate ctts tables", current.trackID)
			}
			entryCount := uint64(beU32(moov[ref.off+12 : ref.off+16]))
			if entryCount > uint64((cttsBoxSize-16)/8) {
				return fmt.Errorf("track %d ctts entries (%d) exceed box bounds", current.trackID, entryCount)
			}
			entryEnd := ref.off + 16 + int64(entryCount)*8
			if entryEnd > ref.off+cttsBoxSize {
				return fmt.Errorf("track %d ctts entries (%d) exceed box bounds", current.trackID, entryCount)
			}
			var sampleCount uint64
			for pos := ref.off + 16; pos < entryEnd; pos += 8 {
				count := uint64(beU32(moov[pos : pos+4]))
				if ^uint64(0)-sampleCount < count {
					return fmt.Errorf("track %d ctts sample count overflows", current.trackID)
				}
				sampleCount += count
			}
			current.cttsSeen = true
			current.cttsSampleCount = sampleCount
		case "stsd":
			// stsd:[size kind][ver/flags][entry_count][sample entry(size+kind+...)]:
			// entry 的 kind 字段在 box 起始 +20。
			if ref.off+20+4 <= int64(len(moov)) {
				sampleEntryKinds[string(moov[ref.off+20:ref.off+24])] = true
			}
		case "stsz":
			if current == nil || ref.off+20 > int64(len(moov)) {
				return fmt.Errorf("stsz is outside a track or truncated")
			}
			if current.stszSeen {
				return fmt.Errorf("track %d has duplicate stsz tables", current.trackID)
			}
			current.stszSeen = true
			current.stszOff = ref.off
			current.stszSampleSize = beU32(moov[ref.off+12 : ref.off+16])
			current.stszCnt = int64(beU32(moov[ref.off+16 : ref.off+20]))
		case "stco":
			if current == nil || ref.off+16 > int64(len(moov)) {
				return fmt.Errorf("stco is outside a track or truncated")
			}
			if current.stcoSeen {
				return fmt.Errorf("track %d has duplicate stco tables", current.trackID)
			}
			// stco:[size kind][ver/flags][entry_count(+12)][entries(+16)]。
			current.stcoSeen = true
			current.stcoOff = ref.off
			current.stcoCnt = int64(beU32(moov[ref.off+12 : ref.off+16]))
		}
		return nil
	}); err != nil {
		return err
	}
	if len(tables) == 0 {
		return fmt.Errorf("moov has no tracks")
	}
	for _, table := range tables {
		if !table.mdhdSeen {
			return fmt.Errorf("track %d missing mdhd", table.trackID)
		}
		if !table.sttsSeen {
			return fmt.Errorf("track %d missing stts timing table", table.trackID)
		}
		if !table.stszSeen {
			return fmt.Errorf("track %d missing stsz sample table", table.trackID)
		}
		if table.sttsSampleCount != uint64(table.stszCnt) {
			return fmt.Errorf("track %d stts sample count %d != stsz sample count %d", table.trackID, table.sttsSampleCount, table.stszCnt)
		}
		if table.sttsDuration != table.mdhdDuration {
			return fmt.Errorf("track %d stts duration %d != mdhd duration %d", table.trackID, table.sttsDuration, table.mdhdDuration)
		}
		if table.cttsSeen && table.cttsSampleCount != uint64(table.stszCnt) {
			return fmt.Errorf("track %d ctts sample count %d != stsz sample count %d", table.trackID, table.cttsSampleCount, table.stszCnt)
		}
		if !table.stcoSeen {
			return fmt.Errorf("track %d missing stco chunk table", table.trackID)
		}
	}
	videoOK = sampleEntryKinds["avc1"]
	audioOK = sampleEntryKinds["mp4a"]
	if !videoOK {
		return fmt.Errorf("moov missing avc1 video track")
	}
	onlyOneTrack, err := hasOnlyOneTrak(moov)
	if err != nil {
		return err
	}
	if !audioOK && !onlyOneTrack {
		return fmt.Errorf("moov missing mp4a audio track")
	}

	// 全部 track 的 stsz 总和必须与 mdat 数据区一致;stco 全部落入边界。
	var totalSamples, totalSize int64
	for _, table := range tables {
		if table.stszSampleSize != 0 {
			totalSize += int64(table.stszSampleSize) * table.stszCnt
			totalSamples += table.stszCnt
			continue
		}
		for i := int64(0); i < table.stszCnt; i++ {
			pos := table.stszOff + 20 + i*4
			if pos+4 > int64(len(moov)) {
				return fmt.Errorf("stsz entries truncated")
			}
			totalSize += int64(beU32(moov[pos : pos+4]))
			totalSamples++
		}
	}
	if totalSamples == 0 {
		return fmt.Errorf("sample table is empty")
	}
	if totalSize != mdatPayload {
		return fmt.Errorf("stsz total %d != mdat payload %d", totalSize, mdatPayload)
	}
	for _, table := range tables {
		for i := int64(0); i < table.stcoCnt; i++ {
			pos := table.stcoOff + 16 + i*4
			if pos+4 > int64(len(moov)) {
				return fmt.Errorf("stco entries truncated")
			}
			offset := int64(beU32(moov[pos : pos+4]))
			if offset < mdat.off+8 || offset >= mdat.off+mdat.size {
				return fmt.Errorf("chunk offset %d out of mdat bounds", offset)
			}
		}
	}
	return nil
}

type boxWalker struct {
	kind string
	off  int64
}

// walkBoxes 遍历 box 树;已知容器 box 递归展开,叶子交给回调。
func walkBoxes(data []byte, start, end int64, fn func(boxWalker) error) error {
	off := start
	for off+8 <= end {
		size := int64(beU32(data[off : off+4]))
		if size < 8 || off+size > end {
			return fmt.Errorf("invalid nested box size %d at %d", size, off)
		}
		kind := string(data[off+4 : off+8])
		if err := fn(boxWalker{kind: kind, off: off}); err != nil {
			return err
		}
		switch kind {
		case "moov", "trak", "mdia", "minf", "stbl", "dinf", "edts":
			if err := walkBoxes(data, off+8, off+size, fn); err != nil {
				return err
			}
		}
		off += size
	}
	if off != end {
		return fmt.Errorf("nested boxes misaligned at %d", off)
	}
	return nil
}

// hasOnlyOneTrak 判断 moov 是否只有一个 trak(video-only 合法,计划 #24)。
func hasOnlyOneTrak(moov []byte) (bool, error) {
	count := 0
	if err := walkBoxes(moov, 8, int64(len(moov)), func(ref boxWalker) error {
		if ref.kind == "trak" {
			count++
		}
		return nil
	}); err != nil {
		return false, err
	}
	return count == 1, nil
}

func boxKinds(boxes []struct {
	kind string
	off  int64
	size int64
}) []string {
	kinds := make([]string, 0, len(boxes))
	for _, b := range boxes {
		kinds = append(kinds, b.kind)
	}
	return kinds
}

func beU32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// ---- .ts 的分块最终结构校验(不整读文件,#36) ----

// tsStreamChunkSize 是最终 .ts 校验的分块大小(188 的整数倍)。
const tsStreamChunkSize = 188 * 21845 // ≈ 4 MB

func validateTSFileStream(path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close TS validation file: %w", closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 || info.Size()%tsPacketSize != 0 {
		return fmt.Errorf("TS output size %d is not %d-byte aligned", info.Size(), tsPacketSize)
	}
	// 逐 packet 校验同步字节即可:PSI/PES 完整性已在 per-segment Layer A
	// 与 codec 检查中覆盖,整文件级重复会因块边界无 PAT 而误报。
	buf := make([]byte, tsStreamChunkSize)
	var off int64
	for {
		n, err := file.ReadAt(buf, off)
		for p := 0; p < n; p += tsPacketSize {
			if buf[p] != 0x47 {
				return fmt.Errorf("final TS validation: invalid sync byte at offset %d", off+int64(p))
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read final TS at offset %d: %w", off, err)
		}
		off += int64(n)
	}
	return nil
}
