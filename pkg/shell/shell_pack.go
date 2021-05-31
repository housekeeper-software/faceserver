package shell

import (
	"bufio"
	"bytes"
	"encoding/binary"
)

func packet(message string) ([]byte, error) {
	var buf bytes.Buffer
	var size = uint16(len(message))
	if err := binary.Write(&buf, binary.BigEndian, size); err != nil {
		return nil, err
	}
	if size > 0 {
		if _, err := buf.Write([]byte(message)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func unPacket(r *bufio.Reader) (string, error) {
	var msg string
	var size uint16 = 0
	err := binary.Read(r, binary.BigEndian, &size)
	if err != nil {
		return msg, err
	}

	if size > 0 {
		buffer := make([]byte, size)
		_, err := r.Read(buffer)
		if err != nil {
			return msg, err
		}
		msg = string(buffer)
	}
	return msg, nil
}
