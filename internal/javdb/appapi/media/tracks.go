package media

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

// H.264/AAC track 化与 Layer B 媒体完整性(input.md 计划 #24/#26/#34/#39)。
// 只做容器级转换(Annex-B→AVCC、ADTS→raw+ASC),不解码像素/PCM。

// h264Sample 是一个 AVCC 帧(4 字节大端长度前缀的 NALU 序列)。
type h264Sample struct {
	Data []byte
	PTS  uint64
	DTS  uint64
	Sync bool
}

type h264Track struct {
	ParamSets [][]byte // SPS/PPS 裸 NALU(avcC 数据),已从样本剥离
	Samples   []h264Sample
}

type aacSample struct {
	Data []byte // 去除 ADTS 头的 raw AAC 帧
	PTS  uint64
}

type aacTrack struct {
	Config     []byte // AudioSpecificConfig(2 字节)
	Channels   int
	SampleRate int
	Samples    []aacSample
}

// aacFrequencies 是 ADTS sampling_frequency_index → Hz。
var aacFrequencies = [13]int{
	96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050,
	16000, 12000, 11025, 8000, 7350,
}

// aacFrequency 把 sampling_frequency_index 转成 Hz;
// index 13..15 是保留值,不查表,避免越界 panic(计划 #14)。
func aacFrequency(freqIdx byte) (int, error) {
	if int(freqIdx) >= len(aacFrequencies) {
		return 0, fmt.Errorf("reserved ADTS sampling_frequency_index %d", freqIdx)
	}
	return aacFrequencies[freqIdx], nil
}

// parseH264Frame 把一个 H.264 PES 转成 AVCC 样本。
// HLS 惯例是每个 PES 承载一个 access unit,SPS/PPS 在帧前内联,剥离进参数集。
func parseH264Frame(frame demuxedFrame) (*h264Track, error) {
	track := &h264Track{}
	seenParams := map[string]bool{}
	nals, err := splitAnnexBNALs(frame.ES)
	if err != nil {
		return nil, err
	}
	avcc := make([]byte, 0, len(frame.ES))
	sync := false
	for _, nal := range nals {
		switch nalType := nal[0] & 0x1F; nalType {
		case 7, 8: // SPS / PPS
			key := string(nal)
			if !seenParams[key] {
				seenParams[key] = true
				track.ParamSets = append(track.ParamSets, append([]byte(nil), nal...))
			}
			continue
		case 5: // IDR
			sync = true
		}
		avcc = append(avcc, byte(len(nal)>>24), byte(len(nal)>>16), byte(len(nal)>>8), byte(len(nal)))
		avcc = append(avcc, nal...)
	}
	if len(avcc) == 0 {
		return track, nil // 仅参数集的帧不产出样本
	}
	track.Samples = append(track.Samples, h264Sample{Data: avcc, PTS: frame.PTS, DTS: frame.DTS, Sync: sync})
	return track, nil
}

// splitAnnexBNALs 按 Annex-B 起始码(00 00 01 / 00 00 00 01)切分 NALU。
func splitAnnexBNALs(es []byte) ([][]byte, error) {
	var nals [][]byte
	start := -1
	for i := 0; i+2 < len(es); {
		if es[i] == 0 && es[i+1] == 0 && es[i+2] == 1 {
			if start >= 0 {
				nal := trimLeadingZero(es[start:i])
				if len(nal) > 0 {
					nals = append(nals, nal)
				}
			}
			start = i + 3
			i += 3
			continue
		}
		i++
	}
	if start < 0 {
		return nil, fmt.Errorf("H.264 frame has no Annex-B start code")
	}
	nal := trimLeadingZero(es[start:])
	if len(nal) > 0 {
		nals = append(nals, nal)
	}
	return nals, nil
}

// trimLeadingZero 去掉 4 字节起始码残留的额外前导零。
func trimLeadingZero(data []byte) []byte {
	for len(data) > 0 && data[0] == 0 {
		data = data[1:]
	}
	return data
}

