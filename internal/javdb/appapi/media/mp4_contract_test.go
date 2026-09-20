package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MP4 结构契约：
// mvhd/tkhd/mdhd 按 ISO BMFF 字段布局;track ID 唯一(video=1, audio=2);
// stts/ctts 带 sample_count 的 RLE entry;stss 语义正确;avcC 分 SPS[]/PPS[];
// mdat A/V chunk interleave;32-bit 溢出明确拒绝。

// parseTrackBoxes 在 moov 内定位全部 trak 的关键字段。
type trackBoxes struct {
	kind            string // vide / soun
	tkhdSize        uint32
	tkhdID          uint32
	tkhdDur         uint64 // movie timescale
	tkhdWidth       uint32
	tkhdHeight      uint32
	mdhdSize        uint32
	mdhdTS          uint32
	mdhdDur         uint32
	mdhdLanguage    uint16
	mdhdPredefined  uint16
	stszSampleCount uint32
	sttsEntryCount  uint32
	sttsSampleCount uint64
	sttsDelta       uint32
	sttsRaw         []byte
	cttsRaw         []byte // nil 表示省略
	stssRaw         []byte // nil 表示省略
	stscRaw         []byte
	stcoRaw         []byte
	stszRaw         []byte
	avcC            []byte
}

func parseMP4Tracks(t *testing.T, mp4 []byte) []trackBoxes {
	t.Helper()
	var tracks []trackBoxes
	_ = walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind != "trak" {
			return nil
		}
		trak := mp4[ref.off : ref.off+int64(boxSize(mp4, int(ref.off)))]
		tb := trackBoxes{}
		_ = walkBoxes(trak, 8, int64(len(trak)), func(tref boxWalker) error {
			switch tref.kind {
			case "tkhd":
				// box头8 + ver/flags4 + ctime4 + mtime4 + track_ID4 + reserved4 → duration 在 +28
				tb.tkhdSize = boxSize(trak, int(tref.off))
				tb.tkhdID = beU32(trak[tref.off+20 : tref.off+24])
				tb.tkhdDur = uint64(beU32(trak[tref.off+28 : tref.off+32]))
				tb.tkhdWidth = beU32(trak[tref.off+84 : tref.off+88])
				tb.tkhdHeight = beU32(trak[tref.off+88 : tref.off+92])
			case "mdhd":
				tb.mdhdSize = boxSize(trak, int(tref.off))
				tb.mdhdTS = beU32(trak[tref.off+20 : tref.off+24])
				tb.mdhdDur = beU32(trak[tref.off+24 : tref.off+28])
				tb.mdhdLanguage = binary.BigEndian.Uint16(trak[tref.off+28 : tref.off+30])
				tb.mdhdPredefined = binary.BigEndian.Uint16(trak[tref.off+30 : tref.off+32])
			case "stts":
				tb.sttsRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
				tb.sttsEntryCount = beU32(tb.sttsRaw[12:16])
				for off := 16; off+8 <= len(tb.sttsRaw); off += 8 {
					tb.sttsSampleCount += uint64(beU32(tb.sttsRaw[off : off+4]))
					if off == 16 {
						tb.sttsDelta = beU32(tb.sttsRaw[off+4 : off+8])
					}
				}
			case "ctts":
				tb.cttsRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "stss":
				tb.stssRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "stsc":
				tb.stscRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "stco":
				tb.stcoRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "stsz":
				tb.stszRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
				tb.stszSampleCount = beU32(trak[tref.off+16 : tref.off+20])
			case "avcC":
				tb.avcC = append([]byte(nil), trak[tref.off+8:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "hdlr":
				tb.kind = string(trak[tref.off+16 : tref.off+20])
			}
			return nil
		})
		tracks = append(tracks, tb)
		return nil
	})
	return tracks
}

func boxSize(data []byte, off int) uint32 {
	if off+4 > len(data) {
		return 0
	}
	return binary.BigEndian.Uint32(data[off : off+4])
}

// ---- #21:unitMatrix 必须是完整单位矩阵(中间对角项 0x00010000) ----

func TestUnitMatrixIsCompleteIdentity(t *testing.T) {
	m := unitMatrix()
	if len(m) != 36 {
		t.Fatalf("matrix length = %d, want 36", len(m))
	}
	u32 := func(off int) uint32 {
		return binary.BigEndian.Uint32(m[off : off+4])
	}
	// ISO BMFF 顺序为 a/b/u/c/d/v/x/y/w；单位阵仅 a/d/w 非零。
	if got := u32(0); got != 0x00010000 {
		t.Fatalf("a = 0x%08X, want 0x00010000", got)
	}
	if got := u32(16); got != 0x00010000 {
		t.Fatalf("d(中间对角项) = 0x%08X, want 0x00010000", got)
	}
	if got := u32(32); got != 0x40000000 {
		t.Fatalf("w = 0x%08X, want 0x40000000", got)
	}
	// 其余项必须为 0。
	for _, off := range []int{4, 8, 12, 20, 24, 28} {
		if got := u32(off); got != 0 {
			t.Fatalf("matrix[%d] = 0x%08X, want 0", off, got)
		}
	}
}

