package media

import (
	"bytes"
	"strings"
	"testing"
)

// Layer B demux 契约：
// TS → PES → H.264 Annex-B(AVCC 化)与 AAC ADTS(raw 帧 + ASC);
// 时间戳来自 PES(90kHz);不支持的 codec 明确失败;timed ID3 丢弃。

// pesBytes 构造带 PTS(与可选 DTS)的 PES 包体(不含 TS 封装),单位 90kHz。
func pesBytes(streamID byte, pts, dts uint64, hasDTS bool, es []byte) []byte {
	flags := byte(0x80) // PTS_DTS_flags = 10(仅 PTS)
	headerLen := byte(5)
	prefix := byte(0x20)
	if hasDTS {
		flags = 0xC0
		headerLen = 10
		prefix = 0x30
	}
	// byte6='10'xxxxxx 固定标志,byte7=PTS_DTS_flags,byte8=header_data_length。
	pes := []byte{0x00, 0x00, 0x01, streamID, 0x00, 0x00, 0x80, flags, headerLen}
	pes = append(pes, tsMarker(pts, prefix)...)
	if hasDTS {
		pes = append(pes, tsMarker(dts, 0x10)...)
	}
	pesLen := len(pes) - 6 + len(es)
	pes[4] = byte(pesLen >> 8)
	pes[5] = byte(pesLen)
	return append(pes, es...)
}

// tsMarker 按 MPEG-PES 规则编码 5 字节 PTS/DTS。
func tsMarker(value uint64, prefix byte) []byte {
	return []byte{
		prefix | byte((value>>30)&0x07)<<1 | 0x01,
		byte(value >> 22),
		byte((value>>15)&0x7F)<<1 | 0x01,
		byte(value >> 7),
		byte(value&0x7F)<<1 | 0x01,
	}
}

// h264Frame 构造单 NAL 的 H.264 ES 帧(Annex-B 起始码 + NAL header + payload)。
func h264Frame(nalType byte, payload []byte) []byte {
	nal := []byte{(nalType & 0x1F) | 0x60, payload[0], payload[1]}
	out := []byte{0x00, 0x00, 0x00, 0x01}
	return append(out, nal...)
}

// adtsFrame 构造一个 AAC-LC 44.1kHz 双通道 ADTS 帧(7 字节头 + 3 字节 raw),
// frame_length = 10。freqIdx=4(44100);channel config: byte2 低位 0 + byte3 高 2 位 10 = 2。
func adtsFrame() []byte {
	header := []byte{0xFF, 0xF1, 0x50, 0x80, 0x01, 0x40, 0x00}
	return append(header, 0x21, 0x10, 0x30)
}

func TestParseTSExtractsVideoAndAudioStreams(t *testing.T) {
	sps := h264Frame(7, []byte{0xAA, 0xBB})
	idr := h264Frame(5, []byte{0x01, 0x02})
	nonIDR := h264Frame(1, []byte{0x03, 0x04})

	var segment []byte
	segment = append(segment, tsPacket(patPID, true, 0, patSection())...)
	segment = append(segment, tsPacket(pmtPID, true, 0, pmtSection())...)
	segment = append(segment, tsPacket(videoPID, true, 1, pesBytes(0xE0, 90000, 90000, true, sps))...)
	segment = append(segment, tsPacket(videoPID, true, 2, pesBytes(0xE0, 93600, 93600, true, idr))...)
	segment = append(segment, tsPacket(videoPID, true, 3, pesBytes(0xE0, 97200, 97200, true, nonIDR))...)
	segment = append(segment, tsPacket(audioPID, true, 4, pesBytes(0xC0, 90000, 0, false, adtsFrame()))...)

	streams, err := parseTSStreams(segment)
	if err != nil {
		t.Fatalf("parseTSStreams: %v", err)
	}
	video, audio := streams[videoPID], streams[audioPID]
	if video == nil || audio == nil {
		t.Fatalf("missing streams: video=%v audio=%v", video, audio)
	}
	if video.streamType != streamTypeH264 {
		t.Fatalf("video stream type = 0x%02X", video.streamType)
	}
	if len(video.frames) != 3 {
		t.Fatalf("video frames = %d, want 3", len(video.frames))
	}
	if video.frames[0].PTS != 90000 || video.frames[0].DTS != 90000 {
		t.Fatalf("first frame timestamps = %d/%d", video.frames[0].PTS, video.frames[0].DTS)
	}
	if video.frames[2].PTS != 97200 || video.frames[2].DTS != 97200 {
		t.Fatalf("third frame timestamps = %d/%d", video.frames[2].PTS, video.frames[2].DTS)
	}
	if len(audio.frames) != 1 || audio.frames[0].PTS != 90000 {
		t.Fatalf("audio frames = %+v", audio.frames)
	}
}

