// +build sproto,sproto_u32

package upstream

import (
	"bytes"
	"encoding/binary"
)

func writePacketLen(buf *bytes.Buffer, n int) {
	binary.Write(buf, binary.BigEndian, uint32(n))
}
