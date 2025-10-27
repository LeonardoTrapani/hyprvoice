package transcriber

import (
	"bytes"
	"encoding/binary"
)

// convertToWAV converts raw 16-bit PCM audio to WAV format
func convertToWAV(rawAudio []byte) ([]byte, error) {
	var buf bytes.Buffer

	const sampleRate = 16000
	const channels = 1
	const bitsPerSample = 16
	const byteRate = sampleRate * channels * bitsPerSample / 8
	const blockAlign = channels * bitsPerSample / 8

	dataSize := len(rawAudio)
	fileSize := 36 + dataSize

	// WAV header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(fileSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))            // fmt chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // PCM format
	binary.Write(&buf, binary.LittleEndian, uint16(channels))      // number of channels
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))    // sample rate
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))      // byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))    // block align
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample)) // bits per sample

	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(rawAudio)

	return buf.Bytes(), nil
}
