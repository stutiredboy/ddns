package ddns

import (
	"testing"
)

func Test_backendHas(t *testing.T) {
	hash := backendHash("www.163.com")
	//hash := backendHash("Hello234")
	/* must compatible with Python zlib.crc32('www.163.com') or binascii.crc32('www.163.com') */
	if (hash != 832174588) {
		t.Error("%s", hash)
		t.Error("test func *backendHash* error")
	}
}