// ---- #21/#22:mvhd/tkhd/mdhd 布局与时间基 ----

func TestMP4TrackIDsAndTimescales(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	// track_ID 通过 tkhd 校验:video=1, audio=2。
	trackIDs := tTrackIDs(t, mp4)
	if len(trackIDs) != 2 {
		t.Fatalf("track count = %d, want 2", len(trackIDs))
	}
	if trackIDs[0] != 1 {
		t.Fatalf("video track_ID = %d, want 1", trackIDs[0])
	}
	if trackIDs[1] != 2 {
		t.Fatalf("audio track_ID = %d, want 2", trackIDs[1])
	}
	// mdhd:video=90000,audio=sample rate;movie timescale 独立。
	tracks := parseMP4Tracks(t, mp4)
	if tracks[0].mdhdTS != 90000 {
		t.Fatalf("video mdhd timescale = %d, want 90000", tracks[0].mdhdTS)
	}
	if tracks[1].mdhdTS != uint32(audio.SampleRate) {
		t.Fatalf("audio mdhd timescale = %d, want %d", tracks[1].mdhdTS, audio.SampleRate)
	}
	var mvhd boxWalker
	if err := walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind == "tkhd" {
			// version 0 tkhd 的 matrix 位于 box 起点 +48，独立检查最终布局。
			want := [9]uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000}
			for i, value := range want {
				off := ref.off + 48 + int64(i*4)
				if got := beU32(mp4[off : off+4]); got != value {
					t.Errorf("tkhd matrix[%d] = 0x%08X, want 0x%08X", i, got, value)
				}
			}
		}
		if ref.kind == "mvhd" {
			mvhd = ref
		}
		return nil
	}); err != nil {
		t.Fatalf("walk mvhd: %v", err)
	}
	if mvhd.off == 0 || boxSize(mp4, int(mvhd.off)) != 108 {
		t.Fatalf("mvhd size = %d, want 108", boxSize(mp4, int(mvhd.off)))
	}
	if got := beU32(mp4[mvhd.off+20 : mvhd.off+24]); got != mp4MovieTimescale {
		t.Fatalf("mvhd timescale = %d, want %d", got, mp4MovieTimescale)
	}
	if got := beU32(mp4[mvhd.off+104 : mvhd.off+108]); got != mp4NextTrackID {
		t.Fatalf("mvhd next_track_ID = %d, want %d", got, mp4NextTrackID)
	}
	wantWidth, wantHeight, err := parseSPSDimensions(video.ParamSets[0])
	if err != nil {
		t.Fatalf("parse test SPS: %v", err)
	}
	for i, track := range tracks {
		if track.tkhdSize != 92 {
			t.Errorf("track %d tkhd size = %d, want 92", i, track.tkhdSize)
		}
		if track.mdhdSize != 32 {
			t.Errorf("track %d mdhd size = %d, want 32", i, track.mdhdSize)
		}
		if track.mdhdLanguage != 0x55C4 || track.mdhdPredefined != 0 {
			t.Errorf("track %d mdhd language/pre_defined = 0x%04X/0x%04X, want 0x55C4/0", i, track.mdhdLanguage, track.mdhdPredefined)
		}
	}
	if tracks[0].tkhdWidth != uint32(wantWidth)<<16 || tracks[0].tkhdHeight != uint32(wantHeight)<<16 {
		t.Fatalf("video tkhd dimensions = 0x%08X×0x%08X, want %dx%d in 16.16", tracks[0].tkhdWidth, tracks[0].tkhdHeight, wantWidth, wantHeight)
	}
	if tracks[1].tkhdWidth != 0 || tracks[1].tkhdHeight != 0 {
		t.Fatalf("audio tkhd dimensions = 0x%08X×0x%08X, want zero", tracks[1].tkhdWidth, tracks[1].tkhdHeight)
	}
	// tkhd duration 必须换算到 movie timescale,不能直接用 video 90k 时间。
	if tracks[0].tkhdDur == 0 {
		t.Fatal("tkhd duration is zero")
	}
}

func tTrackIDs(t *testing.T, mp4 []byte) []uint32 {
	t.Helper()
	var ids []uint32
	_ = walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind != "tkhd" {
			return nil
		}
		// box 头 8 + ver/flags 4 + ctime 4 + mtime 4 → track_ID 在 +20。
		ids = append(ids, beU32(mp4[ref.off+20:ref.off+24]))
		return nil
	})
	return ids
}

// ---- #19:stts 必须是 {sample_count, sample_delta} 的 RLE entry ----