func TestParseH264ExtractsAVCCAndParameters(t *testing.T) {
	sps := h264Frame(7, []byte{0xAA, 0xBB})
	pps := h264Frame(8, []byte{0xCC, 0xDD})
	idr := h264Frame(5, []byte{0x01, 0x02})
	frame0 := append(append(append([]byte{}, sps...), pps...), idr...)
	frame1 := h264Frame(1, []byte{0x03, 0x04})

	track, err := parseH264Track(demuxedStream{
		streamType: streamTypeH264,
		frames: []demuxedFrame{
			{ES: frame0, PTS: 90000, DTS: 90000},
			{ES: frame1, PTS: 93600, DTS: 93600},
		},
	})
	if err != nil {
		t.Fatalf("parseH264Track: %v", err)
	}
	if len(track.ParamSets) != 2 {
		t.Fatalf("parameter sets = %d, want 2 (SPS+PPS)", len(track.ParamSets))
	}
	if len(track.Samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(track.Samples))
	}
	// 样本剥离 SPS/PPS 后只剩 IDR 帧,且为 AVCC 4 字节大端长度前缀。
	avcc := track.Samples[0].Data
	if len(avcc) != 4+3 || avcc[0] != 0 || avcc[3] != 3 {
		t.Fatalf("first sample AVCC = % X", avcc)
	}
	if track.Samples[0].Sync != true {
		t.Fatal("IDR sample must be sync")
	}
	if track.Samples[1].Sync {
		t.Fatal("non-IDR sample must not be sync")
	}
}

func TestParseAACExtractsRawFramesAndConfig(t *testing.T) {
	track, err := parseAACTrack(demuxedStream{
		streamType: streamTypeAAC,
		frames: []demuxedFrame{
			{ES: adtsFrame(), PTS: 90000},
			{ES: adtsFrame(), PTS: 92160},
		},
	})
	if err != nil {
		t.Fatalf("parseAACTrack: %v", err)
	}
	if len(track.Samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(track.Samples))
	}
	if len(track.Config) != 2 || track.Config[0] != 0x12 {
		t.Fatalf("audio config = % X", track.Config)
	}
	if track.Channels != 2 || track.SampleRate != 44100 {
		t.Fatalf("channels=%d sampleRate=%d", track.Channels, track.SampleRate)
	}
	if len(track.Samples[0].Data) != 3 {
		t.Fatalf("raw frame length = %d, want 3 (ADTS header stripped)", len(track.Samples[0].Data))
	}
}

func TestLayerBRejectsUnsupportedVideoCodec(t *testing.T) {
	// PMT 声明 HEVC(0x24)。
	hevcPMT := psiSection(0x02, []byte{
		0x00, 0x01, 0xC1, 0x00, 0x00, 0xE1, 0x01, 0xF0, 0x00,
		0x24, 0xE1, 0x01, 0xF0, 0x00,
	})
	var segment []byte
	segment = append(segment, tsPacket(patPID, true, 0, patSection())...)
	segment = append(segment, tsPacket(pmtPID, true, 0, hevcPMT)...)
	segment = append(segment, tsPacket(videoPID, true, 1, pesBytes(0xE0, 90000, 90000, false, h264Frame(5, []byte{0x01, 0x02})))...)

	err := validateMediaStream(segment)
	if err == nil || !strings.Contains(err.Error(), "unsupported video codec") {
		t.Fatalf("error = %v, want unsupported video codec", err)
	}
}

func TestLayerBRejectsTimestampRegression(t *testing.T) {
	params := append(append([]byte{}, h264Frame(7, []byte{0xAA, 0xBB})...), h264Frame(8, []byte{0xCC, 0xDD})...)
	var segment []byte
	segment = append(segment, tsPacket(patPID, true, 0, patSection())...)
	segment = append(segment, tsPacket(pmtPID, true, 0, pmtSection())...)
	segment = append(segment, tsPacket(videoPID, true, 1, pesBytes(0xE0, 90000, 90000, true, append(append([]byte{}, params...), h264Frame(5, []byte{0x01, 0x02})...)))...)
	segment = append(segment, tsPacket(videoPID, true, 2, pesBytes(0xE0, 89000, 89000, true, h264Frame(1, []byte{0x03, 0x04})))...)

	err := validateMediaStream(segment)
	if err == nil || !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("error = %v, want timestamp regression", err)
	}
}

// ---- PES 时间戳与长度修正 ----

// PTS-only PES 不能设置 HasDTS;DTS=PTS 但 HasDTS=false。
func TestParsePESPTSOnlyHasNoDTSFlag(t *testing.T) {
	frame, err := parsePESFrame(pesBytes(0xE0, 90000, 0, false, []byte{0x01, 0x02}))
	if err != nil {
		t.Fatalf("parsePESFrame: %v", err)
	}
	if frame.HasDTS {
		t.Fatal("PTS-only PES must not set HasDTS")
	}
	if frame.DTS != frame.PTS {
		t.Fatalf("DTS = %d, want PTS %d", frame.DTS, frame.PTS)
	}
}

// 显式 DTS 的 PES 才有 HasDTS=true。
func TestParsePESWithDTSKeepsFlag(t *testing.T) {
	frame, err := parsePESFrame(pesBytes(0xE0, 93600, 90000, true, []byte{0x01}))
	if err != nil {
		t.Fatalf("parsePESFrame: %v", err)
	}
	if !frame.HasDTS {
		t.Fatal("explicit DTS must set HasDTS")
	}
	if frame.DTS != 90000 || frame.PTS != 93600 {
		t.Fatalf("PTS/DTS = %d/%d", frame.PTS, frame.DTS)
	}
}

