package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
)

type trackedProbeBody struct {
	reader io.Reader
	read   int
	closed bool
}

func (b *trackedProbeBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += n
	return n, err
}

func (b *trackedProbeBody) Close() error {
	b.closed = true
	return nil
}

func encodedProbeImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 0x80, G: 0x40, B: 0x20, A: 0xff})
	var out bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&out, img, nil)
	case "png":
		err = png.Encode(&out, img)
	case "gif":
		err = gif.Encode(&out, img, nil)
	default:
		t.Fatalf("unsupported test image format %q", format)
	}
	if err != nil {
		t.Fatalf("encode %s fixture: %v", format, err)
	}
	return out.Bytes()
}

func webPProbeFixture(kind string, width, height int) []byte {
	var payload []byte
	switch kind {
	case "VP8 ":
		payload = []byte{0, 0, 0, 0x9d, 0x01, 0x2a, byte(width), byte(width >> 8), byte(height), byte(height >> 8)}
	case "VP8L":
		packed := uint32(width-1) | uint32(height-1)<<14
		payload = []byte{0x2f, byte(packed), byte(packed >> 8), byte(packed >> 16), byte(packed >> 24)}
	case "VP8X":
		payload = []byte{0, 0, 0, 0, byte(width - 1), byte((width - 1) >> 8), byte((width - 1) >> 16), byte(height - 1), byte((height - 1) >> 8), byte((height - 1) >> 16)}
	}
	chunkSize := len(payload)
	fixture := make([]byte, 20+chunkSize)
	copy(fixture[:4], "RIFF")
	binary.LittleEndian.PutUint32(fixture[4:8], uint32(len(fixture)-8))
	copy(fixture[8:12], "WEBP")
	copy(fixture[12:16], kind)
	binary.LittleEndian.PutUint32(fixture[16:20], uint32(chunkSize))
	copy(fixture[20:], payload)
	return fixture
}

func TestProbeImageMetadataStreamsSupportedFormats(t *testing.T) {
	const width, height = 37, 23
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "jpeg", payload: encodedProbeImage(t, "jpeg", width, height)},
		{name: "png", payload: encodedProbeImage(t, "png", width, height)},
		{name: "gif", payload: encodedProbeImage(t, "gif", width, height)},
		{name: "webp vp8", payload: webPProbeFixture("VP8 ", width, height)},
		{name: "webp vp8l", payload: webPProbeFixture("VP8L", width, height)},
		{name: "webp vp8x", payload: webPProbeFixture("VP8X", width, height)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := &trackedProbeBody{reader: bytes.NewReader(tc.payload)}
			endpoint := NewMedia(func(context.Context, string) (io.ReadCloser, error) {
				return body, nil
			})
			metadata, err := endpoint.ProbeImageMetadata(context.Background(), "https://media.example.test/image")
			if err != nil {
				t.Fatalf("ProbeImageMetadata() error = %v", err)
			}
			if metadata.Width != width || metadata.Height != height {
				t.Fatalf("metadata = %+v, want %dx%d", metadata, width, height)
			}
			if !body.closed {
				t.Fatal("ProbeImageMetadata() did not close response body")
			}
		})
	}
}

func TestProbeImageMetadataUnwrapsXORAndStopsAfterHeader(t *testing.T) {
	const width, height = 41, 29
	payload := encodedProbeImage(t, "png", width, height)
	const key = byte(0x97)
	wrapped := make([]byte, len(payload)+1)
	wrapped[0] = key
	for i, value := range payload {
		wrapped[i+1] = value ^ key
	}
	wrapped = append(wrapped, bytes.Repeat([]byte{0xaa}, 1<<20)...)
	body := &trackedProbeBody{reader: bytes.NewReader(wrapped)}
	endpoint := NewMedia(func(context.Context, string) (io.ReadCloser, error) {
		return body, nil
	})
	metadata, err := endpoint.ProbeImageMetadata(context.Background(), "https://media.example.test/image")
	if err != nil {
		t.Fatalf("ProbeImageMetadata() error = %v", err)
	}
	if metadata.Width != width || metadata.Height != height {
		t.Fatalf("metadata = %+v, want %dx%d", metadata, width, height)
	}
	if body.read >= len(wrapped) {
		t.Fatalf("ProbeImageMetadata() read full response: %d bytes", body.read)
	}
	if !body.closed {
		t.Fatal("ProbeImageMetadata() did not close response body")
	}
}