func TestMP4STTSHasSampleCountEntries(t *testing.T) {
	video, audio := buildTestTracks(t)
	video.Samples = append(video.Samples, h264Sample{
		Data: append([]byte(nil), video.Samples[1].Data...),
		PTS:  video.Samples[1].PTS + 3600,
		DTS:  video.Samples[1].DTS + 3600,
	})
	audio.Samples = append(audio.Samples, aacSample{
		Data: append([]byte(nil), audio.Samples[0].Data...),
		PTS:  audio.Samples[0].PTS + uint64(1024*90000/audio.SampleRate),
	})
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	tracks := parseMP4Tracks(t, mp4)
	// 测试流三帧 DTS 为 0、3600、7200:1 个 entry {3, 3600},而不是 {entry_count=1, delta...}。
	vStts := tracks[0].sttsRaw
	if vStts == nil {
		t.Fatal("video track missing stts")
	}
	entryCount := beU32(vStts[12:16])
	if entryCount != 1 {
		t.Fatalf("stts entry_count = %d, want 1 (RLE of equal deltas)", entryCount)
	}
	sampleCount := beU32(vStts[16:20])
	sampleDelta := beU32(vStts[20:24])
	if sampleCount != 3 || sampleDelta != 3600 {
		t.Fatalf("stts entry = {%d, %d}, want {3, 3600}", sampleCount, sampleDelta)
	}
	if tracks[0].sttsSampleCount != uint64(tracks[0].stszSampleCount) {
		t.Fatalf("video stts sample count = %d, stsz sample count = %d", tracks[0].sttsSampleCount, tracks[0].stszSampleCount)
	}
	if tracks[0].mdhdDur != 10800 {
		t.Fatalf("video mdhd duration = %d, want 10800 from three stts deltas", tracks[0].mdhdDur)
	}
	// 音频 track 同理:N 个 1024 delta → 1 个 entry。
	aStts := tracks[1].sttsRaw
	entryCount = beU32(aStts[12:16])
	sampleCount = beU32(aStts[16:20])
	sampleDelta = beU32(aStts[20:24])
	if entryCount != 1 || sampleCount != 2 || sampleDelta != 1024 {
		t.Fatalf("audio stts entry = {%d, {%d, %d}}, want {1, {2, 1024}}", entryCount, sampleCount, sampleDelta)
	}
	if tracks[1].sttsSampleCount != uint64(len(audio.Samples)) || tracks[1].sttsSampleCount != uint64(tracks[1].stszSampleCount) {
		t.Fatalf("audio stts sample count = %d, stsz sample count = %d, want %d", tracks[1].sttsSampleCount, tracks[1].stszSampleCount, len(audio.Samples))
	}
	if tracks[1].mdhdDur != uint32(len(audio.Samples)*1024) {
		t.Fatalf("audio mdhd duration = %d, want %d samples", tracks[1].mdhdDur, len(audio.Samples)*1024)
	}
}

func TestLayerCRejectsSTTSSampleCountMismatch(t *testing.T) {
	video, audio := buildTestTracks(t)
	audio.Samples = append(audio.Samples, aacSample{
		Data: append([]byte(nil), audio.Samples[0].Data...),
		PTS:  audio.Samples[0].PTS + uint64(1024*90000/audio.SampleRate),
	})
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	var stts []boxWalker
	if err := walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind == "stts" {
			stts = append(stts, ref)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk stts: %v", err)
	}
	if len(stts) != 2 {
		t.Fatalf("stts box count = %d, want 2", len(stts))
	}
	corrupt := append([]byte(nil), mp4...)
	// 第二个 stts 属于音频轨，把其 sample_count 改成与 stsz 不一致的 1。
	binary.BigEndian.PutUint32(corrupt[stts[1].off+16:stts[1].off+20], 1)
	path := filepath.Join(t.TempDir(), "mismatch.mp4")
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(path); err == nil || !strings.Contains(err.Error(), "stts") {
		t.Fatalf("validate mismatch error = %v, want stts sample-count rejection", err)
	}
}

func TestLayerCRejectsSTTSDurationMismatch(t *testing.T) {
	video, audio := buildTestTracks(t)
	video.Samples = append(video.Samples, h264Sample{
		Data: append([]byte(nil), video.Samples[1].Data...),
		PTS:  video.Samples[1].PTS + 3600,
		DTS:  video.Samples[1].DTS + 3600,
	})
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	var videoMDHD boxWalker
	found := false
	if err := walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind == "mdhd" && !found {
			videoMDHD = ref
			found = true
		}
		return nil
	}); err != nil {
		t.Fatalf("walk mdhd: %v", err)
	}
	if !found {
		t.Fatal("video mdhd box not found")
	}
	corrupt := append([]byte(nil), mp4...)
	// 正常视频样本的 stts 总时长为 10800；把 mdhd 改成 7200，Layer C 必须拒绝不一致文件。
	overwriteUint32(corrupt, int(videoMDHD.off)+24, 7200)
	path := filepath.Join(t.TempDir(), "duration-mismatch.mp4")
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(path); err == nil || !strings.Contains(err.Error(), "stts duration") {
		t.Fatalf("validate mismatch error = %v, want stts/mdhd duration rejection", err)
	}
}

// ---- #20:ctts RLE + signed version 1;全零省略 ----

func TestMP4CTTSOmittedWhenAllZero(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	tracks := parseMP4Tracks(t, mp4)
	// 测试流 PTS==DTS:ctts 必须省略。
	if tracks[0].cttsRaw != nil {
		t.Fatalf("ctts must be omitted when all offsets are zero, got % X", tracks[0].cttsRaw[:min(20, len(tracks[0].cttsRaw))])
	}
	_ = audio
}

