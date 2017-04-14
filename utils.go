package ddns

import (
	"hash/crc32"
)

// Python use 0xEDB88320
var Crc32Tab = crc32.MakeTable(0xEDB88320)

func name_hash(s string) int {
	return int(crc32.Checksum([]byte(s), Crc32Tab))
}