func TestProbeHLSPlaylistDuration(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	playlist, err := parseHLSProbePlaylistReader(playlistURL, strings.NewReader("#EXTM3U\n#EXTINF:10.125,\na.ts\n#EXTINF:9.875,\nb.ts\n#EXT-X-ENDLIST\n"))
	if err != nil {
		t.Fatalf("parseHLSProbePlaylistReader() error = %v", err)
	}
	if !playlist.durationValid || playlist.durationSeconds != 20 {
		t.Fatalf("duration = %v valid=%v, want 20 seconds", playlist.durationSeconds, playlist.durationValid)
	}
	if !playlist.hasFirstSegment || playlist.firstSegment.uri != "https://media.example.test/previews/a.ts" {
		t.Fatalf("first segment = %+v, want resolved a.ts", playlist.firstSegment)
	}
}

func TestProbeHLSPlaylistDoesNotFabricateDuration(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	for _, tc := range []struct {
		name     string
		playlist string
		wantErr  bool
	}{
		{name: "unfinished", playlist: "#EXTM3U\n#EXTINF:1.0,\na.ts\n", wantErr: true},
		{name: "malformed extinf", playlist: "#EXTM3U\n#EXTINF:not-a-number,\na.ts\n#EXT-X-ENDLIST\n"},
		{name: "missing extinf", playlist: "#EXTM3U\na.ts\n#EXT-X-ENDLIST\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			playlist, err := parseHLSProbePlaylistReader(playlistURL, strings.NewReader(tc.playlist))
			if tc.wantErr {
				if err == nil {
					t.Fatal("parseHLSProbePlaylistReader() unexpectedly succeeded")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHLSProbePlaylistReader() error = %v", err)
			}
			if playlist.durationValid || playlist.durationSeconds != 0 {
				t.Fatalf("duration = %v valid=%v, want unknown", playlist.durationSeconds, playlist.durationValid)
			}
		})
	}
}

func TestProbeHLSMetadataExtractsDurationAndSPS(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	const segmentURL = "https://media.example.test/previews/a.ts"
	const keyURL = "https://media.example.test/previews/key.bin"
	key := []byte("0123456789abcdef")
	segment := validTSSegmentAt(0)
	for _, tc := range []struct {
		name     string
		playlist []byte
		segment  []byte
		key      []byte
	}{
		{
			name:     "plain",
			playlist: []byte("#EXTM3U\n#EXTINF:10.125,\na.ts\n#EXTINF:9.875,\nb.ts\n#EXT-X-ENDLIST\n"),
			segment:  segment,
		},
		{
			name:     "aes-128",
			playlist: []byte("#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:7\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:10.125,\na.ts\n#EXTINF:9.875,\nb.ts\n#EXT-X-ENDLIST\n"),
			segment:  encryptHLSTestPayload(t, segment, key, hlsSequenceIV(7)),
			key:      key,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resources := map[string][]byte{
				playlistURL: tc.playlist,
				segmentURL:  tc.segment,
				keyURL:      tc.key,
			}
			endpoint := NewMedia(byteFetch(func(_ context.Context, uri string) ([]byte, error) {
				payload, ok := resources[uri]
				if !ok {
					return nil, fmt.Errorf("unexpected media URI %q", uri)
				}
				return payload, nil
			}))
			metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
			if err != nil {
				t.Fatalf("ProbeHLSMetadata() error = %v", err)
			}
			if metadata.Width != 640 || metadata.Height != 480 || metadata.DurationSeconds != 20 {
				t.Fatalf("metadata = %+v, want 640x480 and 20 seconds", metadata)
			}
		})
	}
}

func TestProbeHLSMetadataPreservesDurationWhenSPSProbeFails(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	endpoint := NewMedia(byteFetch(func(_ context.Context, uri string) ([]byte, error) {
		if uri == playlistURL {
			return []byte("#EXTM3U\n#EXTINF:20.0,\nbroken.ts\n#EXT-X-ENDLIST\n"), nil
		}
		return nil, errors.New("segment unavailable")
	}))
	metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
	if err != nil {
		t.Fatalf("ProbeHLSMetadata() error = %v", err)
	}
	if metadata.DurationSeconds != 20 || metadata.Width != 0 || metadata.Height != 0 {
		t.Fatalf("metadata = %+v, want duration-only result", metadata)
	}
}