// B 帧场景:PTS > DTS 的正 offset 用 ctts version 0 RLE。
func TestMP4CTTSRLEPositiveOffsets(t *testing.T) {
	// 手工构造 3 帧:sample1 PTS=DTS, sample2 PTS=DTS+3600(B 帧), sample3 PTS=DTS+1800。
	video := &h264Track{
		ParamSets: [][]byte{
			{0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2},
			{0x68, 0xCC, 0xDD},
		},
		Samples: []h264Sample{
			{Data: avccFrame(0x65, 0x01), PTS: 0, DTS: 0, Sync: true},
			{Data: avccFrame(0x41, 0x02), PTS: 7200, DTS: 3600},
			{Data: avccFrame(0x41, 0x03), PTS: 3600, DTS: 7200},
		},
	}
	meta, err := planVideoTrack(video)
	if err != nil {
		t.Fatalf("planVideoTrack: %v", err)
	}
	stts, ctts, stss, err := buildVideoTimingBoxes(meta.Samples)
	if err != nil {
		t.Fatalf("buildVideoTimingBoxes: %v", err)
	}
	// stts:delta 3600,3600,3600 → 1 个 entry {3, 3600}。
	if ec := beU32(stts[12:16]); ec != 1 {
		t.Fatalf("stts entry_count = %d, want 1", ec)
	}
	if ctts == nil {
		t.Fatal("B-frame samples must produce ctts")
	}
	version := ctts[8]
	if version != 1 {
		t.Fatalf("ctts version = %d, want 1 (sample3 PTS < DTS is a negative offset)", version)
	}
	entryCount := beU32(ctts[12:16])
	if entryCount != 3 {
		t.Fatalf("ctts entry_count = %d, want 3 (offsets 0,7200,-3600 are distinct)", entryCount)
	}
	_ = stss
}

func TestBuildMP4RejectsCompositionOffsetOutsideCTTSRange(t *testing.T) {
	validParamSets := [][]byte{
		{0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2},
		{0x68, 0xCC, 0xDD},
	}
	cases := []struct {
		name      string
		firstPTS  uint64
		firstDTS  uint64
		secondPTS uint64
		secondDTS uint64
		wantInErr string
	}{
		{
			name:      "version 0 positive overflow",
			secondPTS: uint64(^uint32(0)) + 3601,
			secondDTS: 3600,
			wantInErr: "ctts version 0 range",
		},
		{
			name:      "version 1 negative overflow",
			firstDTS:  uint64(1<<31) + 1,
			secondDTS: uint64(1<<31) + 3601,
			wantInErr: "ctts version 1 range",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			video := &h264Track{
				ParamSets: validParamSets,
				Samples: []h264Sample{
					{Data: avccFrame(0x65, 0x01), PTS: tc.firstPTS, DTS: tc.firstDTS, Sync: true},
					{Data: avccFrame(0x41, 0x02), PTS: tc.secondPTS, DTS: tc.secondDTS},
				},
			}
			_, err := buildMP4(video, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("buildMP4 error = %v, want %q", err, tc.wantInErr)
			}
		})
	}
}

func TestLayerCRejectsCTTSSampleCountMismatch(t *testing.T) {
	video, audio := buildTestTracks(t)
	video.Samples[1].PTS += 3600
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	var ctts boxWalker
	found := false
	if err := walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind == "ctts" && !found {
			ctts = ref
			found = true
		}
		return nil
	}); err != nil {
		t.Fatalf("walk ctts: %v", err)
	}
	if !found {
		t.Fatal("ctts box not found")
	}
	corrupt := append([]byte(nil), mp4...)
	// ctts entry_count 改为只保留第一个 entry，累计 sample_count 应与 stsz 不一致。
	overwriteUint32(corrupt, int(ctts.off)+12, 1)
	overwriteUint32(corrupt, int(ctts.off)+16, 1)
	path := filepath.Join(t.TempDir(), "ctts-count-mismatch.mp4")
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(path); err == nil || !strings.Contains(err.Error(), "ctts sample count") {
		t.Fatalf("validate mismatch error = %v, want ctts sample-count rejection", err)
	}
}

// ---- #23:完全没有 sync sample 时必须拒绝生成 MP4 ----

func TestPlanVideoTrackRejectsNoSyncSamples(t *testing.T) {
	video := &h264Track{
		ParamSets: [][]byte{
			{0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2},
			{0x68, 0xCC, 0xDD},
		},
		Samples: []h264Sample{
			{Data: avccFrame(0x41, 0x01), PTS: 0, DTS: 0, Sync: false},
			{Data: avccFrame(0x41, 0x02), PTS: 3600, DTS: 3600, Sync: false},
		},
	}
	_, err := planVideoTrack(video)
	if err == nil || !strings.Contains(err.Error(), "sync") {
		t.Fatalf("error = %v, want no sync sample rejection", err)
	}
}

func TestPlanVideoTrackRejectsSingleTimestampedSample(t *testing.T) {
	video, _ := buildTestTracks(t)
	video.Samples = video.Samples[:1]
	_, err := planVideoTrack(video)
	if err == nil || !strings.Contains(err.Error(), "at least two timestamped samples") {
		t.Fatalf("error = %v, want single-sample duration rejection", err)
	}
}

