package cio

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

func WriteWithLength(w io.Writer, b []byte) (int, error) {
	var err error

	length := uint32(len(b))
	err = binary.Write(w, binary.BigEndian, length)
	if err != nil {
		return 0, err
	}

	n, err := w.Write(b)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func ReadWithLength(r io.Reader) ([]byte, int, error) {
	var err error
	var length uint32

	err = binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, 0, err
	}

	bytes := make([]byte, length)
	n, err := io.ReadFull(r, bytes)
	if err != nil {
		return nil, 0, err
	}

	return bytes, n, nil
}

func WriteObj(w io.Writer, obj any) (int, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return 0, err
	}

	return WriteWithLength(w, bytes)
}

func ReadObj(r io.Reader, obj any) (int, error) {
	bytes, n, err := ReadWithLength(r)
	if err != nil {
		return 0, err
	}

	unmarshalErr := json.Unmarshal(bytes, obj)
	if unmarshalErr != nil {
		return 0, unmarshalErr
	}

	return n, nil
}
