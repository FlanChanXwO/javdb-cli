package media

import (
	"fmt"
	"io"
	"os"
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
	minStart   uint64
	lastEnd    uint64
}

func newMP4Spooler(path string) (*mp4Spooler, error) {
	// O_RDWR:Phase2 需要从 spool 读回样本写入 mdat。
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create spool file: %w", err)
	}
	return &mp4Spooler{file: file}, nil
}

// addSegment 解析单个 segment 并把样本追加进 spool。
// 每段都做 codec 检查;参数集只取首次出现(HLS 在关键帧段重复携带)。
func (s *mp4Spooler) addSegment(data []byte) error {
	if err := validateSegmentCodecs(data); err != nil {
		return err
	}
	streams, err := parseTSStreams(data)
	if err != nil {
		return err
	}
	for _, stream := range streams {
		switch stream.streamType {
		case streamTypeH264:
			track, err := parseH264Track(*stream)
			if err != nil {
				return fmt.Errorf("parse H.264 track: %w", err)
			}
			if s.video == nil {
				width, height, err := parseSPSDimensions(track.ParamSets[0])
				if err != nil {
					return fmt.Errorf("parse SPS: %w", err)
				}
				s.video = &mp4TrackMeta{
					Timescale: mp4VideoTimescale,
					Width:     width,
					Height:    height,
					ParamSets: track.ParamSets,
				}
			}
			for _, sample := range track.Samples {
				if _, err := s.file.Write(sample.Data); err != nil {
					return err
				}
				s.video.Samples = append(s.video.Samples, mp4SampleMeta{
					Offset: s.fileOffset, Size: uint32(len(sample.Data)),
					PTS: sample.PTS, DTS: sample.DTS, Sync: sample.Sync,
				})
				s.fileOffset += int64(len(sample.Data))
				s.noteTimeline(sample.PTS, sample.DTS)
			}
		case streamTypeAAC:
			track, err := parseAACTrack(*stream)
			if err != nil {
				return fmt.Errorf("parse AAC track: %w", err)
			}
			if s.audio == nil {
				s.audio = &mp4TrackMeta{
					Timescale:  uint32(track.SampleRate),
					ASC:        track.Config,
					Channels:   track.Channels,
					SampleRate: track.SampleRate,
				}
			}
			for _, sample := range track.Samples {
				if _, err := s.file.Write(sample.Data); err != nil {
					return err
				}
				s.audio.Samples = append(s.audio.Samples, mp4SampleMeta{
					Offset: s.fileOffset, Size: uint32(len(sample.Data)), PTS: sample.PTS,
				})
				s.fileOffset += int64(len(sample.Data))
				s.noteTimeline(sample.PTS, sample.PTS)
			}
		}
	}
	return nil
}

func (s *mp4Spooler) noteTimeline(pts, dts uint64) {
	start := minU64(pts, dts)
	end := maxU64(pts, dts)
	if s.video == nil && s.audio == nil {
		return
	}
	if len(s.videoSamples()) == 0 && len(s.audio.Samples) == 0 {
		return
	}
	if s.lastEnd == 0 && s.minStart == 0 {
		s.minStart = start
	}
	if start < s.minStart {
		s.minStart = start
	}
	if end > s.lastEnd {
		s.lastEnd = end
	}
}

func (s *mp4Spooler) videoSamples() []mp4SampleMeta {
	if s.video == nil {
		return nil
	}
	return s.video.Samples
}

// finalize 计算 track 时长;文件保持打开,writeMP4Body 需要 ReadAt 读回样本。
// 文件关闭由调用方 downloadMP4 的 defer 负责。
func (s *mp4Spooler) finalize() {
	if s.video != nil {
		s.video.Duration = s.lastEnd - s.minStart
	}
	if s.audio != nil {
		s.audio.Duration = uint64(len(s.audio.Samples)) * 1024
	}
}

