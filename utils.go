package ddns

import (
	"hash/crc32"
)

// BackendCrc32Tab Python use 0xEDB88320
var BackendCrc32Tab = crc32.MakeTable(0xEDB88320)
// ChannelCrc32Tab different to BackendCrc32Tab
var ChannelCrc32Tab = crc32.MakeTable(0xD5828281)

func backendHash(s string) int {
	return int(crc32.Checksum([]byte(s), BackendCrc32Tab))
}

func channelHash(s string) int {
	return int(crc32.Checksum([]byte(s), ChannelCrc32Tab))
}
