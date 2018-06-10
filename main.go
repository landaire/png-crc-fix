package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

const (
	chunkStartOffset = 8
	endChunk         = "IEND"
	usage            = "Usage: png-crc-fix FILE"
	magic            = "\x89PNG\x0d\x0a\x1a\x0a"
)

type pngChunk struct {
	Offset int64
	Length uint32
	Type   [4]byte
	Data   []byte
	CRC    uint32
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, usage)

		return
	}

	filePath := os.Args[1]

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer file.Close()

	if !isPng(file) {
		fmt.Fprintln(os.Stderr, "Not a PNG")
		os.Exit(1)
	}

	// Read all the chunks. They start with IHDR at offset 8
	chunks := readChunks(file)

	for _, chunk := range chunks {
		fmt.Println(chunk)
		if !chunk.CRCIsValid() {
			file.Seek(chunk.CRCOffset(), os.SEEK_SET)
			binary.Write(file, binary.BigEndian, chunk.CalculateCRC())
			fmt.Println("Corrected CRC")
		}
	}
}

func (p pngChunk) String() string {
	return fmt.Sprintf("%s@%x - %X - Valid CRC? %v",
		p.Type,
		p.Offset,
		p.CRC,
		p.CRCIsValid())
}

// BytesForCRC returns the bytes that contribute to the chunk's CRC32
func (p pngChunk) BytesForCRC() []byte {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.BigEndian, p.Type)
	buffer.Write(p.Data)

	return buffer.Bytes()
}

// CRCIsValid returns a boolean value indicating if thethe CRC32 for the given
// chunk is valid
func (p pngChunk) CRCIsValid() bool {
	return p.CRC == p.CalculateCRC()
}

// CalculateCRC calculates the CRC of the chunk
func (p pngChunk) CalculateCRC() uint32 {
	crcTable := crc32.MakeTable(crc32.IEEE)

	return crc32.Checksum(p.BytesForCRC(), crcTable)
}

// Returns the reader offset of the CRC32 for this chunk
func (p pngChunk) CRCOffset() int64 {
	return p.Offset + int64(8+p.Length)
}

// readChunks reads the chunks from the reader. If an error occurs then reading
// stops and the chunks read up to that point are returned
func readChunks(reader io.ReadSeeker) []pngChunk {
	chunks := []pngChunk{}

	reader.Seek(chunkStartOffset, os.SEEK_SET)

	readChunk := func() (*pngChunk, error) {
		var chunk pngChunk
		chunk.Offset, _ = reader.Seek(0, os.SEEK_CUR)

		err := binary.Read(reader, binary.BigEndian, &chunk.Length)
		if err != nil {
			goto read_error
		}

		chunk.Data = make([]byte, chunk.Length)

		err = binary.Read(reader, binary.BigEndian, &chunk.Type)
		if err != nil {
			goto read_error
		}

		if _, err = io.ReadFull(reader, chunk.Data); err != nil {
			goto read_error
		}

		err = binary.Read(reader, binary.BigEndian, &chunk.CRC)
		if err != nil {
			goto read_error
		}

		return &chunk, nil

	read_error:
		return nil, fmt.Errorf("Read error")
	}

	chunk, err := readChunk()
	if err != nil {
		return chunks
	}

	chunks = append(chunks, *chunk)

	// Read the first chunk
	for string(chunks[len(chunks)-1].Type[:]) != endChunk {

		chunk, err := readChunk()
		if err != nil {
			break
		}

		chunks = append(chunks, *chunk)
	}

	return chunks
}

// Checks if the file is a valid PNG
func isPng(s io.Reader) bool {
	h := make([]byte, 8)
	_, err := io.ReadFull(s, h)
	if err != nil {
		return false
	}

	return string(h) == magic
}