// PES_packet_length 大于实际重组长度必须报 truncated,不能继续解析。
func TestParsePESRejectsTruncatedPacketLength(t *testing.T) {
	// 构造 PES_packet_length 声称 1000 字节,实际远少。
	pes := []byte{0x00, 0x00, 0x01, 0xE0, 0x03, 0xE8, 0x80, 0x80, 0x05, 0x21, 0x00, 0x05, 0xBF, 0x21}
	_, err := parsePESFrame(pes)
	if err == nil || !strings.Contains(err.Error(), "truncat") {
		t.Fatalf("error = %v, want truncated PES", err)
	}
}

// ---- ADTS 解析修正 ----

// freqIdx=13..15 是保留值,必须拒绝而非越界 panic。
func TestParseADTSRejectsReservedFrequencyIndex(t *testing.T) {
	for _, idx := range []byte{13, 14, 15} {
		// data[2]: freqIdx 占高 4 bit 高段(bit6-2);freqIdx<<2。
		data := []byte{0xFF, 0xF1, byte(idx << 2), 0x00, 0x01, 0x40, 0x00, 0x21, 0x10, 0x30}
		_, _, _, _, err := parseADTS(data)
		if err == nil || !strings.Contains(err.Error(), "sampling") {
			t.Fatalf("freqIdx %d: error = %v, want reserved sampling frequency error", idx, err)
		}
	}
}

// channel configuration 拼接:byte2 最低位是高 1 bit,byte3 高 2 位是低 2 bit。
// channel=4:high=1(byte2&1), low=0(byte3>>6=0)。
func TestParseADTSChannelConfiguration(t *testing.T) {
	// channel config 4:byte2&1=1,byte3>>6=00。
	data := []byte{0xFF, 0xF1, 0x53, 0x00, 0x01, 0x40, 0x00, 0x21, 0x10, 0x30}
	_, _, channels, _, err := parseADTS(data)
	if err != nil {
		t.Fatalf("parseADTS: %v", err)
	}
	if channels != 4 {
		t.Fatalf("channels = %d, want 4 (byte2 low bit=1, byte3 top=00)", channels)
	}
}

// muxer 只明确支持 mono/stereo:channel configuration 0 或 > 2 必须拒绝，
// 不能产生错误的 MP4；合法的 1/2 应继续通过。
func TestParseAACTrackRejectsMoreThanStereo(t *testing.T) {
	for _, tc := range []struct {
		name     string
		channels byte
		wantErr  bool
	}{
		{name: "unspecified", channels: 0, wantErr: true},
		{name: "mono", channels: 1},
		{name: "stereo", channels: 2},
		{name: "unsupported_multichannel", channels: 4, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := adtsFrame()
			data[2] = (data[2] & 0xFE) | (tc.channels >> 2)
			data[3] = (data[3] & 0x3F) | (tc.channels&0x03)<<6
			track, err := parseAACTrack(demuxedStream{streamType: streamTypeAAC, frames: []demuxedFrame{{ES: data, PTS: 90000}}})
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "channel") {
					t.Fatalf("error = %v, want channel configuration rejection", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAACTrack: %v", err)
			}
			if track.Channels != int(tc.channels) {
				t.Fatalf("channels = %d, want %d", track.Channels, tc.channels)
			}
		})
	}
}

func parseTSStreams(data []byte) (map[uint16]*demuxedStream, error) {
	streams := map[uint16]*demuxedStream{}
	types, err := walkTSFrames(bytes.NewReader(data), func(pid uint16, kind byte, frame demuxedFrame) error {
		if streams[pid] == nil {
			streams[pid] = &demuxedStream{streamType: kind}
		}
		streams[pid].frames = append(streams[pid].frames, frame)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for pid, kind := range types {
		if streams[pid] == nil {
			streams[pid] = &demuxedStream{streamType: kind}
		}
	}
	return streams, nil
}

func parseAACTrack(stream demuxedStream) (*aacTrack, error) {
	var result *aacTrack
	for _, frame := range stream.frames {
		if err := walkAACFrame(frame, func(track *aacTrack, sample aacSample) error {
			if result == nil {
				result = track
			}
			sample.Data = append([]byte(nil), sample.Data...)
			result.Samples = append(result.Samples, sample)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// demuxedStream 是单个 elementary stream 的帧序列。
type demuxedStream struct {
	streamType byte
	frames     []demuxedFrame
}

func parseH264Track(stream demuxedStream) (*h264Track, error) {
	result := &h264Track{}
	for _, frame := range stream.frames {
		track, err := parseH264Frame(frame)
		if err != nil {
			return nil, err
		}
		result.ParamSets = append(result.ParamSets, track.ParamSets...)
		result.Samples = append(result.Samples, track.Samples...)
	}
	return result, nil
}
