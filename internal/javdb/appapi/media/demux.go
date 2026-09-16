package media

import (
	"fmt"
	"io"
)

// PMT stream_type 常量(仅本工具关心的子集)。
const (
	streamTypeH264 = 0x1B // ITU-T H.264 / ISO 14496-10 AVC
	streamTypeAAC  = 0x0F // AAC ADTS
	streamTypeID3  = 0x15 // HLS timed ID3 metadata
)

// demuxedFrame 是一个完整 PES(帧级 access unit)与其时间戳(90kHz)。
type demuxedFrame struct {
	ES     []byte
	PTS    uint64
	DTS    uint64
	HasDTS bool
}

// walkTSFrames 先读取 PSI 元数据，再逐包重组每个 PID 当前未完成的 PES。
// consume 返回后不保留 frame；调用方必须在回调内处理并释放 payload。
func walkTSFrames(reader io.ReadSeeker, consume func(uint16, byte, demuxedFrame) error) (map[uint16]byte, error) {
	pmtPIDs := map[uint16]bool{}
	streamTypes := map[uint16]byte{}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind TS segment: %w", err)
	}
	_, err := forEachTSPacket(reader, func(_ int, packet []byte) error {
		pid, pusi, payload, ok := tsPacketPayload(packet)
		if !ok || !pusi {
			return nil
		}
		switch {
		case pid == 0:
			pids, err := parsePSIMap(payload, 0x00)
			if err != nil {
				return fmt.Errorf("PAT not parseable: %w", err)
			}
			for pmtPID := range pids {
				pmtPIDs[pmtPID] = true
			}
		case pmtPIDs[pid]:
			types, err := parsePMTTypes(payload)
			if err != nil {
				return fmt.Errorf("PMT not parseable: %w", err)
			}
			for streamPID, streamType := range types {
				streamTypes[streamPID] = streamType
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	pending := map[uint16][]byte{}
	flush := func(pid uint16) error {
		raw, ok := pending[pid]
		if !ok || len(raw) == 0 {
			delete(pending, pid)
			return nil
		}
		delete(pending, pid)
		streamType, ok := streamTypes[pid]
		if !ok {
			return nil // 未在 PMT 声明(如填充流):忽略
		}
		frame, err := parsePESFrame(raw)
		if err != nil {
			return err
		}
		return consume(pid, streamType, frame)
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind TS segment: %w", err)
	}
	_, err = forEachTSPacket(reader, func(_ int, packet []byte) error {
		pid, pusi, payload, ok := tsPacketPayload(packet)
		if !ok {
			return nil
		}
		if _, known := streamTypes[pid]; !known {
			return nil
		}
		if pusi {
			if err := flush(pid); err != nil {
				return err
			}
			pending[pid] = append([]byte(nil), payload...)
			return nil
		}
		pending[pid] = append(pending[pid], payload...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for pid := range streamTypes {
		if err := flush(pid); err != nil {
			return nil, err
		}
	}
	return streamTypes, nil
}

// parsePMTTypes 从单包 PMT section 提取 PID→stream_type。
func parsePMTTypes(payload []byte) (map[uint16]byte, error) {
	if len(payload) < 4 {
		return nil, fmt.Errorf("payload too short")
	}
	pointer := int(payload[0])
	if 1+pointer+3 > len(payload) {
		return nil, fmt.Errorf("section header truncated")
	}
	section := payload[1+pointer:]
	if section[0] != 0x02 {
		return nil, fmt.Errorf("unexpected table_id 0x%02X", section[0])
	}
	length := (int(section[1]&0x0F) << 8) | int(section[2])
	end := 3 + length - 4 // 去掉 CRC32
	if end < 9 || end > len(section) {
		return nil, fmt.Errorf("section length %d out of bounds", length)
	}
	types := map[uint16]byte{}
	// body 前 9 字节:program_number(2)+version(1)+序号(2)+PCR_PID(2)+info_length(2)。
	// 访问 section[10]/[11] 前必须证明 section 足够长。
	if len(section) < 12 {
		return nil, fmt.Errorf("PMT header truncated")
	}
	infoLen := (int(section[10]&0x0F) << 8) | int(section[11])
	start := 3 + 9 + infoLen
	if start > end {
		return nil, fmt.Errorf("PMT program_info_length %d out of bounds", infoLen)
	}
	for pos := start; pos+5 <= end; {
		entryPID := (uint16(section[pos+1]&0x1F) << 8) | uint16(section[pos+2])
		types[entryPID] = section[pos]
		esLen := (int(section[pos+3]&0x0F) << 8) | int(section[pos+4])
		// ES_info_length 跳出 section 末尾是 malformed TS，必须显式拒绝。
		if pos+5+esLen > end {
			return nil, fmt.Errorf("PMT ES_info_length %d out of bounds", esLen)
		}
		pos += 5 + esLen
	}
	return types, nil
}

// parsePESFrame 解析 PES header,返回帧与其时间戳。
// TS packet 的 0xFF stuffing 不属于 PES:按 PES_packet_length 截断(0=未定长,取整段)。
func parsePESFrame(pes []byte) (demuxedFrame, error) {
	if len(pes) < 9 || !isPESPayload(pes) {
		return demuxedFrame{}, fmt.Errorf("malformed PES header")
	}
	if pesLen := int(pes[4])<<8 | int(pes[5]); pesLen > 0 && 6+pesLen <= len(pes) {
		pes = pes[:6+pesLen]
	}
	flags := pes[7]
	headerLen := int(pes[8])
	if len(pes) < 9+headerLen {
		return demuxedFrame{}, fmt.Errorf("PES header truncated")
	}
	// PES_packet_length 声称的包体长度大于实际重组长度时,说明流被截断:
	// 继续解析会把 0xFF stuffing 或下一包内容当成 ES，必须显式拒绝。
	if pesLen := int(pes[4])<<8 | int(pes[5]); pesLen > 0 && 6+pesLen > len(pes) {
		return demuxedFrame{}, fmt.Errorf("PES packet truncated: header claims %d bytes, got %d", pesLen, len(pes)-6)
	}
	frame := demuxedFrame{ES: pes[9+headerLen:]}
	pos := 9
	if flags&0x80 != 0 {
		if pos+5 > len(pes) {
			return demuxedFrame{}, fmt.Errorf("PES PTS truncated")
		}
		frame.PTS = parsePESMarker(pes[pos:])
		pos += 5
	}
	// 只有显式包含 DTS(flags=11xx)时才设置 HasDTS;PTS-only 时
	// DTS 复用 PTS 便于下游计算，但 HasDTS 保持 false。
	if flags&0xC0 == 0xC0 {
		if pos+5 > len(pes) {
			return demuxedFrame{}, fmt.Errorf("PES DTS truncated")
		}
		frame.DTS = parsePESMarker(pes[pos:])
		frame.HasDTS = true
	}
	if !frame.HasDTS {
		frame.DTS = frame.PTS
	}
	return frame, nil
}

func parsePESMarker(b []byte) uint64 {
	return uint64(b[0]>>1&0x07)<<30 |
		uint64(b[1])<<22 |
		uint64(b[2]>>1)<<15 |
		uint64(b[3])<<7 |
		uint64(b[4]>>1)
}