// ---- #24:avcC 分 SPS[]/PPS[] 收集 ----

func TestBuildAVCCCollectsSPSAndPPSSeparately(t *testing.T) {
	sps1 := []byte{0x67, 0x42, 0x00, 0x28}
	sps2 := []byte{0x67, 0x42, 0x00, 0x29}
	pps1 := []byte{0x68, 0xCC, 0xDD}
	pps2 := []byte{0x68, 0xCC, 0xEE}
	avcC, err := buildAVCC([][]byte{sps1, pps1, sps2, pps2})
	if err != nil {
		t.Fatalf("buildAVCC: %v", err)
	}
	if len(avcC) < 8 {
		t.Fatalf("avcC too short: % X", avcC)
	}
	payload := avcC[8:]
	if payload[0] != 0x01 {
		t.Fatalf("configurationVersion = %d", payload[0])
	}
	numSPS := payload[5] & 0x1F
	if numSPS != 2 {
		t.Fatalf("numOfSequenceParameterSets = %d, want 2", numSPS)
	}
	// 第一个 SPS 长度 = 4。
	spsLen := int(binary.BigEndian.Uint16(payload[6:8]))
	if spsLen != len(sps1) {
		t.Fatalf("first SPS length = %d, want %d", spsLen, len(sps1))
	}
	// 扫描到 PPS 区:numOfPictureParameterSets 必须为 2。
	// payload 布局:[0]=configVersion,[1..3] profile/compat/level,[4]=0xFF,
	// [5]=0xE0|numSPS,之后逐 SPS(len 2B + 数据),最后是 numPPS 计数。
	pos := 6 + (2 + len(sps1)) + (2 + len(sps2))
	if payload[pos] != byte(len(pps1)+len(pps2)) && payload[pos] != 2 {
		t.Fatalf("expected numOfPictureParameterSets at %d, got 0x%02X", pos, payload[pos])
	}
	numPPS := payload[pos]
	if numPPS != 2 {
		t.Fatalf("numOfPictureParameterSets = %d, want 2", numPPS)
	}
}

// ---- #18:video-only MP4 正常(audio 为 nil) ----

func TestBuildMP4VideoOnlyNoPanic(t *testing.T) {
	video, _ := buildTestTracks(t)
	mp4, err := buildMP4(video, nil)
	if err != nil {
		t.Fatalf("buildMP4 video-only: %v", err)
	}
	if len(mp4) == 0 {
		t.Fatal("video-only mp4 is empty")
	}
	tracks := parseMP4Tracks(t, mp4)
	if len(tracks) != 1 || tracks[0].kind != "vide" {
		t.Fatalf("video-only tracks = %v", tracks)
	}
}

// ---- #26:32-bit 溢出明确拒绝 ----

func TestBuildMoovRejects32BitOverflow(t *testing.T) {
	// mdatStart 接近 4 GiB:stco 偏移会溢出 32-bit。
	videoRaw, audioRaw := buildTestTracks(t)
	video, err := planVideoTrack(videoRaw)
	if err != nil {
		t.Fatal(err)
	}
	audio, err := planAudioTrack(audioRaw)
	if err != nil {
		t.Fatal(err)
	}
	const overflowStart = int64(0xFFFFFFF0)
	videoChunks, audioChunks := interleaveSamples(video, audio, overflowStart)
	_, err = buildMoov(video, audio, videoChunks, audioChunks, overflowStart)
	if err == nil || !strings.Contains(err.Error(), "32") {
		t.Fatalf("error = %v, want 32-bit overflow rejection", err)
	}
}

// ---- #25:mdat A/V chunk interleave 与最终 stsc/stco 一致 ----

// multiChunkTestTracks 在真实 SPS/PPS 之上扩散出多 GOP 视频与约 1.28 秒音频：
// 每个 chunk 必须含多个样本，否则 "1 sample = 1 chunk" 的坏实现也会偶然通过。
func multiChunkTestTracks(t *testing.T) (*h264Track, *aacTrack) {
	t.Helper()
	video, audio := buildTestTracks(t)
	baseVideo := append([]h264Sample(nil), video.Samples...)
	baseAudio := append([]aacSample(nil), audio.Samples...)

	// 3 个 GOP,每 GOP 2 帧(首帧 sync);DTS/PTS 以 3600(90kHz 下 40ms)推进。
	video.Samples = nil
	for gop := 0; gop < 3; gop++ {
		for i, sample := range baseVideo {
			video.Samples = append(video.Samples, h264Sample{
				Data: append([]byte(nil), sample.Data...),
				PTS:  uint64(gop*len(baseVideo)+i) * 3600,
				DTS:  uint64(gop*len(baseVideo)+i) * 3600,
				Sync: i == 0,
			})
		}
	}

	// 约 1.28 秒音频:每样本 1024 采样,PTS 步进 1024*90000/sampleRate;
	// 1 秒窗口应产生远少于样本数的 audio chunk。
	audio.Samples = nil
	step := uint64(1024) * 90000 / uint64(audio.SampleRate)
	for i := 0; i < 60; i++ {
		audio.Samples = append(audio.Samples, aacSample{
			Data: append([]byte(nil), baseAudio[0].Data...),
			PTS:  uint64(i) * step,
		})
	}
	return video, audio
}

