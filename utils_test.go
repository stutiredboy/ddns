package ddns

import (
	"testing"
)

func Test_backendHas(t *testing.T) {
	hash_exist := backendHash("www.163.com")
	//hash := backendHash("Hello234")
	/* must compatible with Python zlib.crc32('www.163.com') or binascii.crc32('www.163.com') */
	if (hash_exist != 832174588) {
		t.Error("%s", hash_exist)
		t.Error("test func *backendHash* error")
	}
	hash_nil := backendHash("")
	if (hash_nil != 0) {
		t.Error("%s", hash_nil)
		t.Error("test func *backendHash* error")
	}
}

func Test_channelHash(t *testing.T){
	hash_exist := channelHash("www.163.com")
	if (hash_exist != 684573356) {
		t.Error("%s", hash_exist)
		t.Error("test func *channelHash* error")
	}
	hash_nil := channelHash("")
	if (hash_nil != 0) {
		t.Error("%s", hash_nil)
		t.Error("test func *channelHash* error")
	}
}
