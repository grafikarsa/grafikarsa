package process

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const glbHeaderBytes = 12

// validateGLB checks the binary glTF header: magic "glTF", version 2, and a
// declared length that matches the actual file size.
func validateGLB(data []byte) error {
	if len(data) < glbHeaderBytes {
		return errors.New("file is smaller than the 12-byte GLB header")
	}
	if string(data[0:4]) != "glTF" {
		return errors.New("invalid GLB magic")
	}
	if version := binary.LittleEndian.Uint32(data[4:8]); version != 2 {
		return fmt.Errorf("unsupported GLB version %d", version)
	}
	declared := binary.LittleEndian.Uint32(data[8:12])
	if int64(declared) != int64(len(data)) {
		return fmt.Errorf("declared length %d does not match file size %d", declared, len(data))
	}
	return nil
}
