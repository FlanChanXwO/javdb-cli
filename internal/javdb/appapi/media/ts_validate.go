package media

import (
	"fmt"
)

// Layer A 的 segment 结构校验(input.md 计划 #29-#32):
// 只证明 segment 是结构可解析的 MPEG-TS(188 对齐、0x47 同步、PAT/PMT 可解析、
// 声明的 elementary stream 携带 PES 数据、无明显截断),不做任何重新多路复用。
// 连续性计数器刻意不校验:真实 CDN 流在 segment 边界与 EXT-X-DISCONTINUITY 处
// 合法重置,过严的 cc 检查会误报损坏(计划 #31)。

const tsPacketSize = 188

// validateTSSegment 校验解密后的 segment 是结构合法的 MPEG-TS。
// 两遍扫描:先收集 PSI(PAT/PMT),再校验各 elementary stream 的 PES 载荷,
// 避免 PMT 声明晚于首包造成误报。
func validateTSSegment(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty TS segment")
	}
	if len(data)%tsPacketSize != 0 {
		return fmt.Errorf("TS segment size %d is not %d-byte aligned", len(data), tsPacketSize)
	}
	packets := make([][]byte, 0, len(data)/tsPacketSize)
	for off := 0; off < len(data); off += tsPacketSize {
		packet := data[off : off+tsPacketSize]
		if packet[0] != 0x47 {
			return fmt.Errorf("TS packet at offset %d has invalid sync byte 0x%02X", off, packet[0])
		}
		packets = append(packets, packet)
	}

	pmtPIDs := map[uint16]bool{}
	streamPIDs := map[uint16]bool{}
	for _, packet := range packets {
		pid, pusi, payload, ok := tsPacketPayload(packet)
		if !ok || !pusi {
			continue
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
			pids, err := parsePSIMap(payload, 0x02)
			if err != nil {
				return fmt.Errorf("PMT not parseable: %w", err)
			}
			for streamPID := range pids {
				streamPIDs[streamPID] = true
			}
		}
	}
	if len(pmtPIDs) == 0 {
		return fmt.Errorf("TS segment has no parseable PAT")
	}
	if len(streamPIDs) == 0 {
		return fmt.Errorf("TS segment PMT declares no elementary streams")
	}

	seen := map[uint16]bool{}
	for _, packet := range packets {
		pid, pusi, payload, ok := tsPacketPayload(packet)
		if !ok || !streamPIDs[pid] {
			continue
		}
		if pusi && !isPESPayload(payload) {
			return fmt.Errorf("PES payload for PID 0x%04X does not start with a PES prefix", pid)
		}
		seen[pid] = true
	}
	for pid := range streamPIDs {
		if !seen[pid] {
			return fmt.Errorf("TS segment has no payload for stream PID 0x%04X", pid)
		}
	}
	return nil
}

// tsPacketPayload 提取 packet 的有效负载;adaptation-only/空负载返回 false。
func tsPacketPayload(packet []byte) (pid uint16, pusi bool, payload []byte, ok bool) {
	pid = (uint16(packet[1]&0x1F) << 8) | uint16(packet[2])
	pusi = packet[1]&0x40 != 0
	switch (packet[3] >> 4) & 0x3 {
	case 0, 2:
		return pid, pusi, nil, false
	case 3:
		if len(packet) < 5 {
			return pid, pusi, nil, false
		}
		length := int(packet[4])
		start := 5 + length
		if start >= tsPacketSize {
			return pid, pusi, nil, false
		}
		return pid, pusi, packet[start:], true
	default:
		return pid, pusi, packet[4:], true
	}
}

// parsePSIMap 解析单包 PSI section 的 table(PAT→PMT PID / PMT→stream PID)。
// PSI section 跨 packet 或多 section 的流不属于预览视频范畴,解析失败即拒绝。
func parsePSIMap(payload []byte, tableID byte) (map[uint16]bool, error) {
	if len(payload) < 4 {
		return nil, fmt.Errorf("payload too short")
	}
	pointer := int(payload[0])
	if 1+pointer+3 > len(payload) {
		return nil, fmt.Errorf("section header truncated")
	}
	section := payload[1+pointer:]
	if section[0] != tableID {
		return nil, fmt.Errorf("unexpected table_id 0x%02X", section[0])
	}
	length := (int(section[1]&0x0F) << 8) | int(section[2])
	end := 3 + length - 4 // 去掉 CRC32
	if end < 9 || end > len(section) {
		return nil, fmt.Errorf("section length %d out of bounds", length)
	}
	table := map[uint16]bool{}
	// body 前缀:公共 5 字节(TSID/版本/序号);PMT 再加 4 字节(PCR_PID+program_info_length)。
	start := 3 + 5
	if tableID == 0x02 {
		start = 3 + 9
	}
	for pos := start; pos < end; {
		switch tableID {
		case 0x00: // PAT:program_number(2)+PID(2);program_number 0 是 network PID,跳过
			if pos+4 > end {
				return nil, fmt.Errorf("truncated PAT entry")
			}
			if uint16(section[pos])<<8|uint16(section[pos+1]) != 0 {
				table[(uint16(section[pos+2]&0x1F)<<8)|uint16(section[pos+3])] = true
			}
			pos += 4
		case 0x02: // PMT:stream_type(1)+elementary_PID(2)+ES_info_length(2)
			if pos+5 > end {
				return nil, fmt.Errorf("truncated PMT entry")
			}
			table[(uint16(section[pos+1]&0x1F)<<8)|uint16(section[pos+2])] = true
			esLen := (int(section[pos+3]&0x0F) << 8) | int(section[pos+4])
			pos += 5 + esLen
		}
	}
	return table, nil
}

// isPESPayload 判断负载是否以 PES 起始码 00 00 01 开头。
func isPESPayload(payload []byte) bool {
	return len(payload) >= 3 && payload[0] == 0x00 && payload[1] == 0x00 && payload[2] == 0x01
}