// walkAACFrame 逐 ADTS 帧消费借用的 raw payload；回调返回后不持有样本。
// 每帧均传递配置，让消费方同时检查 PES 内和跨 PES/segment 的连续性。
func walkAACFrame(frame demuxedFrame, consume func(*aacTrack, aacSample) error) error {
	es, pts := frame.ES, frame.PTS
	if len(es) == 0 {
		return fmt.Errorf("audio track has no AAC frames")
	}
	for len(es) > 0 {
		payload, freqIdx, channels, consumed, err := parseADTS(es)
		if err != nil {
			return err
		}
		sampleRate, err := aacFrequency(freqIdx)
		if err != nil {
			return err
		}
		if channels > 2 {
			return fmt.Errorf("unsupported ADTS channel configuration %d: only mono/stereo can be remuxed", channels)
		}
		track := &aacTrack{Config: []byte{2<<3 | freqIdx>>1, freqIdx<<7 | channels<<3}, SampleRate: sampleRate, Channels: int(channels)}
		if err := consume(track, aacSample{Data: payload, PTS: pts}); err != nil {
			return err
		}
		pts += uint64(1024) * 90000 / uint64(sampleRate)
		es = es[consumed:]
	}
	return nil
}

// parseADTS 剥离单个 ADTS 帧(protection_absent=1,7 字节头),返回帧总长。
func parseADTS(data []byte) (payload []byte, freqIdx byte, channels byte, consumed int, err error) {
	if len(data) < 7 || data[0] != 0xFF || data[1]&0xF0 != 0xF0 {
		return nil, 0, 0, 0, fmt.Errorf("malformed ADTS frame")
	}
	if data[1]&0x01 != 1 {
		return nil, 0, 0, 0, fmt.Errorf("ADTS CRC frames are not supported")
	}
	freqIdx = (data[2] >> 2) & 0x0F
	if int(freqIdx) >= len(aacFrequencies) {
		return nil, 0, 0, 0, fmt.Errorf("reserved ADTS sampling_frequency_index %d", freqIdx)
	}
	// channel_configuration:byte2 最低 1 bit 是高位,byte3 最高 2 bit 是低位。
	// 旧实现把高位左移 1 位,channel >= 4 会被错误解析(计划 #14)。
	channels = (data[2]&0x01)<<2 | data[3]>>6
	length := int(data[3]&0x03)<<11 | int(data[4])<<3 | int(data[5])>>5
	if length < 7 || length > len(data) {
		return nil, 0, 0, 0, fmt.Errorf("ADTS frame length %d out of bounds", length)
	}
	return data[7:length], freqIdx, channels, length, nil
}

// codecNames 映射已知 stream_type 到可读名称,用于明确的不支持错误。
var codecNames = map[uint16]string{
	0x02: "MPEG-2 video",
	0x10: "MPEG-4 video",
	0x1B: "H.264",
	0x24: "HEVC",
	0x03: "MP3",
	0x04: "MPEG audio",
	0x0F: "AAC",
	0x11: "AAC LATM",
	0x81: "AC-3",
	0x84: "E-AC-3",
	0x87: "E-AC-3",
}

// validateMediaStream 是 Layer B 入口:拆流、分类 codec、校验媒体模型。
// 任一检查失败即拒绝进入 MP4 finalize(input.md 计划 #29/#34)。
func validateMediaStream(data []byte) error {
	if len(data)%tsPacketSize != 0 {
		return fmt.Errorf("TS segment size %d is not %d-byte aligned", len(data), tsPacketSize)
	}
	return validateMediaStreamReader(bytes.NewReader(data))
}