func TestProbeHLSMetadataStopsAfterSPS(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	const segmentURL = "https://media.example.test/previews/segment.ts"
	segment := append(validTSSegmentAt(0), bytes.Repeat([]byte{0xaa}, 1<<20)...)
	segmentBody := &trackedProbeBody{reader: bytes.NewReader(segment)}
	endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
		switch uri {
		case playlistURL:
			return io.NopCloser(strings.NewReader("#EXTM3U\n#EXTINF:1.0,\nsegment.ts\n#EXT-X-ENDLIST\n")), nil
		case segmentURL:
			return segmentBody, nil
		default:
			return nil, fmt.Errorf("unexpected media URI %q", uri)
		}
	})
	metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
	if err != nil {
		t.Fatalf("ProbeHLSMetadata() error = %v", err)
	}
	if metadata.Width != 640 || metadata.Height != 480 {
		t.Fatalf("metadata = %+v, want 640x480", metadata)
	}
	if segmentBody.read >= len(segment) {
		t.Fatalf("ProbeHLSMetadata() read full segment: %d bytes", segmentBody.read)
	}
	if !segmentBody.closed {
		t.Fatal("ProbeHLSMetadata() did not close segment body")
	}
}

func TestProbeHLSMetadataReturnsContextCancellation(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	t.Run("before playlist", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		endpoint := NewMedia(func(context.Context, string) (io.ReadCloser, error) {
			t.Fatal("fetch called after context cancellation")
			return nil, nil
		})
		_, err := endpoint.ProbeHLSMetadata(ctx, playlistURL)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ProbeHLSMetadata() error = %v, want context.Canceled", err)
		}
	})
	t.Run("during segment", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
			if uri == playlistURL {
				return io.NopCloser(strings.NewReader("#EXTM3U\n#EXTINF:20.0,\nsegment.ts\n#EXT-X-ENDLIST\n")), nil
			}
			cancel()
			return nil, context.Canceled
		})
		_, err := endpoint.ProbeHLSMetadata(ctx, playlistURL)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ProbeHLSMetadata() error = %v, want context.Canceled", err)
		}
	})
	t.Run("transport deadline", func(t *testing.T) {
		endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
			if uri == playlistURL {
				return io.NopCloser(strings.NewReader("#EXTM3U\n#EXTINF:20.0,\nsegment.ts\n#EXT-X-ENDLIST\n")), nil
			}
			return nil, context.DeadlineExceeded
		})
		_, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ProbeHLSMetadata() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

type cancelOnReadBody struct {
	reader io.Reader
	cancel context.CancelFunc
	read   bool
	bytes  int
}

func (b *cancelOnReadBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.bytes += n
	if !b.read {
		b.read = true
		b.cancel()
	}
	return n, err
}

func (b *cancelOnReadBody) Close() error { return nil }

func TestProbeHLSMetadataStopsWhenContextCancelsDuringReader(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	const segmentURL = "https://media.example.test/previews/segment.ts"
	ctx, cancel := context.WithCancel(context.Background())
	playlistPayload := append([]byte("#EXTM3U\n#EXTINF:20.0\nsegment.ts\n#EXT-X-ENDLIST\n"), bytes.Repeat([]byte("#COMMENT\n"), 1<<17)...)
	var playlistBody *cancelOnReadBody
	endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
		switch uri {
		case playlistURL:
			playlistBody = &cancelOnReadBody{reader: bytes.NewReader(playlistPayload), cancel: cancel}
			return playlistBody, nil
		case segmentURL:
			return nil, context.Canceled
		default:
			return nil, fmt.Errorf("unexpected media URI %q", uri)
		}
	})
	_, err := endpoint.ProbeHLSMetadata(ctx, playlistURL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ProbeHLSMetadata() error = %v, want context.Canceled", err)
	}
	if playlistBody == nil || playlistBody.bytes >= len(playlistPayload) {
		t.Fatalf("playlist reader consumed %d/%d bytes after cancellation", playlistBody.bytes, len(playlistPayload))
	}
}

func TestAnnexBSPSProbeStopsCollectingMalformedUnterminatedSPS(t *testing.T) {
	var scanner annexBSPSProbe
	if scanner.feed([]byte{0, 0, 0, 1, 0x67}) {
		t.Fatal("malformed SPS unexpectedly produced dimensions")
	}
	scanner.feed(bytes.Repeat([]byte{0}, 1<<20))
	if scanner.feed([]byte{2}) {
		t.Fatal("malformed SPS unexpectedly produced dimensions")
	}
	if scanner.collecting {
		t.Fatal("malformed SPS remained in collecting state")
	}
	if len(scanner.sps) > 64 {
		t.Fatalf("malformed SPS buffer grew to %d bytes, want bounded parser state", len(scanner.sps))
	}
	if scanner.lastErr == nil {
		t.Fatal("malformed SPS did not expose a parse error")
	}
}
