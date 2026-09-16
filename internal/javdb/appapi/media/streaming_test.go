package media

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// 在 reader 尚有后续 PES 时检查真实 spool，验证 payload 已被消费，
// 避免以依赖 GC 时机的 RSS 阈值替代生命周期契约。
type spoolProgressReader struct {
	*bytes.Reader
	file       *os.File
	progressed bool
}

func (r *spoolProgressReader) Read(p []byte) (int, error) {
	if r.Len() > 188*4 {
		info, err := r.file.Stat()
		if err != nil {
			return 0, err
		}
		r.progressed = r.progressed || info.Size() > 0
	}
	return r.Reader.Read(p)
}

func TestSegmentSpoolsPESBeforeReadingWholeSegment(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer spool.file.Close()
	var segment []byte
	for i := 0; i < 2048; i++ {
		segment = append(segment, validTSSegmentAt(uint64(i)*180000)...)
	}
	reader := &spoolProgressReader{Reader: bytes.NewReader(segment), file: spool.file}
	if err := spool.addSegmentReader(reader); err != nil {
		t.Fatal(err)
	}
	if !reader.progressed {
		t.Fatal("all segment PES were read before the first payload reached spool")
	}
	if err := spool.finalize(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := writeMP4Body(spool, &out); err != nil {
		t.Fatal(err)
	}
	if len(spool.video.Samples) != 4096 || len(spool.audio.Samples) != 2048 {
		t.Fatalf("sample counts: video=%d audio=%d", len(spool.video.Samples), len(spool.audio.Samples))
	}
	if spool.video.Samples[1].DTS != 93600 || spool.audio.Samples[1].PTS != 270000 || spool.audio.SampleRate != 44100 {
		t.Fatal("multi-PES timestamps or codec config mismatch")
	}
	path := filepath.Join(t.TempDir(), "multi.mp4")
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(path); err != nil {
		t.Fatal(err)
	}
}

// 写入器记录实际 I/O 块大小；大 sample 仍必须通过有界缓冲复制。
type copyBlockWriter struct {
	bytes.Buffer
	largest int
}

func (w *copyBlockWriter) Write(p []byte) (int, error) {
	if len(p) > w.largest {
		w.largest = len(p)
	}
	return w.Buffer.Write(p)
}
func TestSpoolCopiesLargeSampleWithBoundedBuffer(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer spool.file.Close()
	payload := bytes.Repeat([]byte{0x12, 0x34, 0x56}, 40000)
	if _, err := spool.file.Write(append([]byte("prefix"), payload...)); err != nil {
		t.Fatal(err)
	}
	out := &copyBlockWriter{}
	n, err := copySamplesFromSpool(spool, []mp4SampleMeta{{SpoolOffset: 6, Size: uint32(len(payload))}}, out)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(payload)) || !bytes.Equal(out.Bytes(), payload) {
		t.Fatal("spool payload mismatch")
	}
	if out.largest > 32*1024 {
		t.Fatalf("copy allocated sample-sized output block: %d", out.largest)
	}
}

func TestSpoolRejectsAACConfigChangeInsidePES(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer spool.file.Close()
	segment := validTSSegmentAt(0)
	changed := adtsFrame()
	changed[2] = 0x4c // 48 kHz，首帧为 44.1 kHz。
	audio := pesBytes(0xc0, 90000, 0, false, append(adtsFrame(), changed...))
	segment = append(segment[:4*188], tsPacket(audioPID, true, 3, audio)...)
	if err := spool.addSegment(segment); err == nil {
		t.Fatal("AAC configuration change inside PES was accepted")
	}
}

func TestSpoolAcceptsParameterSetsInSeparatePES(t *testing.T) {
	spool, err := newMP4Spooler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer spool.file.Close()
	original := validTSSegmentAt(0)
	sps := []byte{0, 0, 0, 1, 0x67, 0x42, 0, 0x28, 0xf8, 0x14, 7, 0xb2}
	ppsIDR := append(h264Frame(8, []byte{0xcc, 0xdd}), h264Frame(5, []byte{1, 2})...)
	segment := append([]byte(nil), original[:2*188]...)
	segment = append(segment, tsPacket(videoPID, true, 0, pesBytes(0xe0, 90000, 90000, true, sps))...)
	segment = append(segment, tsPacket(videoPID, true, 1, pesBytes(0xe0, 90000, 90000, true, ppsIDR))...)
	segment = append(segment, original[3*188:]...)
	if err := spool.addSegment(segment); err != nil {
		t.Fatal(err)
	}
	if err := spool.finalize(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := writeMP4Body(spool, &out); err != nil {
		t.Fatal(err)
	}
}

func TestLayerBRejectsDeclaredAudioTrackWithoutFrames(t *testing.T) {
	segment := validTSSegmentAt(0)
	pmt := pmtSection()
	body := append([]byte(nil), pmt[4:len(pmt)-4]...)
	body = append(body, 0x0f, 0xe1, 0x03, 0xf0, 0) // 额外声明 PID 0x103，但不提供该轨道 PES。
	copy(segment[188:376], tsPacket(pmtPID, true, 0, psiSection(2, body)))
	if err := validateMediaStream(segment); err == nil {
		t.Fatal("empty declared AAC track was masked by another audio track")
	}
}