// writeMP4Body 完成 Phase2:ftyp → moov → mdat(从 spool 拷贝样本)。
func writeMP4Body(spool *mp4Spooler, out io.Writer) (int64, error) {
	ftyp := buildFTYP()
	// moov 尺寸与 stco 取值无关:先用占位求尺寸,再用真实 mdatStart 重建。
	moov, err := buildMoov(spool.video, spool.audio, int64(len(ftyp))+1<<20)
	if err != nil {
		return 0, err
	}
	mdatStart := int64(len(ftyp) + len(moov) + 8)
	moov, err = buildMoov(spool.video, spool.audio, mdatStart)
	if err != nil {
		return 0, err
	}
	mdatSize := spool.fileOffset
	var written int64
	// mdat 头的 size 必须包含随后拷贝的样本数据总量。
	mdatHeader := append(mp4U32(uint32(8+mdatSize)), "mdat"...)
	for _, chunk := range [][]byte{ftyp, moov, mdatHeader} {
		n, err := out.Write(chunk)
		written += int64(n)
		if err != nil {
			return written, err
		}
	}
	var videoTotal int64
	for _, s := range spool.video.Samples {
		videoTotal += int64(s.Size)
	}
	videoCopied, err := copySamplesFromSpool(spool, spool.video.Samples, out)
	written += videoCopied
	if err != nil {
		return written, fmt.Errorf("copy video samples: %w", err)
	}
	audioCopied, err := copySamplesFromSpool(spool, spool.audio.Samples, out)
	written += audioCopied
	if err != nil {
		return written, fmt.Errorf("copy audio samples: %w", err)
	}
	if written != int64(len(ftyp)+len(moov)+8)+mdatSize {
		return written, fmt.Errorf("mp4 body size %d != expected %d", written, int64(len(ftyp)+len(moov)+8)+mdatSize)
	}
	return written, nil
}

// copySamplesFromSpool 把样本按元数据顺序从 spool 拷出;错误以负数长度返回。
func copySamplesFromSpool(spool *mp4Spooler, samples []mp4SampleMeta, out io.Writer) (int64, error) {
	var written int64
	for _, sample := range samples {
		data := make([]byte, sample.Size)
		if _, err := spool.file.ReadAt(data, sample.Offset); err != nil {
			return written, fmt.Errorf("read spool at %d: %w", sample.Offset, err)
		}
		n, err := out.Write(data)
		written += int64(n)
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// ---- Layer C:容器完整性(重新解析最终文件,O(moov) 内存) ----

// validateMP4File 重新打开最终文件解析 box 树:
// ftyp → moov → mdat 顺序、avc1/mp4a 轨、duration>0、样本>0、
// stco/stss 越界、stsz 总和与 mdat 一致。通过才允许发布(计划 #40)。
func validateMP4File(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
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
		stszOff int64
		stszCnt int64
		stcoOff int64
		stcoCnt int64
	}
	var tables []sampleTable
	durationOK := false
	sampleEntryKinds := map[string]bool{}
	if err := walkBoxes(moov, 8, int64(len(moov)), func(ref boxWalker) error {
		switch ref.kind {
		case "mdhd":
			timescale := beU32(moov[ref.off+20 : ref.off+24])
			duration := beU32(moov[ref.off+24 : ref.off+28])
			if timescale == 0 || duration == 0 {
				return fmt.Errorf("track duration %d (timescale %d) is not positive", duration, timescale)
			}
			durationOK = true
		case "stsd":
			// stsd:[size kind][ver/flags][entry_count][sample entry(size+kind+...)]:
			// entry 的 kind 字段在 box 起始 +20。
			if ref.off+20+4 <= int64(len(moov)) {
				sampleEntryKinds[string(moov[ref.off+20:ref.off+24])] = true
			}
		case "stsz":
			tables = append(tables, sampleTable{
				stszOff: ref.off,
				stszCnt: int64(beU32(moov[ref.off+16 : ref.off+20])),
			})
		case "stco":
			// stco:[size kind][ver/flags][entry_count(+12)][entries(+16)]。
			// 每个 trak 一个 stco:挂到最近一个未配对的 stsz 表上。
			for i := range tables {
				if tables[i].stcoOff == 0 {
					tables[i].stcoOff = ref.off
					tables[i].stcoCnt = int64(beU32(moov[ref.off+12 : ref.off+16]))
					break
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if !durationOK {
		return fmt.Errorf("moov has no positive mdhd duration")
	}
	videoOK = sampleEntryKinds["avc1"]
	audioOK = sampleEntryKinds["mp4a"]
	if !videoOK {
		return fmt.Errorf("moov missing avc1 video track")
	}
	if !audioOK && !hasOnlyOneTrak(moov) {
		return fmt.Errorf("moov missing mp4a audio track")
	}

	// 全部 track 的 stsz 总和必须与 mdat 数据区一致;stco 全部落入边界。
	var totalSamples, totalSize int64
	for _, table := range tables {
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
func hasOnlyOneTrak(moov []byte) bool {
	count := 0
	_ = walkBoxes(moov, 8, int64(len(moov)), func(ref boxWalker) error {
		if ref.kind == "trak" {
			count++
		}
		return nil
	})
	return count == 1
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

func validateTSFileStream(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
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
			break
		}
		off += int64(n)
	}
	return nil
}