// expandSTSC 把 stsc 展开为每个 chunk 的 samples_per_chunk。
func expandSTSC(stsc []byte, chunkCount uint32) []uint32 {
	entryCount := beU32(stsc[12:16])
	firsts := make([]uint32, 0, entryCount)
	perChunk := make([]uint32, 0, entryCount)
	for i := uint32(0); i < entryCount; i++ {
		off := 16 + int(i)*12
		firsts = append(firsts, beU32(stsc[off:off+4]))
		perChunk = append(perChunk, beU32(stsc[off+4:off+8]))
	}
	var out []uint32
	for i := range firsts {
		last := chunkCount
		if i+1 < len(firsts) {
			last = firsts[i+1] - 1
		}
		for chunk := firsts[i]; chunk <= last; chunk++ {
			out = append(out, perChunk[i])
		}
	}
	return out
}

// spoolTestMP4 把内存 track 经生产 spool 路径写出 MP4。
func spoolTestMP4(t *testing.T, video *h264Track, audio *aacTrack) []byte {
	t.Helper()
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer spool.file.Close()
	spool.video, err = planVideoTrack(video)
	if err != nil {
		t.Fatal(err)
	}
	if audio != nil {
		spool.audio, err = planAudioTrack(audio)
		if err != nil {
			t.Fatal(err)
		}
	}
	// 按 track 原始顺序写入 spool 字节，并记录样本在 spool 内的位置。
	writeTrack := func(samples []mp4SampleMeta, data func(i int) []byte) error {
		for i := range samples {
			blob := data(i)
			samples[i].SpoolOffset = spool.fileOffset
			if _, err := spool.file.WriteAt(blob, spool.fileOffset); err != nil {
				return err
			}
			spool.fileOffset += int64(len(blob))
		}
		return nil
	}
	if err := writeTrack(spool.video.Samples, func(i int) []byte { return video.Samples[i].Data }); err != nil {
		t.Fatal(err)
	}
	if spool.audio != nil {
		if err := writeTrack(spool.audio.Samples, func(i int) []byte { return audio.Samples[i].Data }); err != nil {
			t.Fatal(err)
		}
	}
	if err := spool.finalize(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := writeMP4Body(spool, &out); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// assertChunkTablesMatchMDAT 校验一个最终 MP4 的 stsc/stco 与 mdat 基本契约：
// stco 一项对应一个真实 chunk(不是 sample),stsc 描述真实 samples_per_chunk,
// 展开后样本总数等于 stsz sample_count,每个 track 的 stco 偏移递增且落在 mdat 内。
func assertChunkTablesMatchMDAT(t *testing.T, mp4 []byte) {
	t.Helper()
	boxes := scanTopLevelBoxes(t, mp4)
	mdat := boxes[len(boxes)-1]
	if mdat.kind != "mdat" {
		t.Fatalf("last box = %q, want mdat", mdat.kind)
	}
	mdatPayloadStart := int64(mdat.off + 8)
	mdatPayloadEnd := int64(mdat.off + mdat.size)

	tracks := parseMP4Tracks(t, mp4)
	if len(tracks) != 2 {
		t.Fatalf("tracks = %d, want video+audio", len(tracks))
	}
	multiSampleChunks := false
	for _, track := range tracks {
		if track.stscRaw == nil || track.stcoRaw == nil || track.stszRaw == nil {
			t.Fatalf("track %q is missing stsc/stco/stsz", track.kind)
		}
		chunkCount := beU32(track.stcoRaw[12:16])
		sampleCount := track.stszSampleCount
		if chunkCount == 0 {
			t.Fatalf("track %q has zero chunks", track.kind)
		}
		samplesPerChunk := expandSTSC(track.stscRaw, chunkCount)
		if uint32(len(samplesPerChunk)) != chunkCount {
			t.Fatalf("track %q stsc expands to %d chunks, stco has %d", track.kind, len(samplesPerChunk), chunkCount)
		}
		var expanded uint64
		for _, count := range samplesPerChunk {
			if count == 0 {
				t.Fatalf("track %q has a zero-sample chunk", track.kind)
			}
			expanded += uint64(count)
			if count > 1 {
				multiSampleChunks = true
			}
		}
		if expanded != uint64(sampleCount) {
			t.Fatalf("track %q stsc expands to %d samples, stsz says %d", track.kind, expanded, sampleCount)
		}
		// 错误实现会把每个 sample 伪装成一个 chunk,这里必须直接红灯。
		if chunkCount >= sampleCount {
			t.Fatalf("track %q has %d chunks for %d samples; chunk table still describes samples as independent chunks", track.kind, chunkCount, sampleCount)
		}
		var previousOffset int64
		for i := uint32(0); i < chunkCount; i++ {
			offset := int64(beU32(track.stcoRaw[16+int(i)*4 : 20+int(i)*4]))
			if offset < mdatPayloadStart || offset >= mdatPayloadEnd {
				t.Fatalf("track %q chunk %d offset %d outside mdat payload [%d,%d)", track.kind, i, offset, mdatPayloadStart, mdatPayloadEnd)
			}
			if i > 0 && offset <= previousOffset {
				t.Fatalf("track %q stco offset %d (%d) is not strictly increasing", track.kind, i, offset)
			}
			previousOffset = offset
		}
	}
	if !multiSampleChunks {
		t.Fatal("fixture produced only single-sample chunks; it cannot detect the 1-sample=1-chunk regression")
	}
}

// TestMP4ChunkTablesMatchInterleavedMDAT 是最终容器的 chunk-table contract。
// 这里只覆盖生产 writeMP4Body(spool) 路径，完整 mdat 字节布局由 Layer C 校验。
func TestMP4ChunkTablesMatchInterleavedMDAT(t *testing.T) {
	video, audio := multiChunkTestTracks(t)
	assertChunkTablesMatchMDAT(t, spoolTestMP4(t, video, audio))
}

func TestValidateMP4RejectsBrokenChunkTable(t *testing.T) {
	video, audio := multiChunkTestTracks(t)
	mp4 := spoolTestMP4(t, video, audio)
	mutated := append([]byte(nil), mp4...)
	mutatedSTSC := false
	if err := walkBoxes(mutated, 0, int64(len(mutated)), func(ref boxWalker) error {
		if ref.kind == "stsc" && !mutatedSTSC {
			original := beU32(mutated[ref.off+20 : ref.off+24])
			overwriteUint32(mutated, int(ref.off)+20, original+1)
			mutatedSTSC = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !mutatedSTSC {
		t.Fatal("stsc not found")
	}
	tmp := t.TempDir() + "/broken-stsc.mp4"
	if err := writeFile(tmp, mutated); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("broken stsc sample mapping must be rejected")
	}
}

// ---- #30:validator 独立检查 stts/ctts/stco/track ID/interleave ----

func TestValidateMP4RejectsBadSTTSEntryCount(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	// 篡改 video stts 的 entry_count 为超大值,使 stts entries 越出 box。
	stcoOff := bytes.Index(mp4, []byte("stts"))
	if stcoOff < 0 {
		t.Fatal("stts not found")
	}
	overwriteUint32(mp4, stcoOff+8, 0xFFFFFFF0)
	tmp := t.TempDir() + "/v.mp4"
	if err := writeFile(tmp, mp4); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("oversized stts entry_count must be rejected")
	}
}

func TestValidateMP4RejectsSingleTrackIDCollision(t *testing.T) {
	// 两个 track 都用 track ID 1:validator 必须拒绝。
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	// 把第二个 tkhd 的 track_ID 改成 1(与第一个冲突)。
	seen := 0
	_ = walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind != "tkhd" {
			return nil
		}
		seen++
		if seen == 2 {
			overwriteUint32(mp4, int(ref.off)+20, 1)
		}
		return nil
	})
	tmp := t.TempDir() + "/v.mp4"
	if err := writeFile(tmp, mp4); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("duplicate track ID must be rejected")
	}
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func avccFrame(nalType, payload byte) []byte {
	nal := []byte{(nalType & 0x1F) | 0x60, payload, payload}
	out := make([]byte, 4+len(nal))
	binary.BigEndian.PutUint32(out, uint32(len(nal)))
	copy(out[4:], nal)
	return out
}

// ---- #17:校验跨 segment codec configuration ----

// 后续 segment 出现不同 SPS 必须拒绝 remux。
func TestSpoolerRejectsCrossSegmentSPSChange(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(spool.file.Name())
	defer spool.file.Close()
	if err := spool.addSegment(validTSSegmentAt(0)); err != nil {
		t.Fatalf("first segment: %v", err)
	}
	// 第二个 segment 携带不同 SPS(不同分辨率)。
	// 构造不同 SPS 的 segment:640x480 → 320x240。
	altSegment := validTSSegmentWithSPS(t, []byte{0x67, 0x42, 0x00, 0x1E, 0xF8, 0x14, 0x07, 0xB2})
	err = spool.addSegment(altSegment)
	if err == nil || !strings.Contains(err.Error(), "SPS") {
		t.Fatalf("error = %v, want SPS change rejection", err)
	}
}

func TestSpoolerRejectsCrossSegmentDTSRegression(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(spool.file.Name())
	defer spool.file.Close()
	if err := spool.addSegment(validTSSegmentAt(0)); err != nil {
		t.Fatalf("first segment: %v", err)
	}
	if err := spool.addSegment(validTSSegmentAt(0)); err == nil || !strings.Contains(err.Error(), "timestamp regression") {
		t.Fatalf("error = %v, want cross-segment timestamp regression", err)
	}
}

func TestSpoolerRejectsCrossSegmentPPSChange(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(spool.file.Name())
	defer spool.file.Close()
	sps := []byte{0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2}
	if err := spool.addSegment(validTSSegmentWithSPSAndPPS(t, sps, []byte{0xCC, 0xDD})); err != nil {
		t.Fatalf("first segment: %v", err)
	}
	err = spool.addSegment(validTSSegmentWithSPSAndPPS(t, sps, []byte{0xEE, 0xFF}))
	if err == nil || !strings.Contains(err.Error(), "PPS") {
		t.Fatalf("error = %v, want PPS change rejection", err)
	}
}

// validTSSegmentWithSPS 构造携带指定 SPS 的合法 segment。
func validTSSegmentWithSPS(t *testing.T, sps []byte) []byte {
	return validTSSegmentWithSPSAndPPS(t, sps, []byte{0xCC, 0xDD})
}

func validTSSegmentWithSPSAndPPS(t *testing.T, sps, ppsPayload []byte) []byte {
	t.Helper()
	pps := h264Frame(8, ppsPayload)
	frame0 := append(append(append([]byte{}, append([]byte{0x00, 0x00, 0x00, 0x01}, sps...)...), pps...), h264Frame(5, []byte{0x01, 0x02})...)
	frame1 := h264Frame(1, []byte{0x03, 0x04})
	var data []byte
	data = append(data, tsPacket(patPID, true, 0, patSection())...)
	data = append(data, tsPacket(pmtPID, true, 0, pmtSection())...)
	data = append(data, tsPacket(videoPID, true, 1, pesBytes(0xE0, 90000, 90000, true, frame0))...)
	data = append(data, tsPacket(videoPID, true, 2, pesBytes(0xE0, 93600, 93600, true, frame1))...)
	data = append(data, tsPacket(audioPID, true, 3, pesBytes(0xC0, 90000, 0, false, adtsFrame()))...)
	return data
}

// 相同 codec configuration 也不能把不同 PID 的 elementary stream 合成一轨。
func TestMP4RejectsMultipleMediaPIDs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extraKind byte
		wantError string
	}{
		{"two_video", streamTypeH264, "multiple H.264 video tracks"},
		{"two_audio", streamTypeAAC, "multiple AAC audio tracks"},
		{"single_video_and_audio", 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			segment := validTSSegment()
			if tc.extraKind != 0 {
				body := []byte{0, 1, 0xC1, 0, 0, 0xE1, 1, 0xF0, 0,
					streamTypeH264, 0xE1, 1, 0xF0, 0, streamTypeAAC, 0xE1, 2, 0xF0, 0,
					tc.extraKind, 0xE1, 3, 0xF0, 0}
				copy(segment[188:376], tsPacket(pmtPID, true, 0, psiSection(2, body)))
				packetIndex := 2
				if tc.extraKind == streamTypeAAC {
					packetIndex = 4
				}
				duplicate := append([]byte(nil), segment[packetIndex*188:(packetIndex+1)*188]...)
				duplicate[2] = 3 // PID 0x0103，ES 内容与原轨一致。
				segment = append(segment, duplicate...)
			}
			spool, err := newMP4Spooler(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer spool.file.Close()
			err = spool.addSegmentReader(bytes.NewReader(segment))
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %s", err, tc.wantError)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TS 保留 ADTS 原文；MP4 只接受能按 AAC-LC/1024 samples 正确封装的帧。
func TestAACRemuxBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name            string
		profile, blocks byte
		channels        byte
		wantError       string
	}{
		{name: "lc_one_block", profile: 1, channels: 2},
		{name: "non_lc", profile: 0, channels: 2, wantError: "unsupported ADTS profile"},
		{name: "multiple_blocks", profile: 1, blocks: 1, channels: 2, wantError: "raw data blocks"},
		{name: "unspecified_channels", profile: 1, channels: 0, wantError: "unsupported ADTS channel configuration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adts := adtsFrame()
			adts[2] = (adts[2] & 0x3E) | tc.profile<<6 | (tc.channels >> 2)
			adts[3] = (adts[3] & 0x3F) | (tc.channels&0x03)<<6
			adts[6] = (adts[6] & 0xFC) | tc.blocks
			segment := validTSSegment()
			copy(segment[4*188:], tsPacket(audioPID, true, 3, pesBytes(0xC0, 90000, 0, false, adts)))
			resources := map[string][]byte{
				"https://media.example.test/p.m3u8": []byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1,\ns.ts\n#EXT-X-ENDLIST\n"),
				"https://media.example.test/s.ts":   segment,
			}
			tsTarget := filepath.Join(t.TempDir(), "preview.ts")
			if _, err := downloadTS(context.Background(), hlsFetch(resources), "https://media.example.test/p.m3u8", tsTarget); err != nil {
				t.Fatalf("TS preservation: %v", err)
			}
			saved, err := os.ReadFile(tsTarget)
			if err != nil || !bytes.Equal(saved, segment) {
				t.Fatalf("TS bytes changed: %v", err)
			}
			dir := t.TempDir()
			_, err = downloadMP4(context.Background(), hlsFetch(resources), "https://media.example.test/p.m3u8", filepath.Join(dir, "preview.mp4"))
			if tc.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("error = %v, want %s", err, tc.wantError)
				}
				entries, readErr := os.ReadDir(dir)
				if readErr != nil || len(entries) != 0 {
					t.Fatalf("rejected MP4 left files: %v (%v)", entries, readErr)
				}
			}
		})
	}
}
