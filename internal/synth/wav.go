package synth

import (
	"bytes"
	"encoding/binary"
)

func Int16ToBytesLE(samples []int16) []byte {
	b := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(s))
	}
	return b
}

func PCM16ToWAV(samples []int16, sampleRate int) []byte {
	audioData := Int16ToBytesLE(samples)
	dataLen := uint32(len(audioData))
	byteRate := uint32(sampleRate * 2) // mono * 16-bit
	blockAlign := uint16(2)
	var buf bytes.Buffer

	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36)+dataLen)
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(&buf, binary.LittleEndian, byteRate)
	_ = binary.Write(&buf, binary.LittleEndian, blockAlign)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))

	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, dataLen)
	buf.Write(audioData)
	return buf.Bytes()
}