// validateMediaStreamReader 对可回退的 TS reader 执行 Layer B 校验,避免文件模式先
// os.ReadFile 再解析而产生完整媒体 segment 的内存副本。
func validateMediaStreamReader(reader io.ReadSeeker) error {
	var video h264Track
	var audioFrames int
	types, err := walkTSFrames(reader, func(kind byte, frame demuxedFrame) error {
		switch kind {
		case streamTypeH264:
			track, err := parseH264Frame(frame)
			if err != nil {
				return fmt.Errorf("parse H.264 track: %w", err)
			}
			for _, param := range track.ParamSets {
				found := false
				for _, existing := range video.ParamSets {
					if bytes.Equal(param, existing) {
						found = true
						break
					}
				}
				if !found {
					video.ParamSets = append(video.ParamSets, param)
				}
			}
			for _, sample := range track.Samples {
				sample.Data = nil // Layer B 只保留时间戳，不延长媒体 payload 生命周期。
				video.Samples = append(video.Samples, sample)
			}
		case streamTypeAAC:
			if err := walkAACFrame(frame, func(_ *aacTrack, _ aacSample) error { audioFrames++; return nil }); err != nil {
				return fmt.Errorf("parse AAC track: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := validateStreamTypes(types); err != nil {
		return err
	}
	videos, audios := 0, 0
	for _, kind := range types {
		if kind == streamTypeH264 {
			videos++
		}
		if kind == streamTypeAAC {
			audios++
		}
	}
	if videos != 1 {
		return fmt.Errorf("expected exactly one H.264 video track, got %d", videos)
	}
	if len(video.Samples) == 0 {
		return fmt.Errorf("video track has no samples")
	}
	if len(video.ParamSets) < 2 {
		return fmt.Errorf("video track is missing SPS/PPS parameter sets")
	}
	if err := checkMonotonicTimestamps(video.Samples); err != nil {
		return err
	}
	if video.Samples[len(video.Samples)-1].PTS <= video.Samples[0].PTS {
		return fmt.Errorf("video duration is not positive")
	}
	if audios > 0 && audioFrames == 0 {
		return fmt.Errorf("audio track has no samples")
	}
	return nil
}

// checkMonotonicTimestamps 拒绝不可能是真实时间轴的 DTS 回退。
func checkMonotonicTimestamps(samples []h264Sample) error {
	for i := 1; i < len(samples); i++ {
		if samples[i].DTS < samples[i-1].DTS {
			return fmt.Errorf("video timestamp regression at sample %d: DTS %d < %d", i, samples[i].DTS, samples[i-1].DTS)
		}
	}
	return nil
}

func isVideoStreamType(t byte) bool {
	return t == 0x24 || t == 0x1B || t == 0x10 || t == 0x02 || (t >= 0x20 && t <= 0x27)
}

func isAudioStreamType(t byte) bool {
	return t == 0x0F || t == 0x11 || t == 0x03 || t == 0x04 || t == 0x81 || t == 0x84 || t == 0x87
}

// validateSegmentCodecs 是单 segment 的 codec 分类检查(.ts 模式逐段执行):
// 不支持的 codec 明确拒绝;timed ID3 丢弃;参数集等文件级检查不在此处。
func validateSegmentCodecs(data []byte) error {
	if len(data)%tsPacketSize != 0 {
		return fmt.Errorf("TS segment size %d is not %d-byte aligned", len(data), tsPacketSize)
	}
	return validateSegmentCodecsReader(bytes.NewReader(data))
}

// validateSegmentCodecsFile 是单 segment 文件模式的 codec gate,只保留当前 PES 数据
// 与 demux 元数据,不把完整文件读入内存。
func validateSegmentCodecsFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	validateErr := validateSegmentCodecsReader(file)
	closeErr := file.Close()
	return errors.Join(validateErr, closeErr)
}

func validateSegmentCodecsReader(reader io.ReadSeeker) error {
	types, err := walkTSFrames(reader, func(byte, demuxedFrame) error { return nil })
	if err != nil {
		return err
	}
	return validateStreamTypes(types)
}

func validateStreamTypes(types map[uint16]byte) error {
	for _, kind := range types {
		switch kind {
		case streamTypeH264, streamTypeAAC, streamTypeID3:
			continue
		default:
			name := codecNames[uint16(kind)]
			if name == "" {
				name = fmt.Sprintf("0x%02X", kind)
			}
			if isVideoStreamType(kind) {
				return fmt.Errorf("unsupported video codec: %s", name)
			}
			if isAudioStreamType(kind) {
				return fmt.Errorf("unsupported audio codec: %s", name)
			}
			return fmt.Errorf("unsupported stream type %s", name)
		}
	}
	return nil
}
