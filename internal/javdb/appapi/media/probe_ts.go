package media

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
)

var errProbeDimensionsFound = errors.New("probe dimensions found")

type probeCBCReader struct {
	reader  io.Reader
	mode    cipher.BlockMode
	pending []byte
	block   []byte
}

func newProbeCBCReader(reader io.Reader, key, iv []byte) (io.Reader, error) {
	if len(key) != aes.BlockSize {
		return nil, fmt.Errorf("HLS AES-128 key has %d bytes, want %d", len(key), aes.BlockSize)
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("HLS AES-128 IV has %d bytes, want %d", len(iv), aes.BlockSize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &probeCBCReader{
		reader: reader,
		mode:   cipher.NewCBCDecrypter(block, iv),
		block:  make([]byte, aes.BlockSize),
	}, nil
}

func (r *probeCBCReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		n, err := io.ReadFull(r.reader, r.block)
		if errors.Is(err, io.EOF) && n == 0 {
			return 0, io.EOF
		}
		if err != nil {
			return 0, fmt.Errorf("encrypted HLS segment has invalid block alignment: %w", err)
		}
		r.mode.CryptBlocks(r.block, r.block)
		r.pending = r.block
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

type probePESStripper struct {
	header []byte
	ready  bool
	active bool
}

func (s *probePESStripper) payload(pusi bool, payload []byte) ([]byte, error) {
	if pusi {
		s.header = s.header[:0]
		s.ready = false
		s.active = true
	}
	if !s.active {
		return nil, nil
	}
	if s.ready {
		return payload, nil
	}
	s.header = append(s.header, payload...)
	if len(s.header) < 9 {
		return nil, nil
	}
	if !isPESPayload(s.header) {
		return nil, fmt.Errorf("H.264 payload does not start with a PES prefix")
	}
	headerLength := 9 + int(s.header[8])
	if len(s.header) < headerLength {
		return nil, nil
	}
	es := s.header[headerLength:]
	s.header = nil
	s.ready = true
	return es, nil
}

type annexBSPSProbe struct {
	zeroCount     int
	waitingHeader bool
	collecting    bool
	sps           []byte
	lastErr       error
	width         int
	height        int
}

func (p *annexBSPSProbe) feed(data []byte) bool {
	for _, value := range data {
		if value == 0 {
			p.zeroCount++
			continue
		}
		if value == 1 && p.zeroCount >= 2 {
			if p.collecting && p.finishSPS() {
				return true
			}
			p.waitingHeader = true
			p.collecting = false
			p.sps = p.sps[:0]
			p.zeroCount = 0
			continue
		}
		if p.waitingHeader {
			p.waitingHeader = false
			p.collecting = value&0x1f == 7
			if p.collecting {
				p.sps = append(p.sps, value)
			}
			p.zeroCount = 0
			continue
		}
		if p.collecting {
			for i := 0; i < p.zeroCount; i++ {
				p.sps = append(p.sps, 0)
			}
			p.sps = append(p.sps, value)
		}
		p.zeroCount = 0
	}
	return false
}

func (p *annexBSPSProbe) finishSPS() bool {
	if len(p.sps) == 0 {
		return false
	}
	width, height, err := parseSPSDimensions(p.sps)
	if err != nil {
		p.lastErr = err
		return false
	}
	p.width = int(width)
	p.height = int(height)
	return true
}

func probeTSDimensions(reader io.Reader) (int, int, error) {
	pmtPIDs := map[uint16]bool{}
	var (
		seenPAT  bool
		seenPMT  bool
		h264PID  uint16
		hasH264  bool
		stripper probePESStripper
		scanner  annexBSPSProbe
	)
	packetCount, err := forEachTSPacket(reader, func(_ int, packet []byte) error {
		pid, pusi, payload, ok := tsPacketPayload(packet)
		if !ok {
			return nil
		}
		if pusi {
			switch {
			case pid == 0:
				pids, err := parsePSIMap(payload, 0x00)
				if err != nil {
					return fmt.Errorf("PAT not parseable: %w", err)
				}
				seenPAT = true
				for pmtPID := range pids {
					pmtPIDs[pmtPID] = true
				}
				return nil
			case pmtPIDs[pid]:
				types, err := parsePMTTypes(payload)
				if err != nil {
					return fmt.Errorf("PMT not parseable: %w", err)
				}
				seenPMT = true
				for streamPID, streamType := range types {
					if streamType == streamTypeH264 {
						h264PID = streamPID
						hasH264 = true
						break
					}
				}
				return nil
			}
		}
		if !hasH264 || pid != h264PID {
			return nil
		}
		es, err := stripper.payload(pusi, payload)
		if err != nil {
			return err
		}
		if scanner.feed(es) {
			return errProbeDimensionsFound
		}
		return nil
	})
	if errors.Is(err, errProbeDimensionsFound) {
		return scanner.width, scanner.height, nil
	}
	if err != nil {
		return 0, 0, err
	}
	if scanner.collecting && scanner.finishSPS() {
		return scanner.width, scanner.height, nil
	}
	if packetCount == 0 {
		return 0, 0, fmt.Errorf("empty TS segment")
	}
	if !seenPAT {
		return 0, 0, fmt.Errorf("TS segment has no parseable PAT")
	}
	if !seenPMT {
		return 0, 0, fmt.Errorf("TS segment has no parseable PMT")
	}
	if !hasH264 {
		return 0, 0, fmt.Errorf("TS segment PMT has no H.264 stream")
	}
	if scanner.lastErr != nil {
		return 0, 0, fmt.Errorf("parse H.264 SPS: %w", scanner.lastErr)
	}
	return 0, 0, fmt.Errorf("H.264 stream has no SPS")
}
