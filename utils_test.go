package ddns

import (
	"testing"
)

func Test_backendHas(t *testing.T) {
	hashExist := backendHash("www.163.com")
	//hash := backendHash("Hello234")
	/* must compatible with Python zlib.crc32('www.163.com') or binascii.crc32('www.163.com') */
	if hashExist != 832174588 {
		t.Errorf("%d", hashExist)
		t.Error("test func *backendHash* error")
	}
	hashNil := backendHash("")
	if hashNil != 0 {
		t.Errorf("%d", hashNil)
		t.Error("test func *backendHash* error")
	}
}

func Test_channelHash(t *testing.T) {
	hashExist := channelHash("www.163.com")
	if hashExist != 684573356 {
		t.Errorf("%d", hashExist)
		t.Error("test func *channelHash* error")
	}
	hashNil := channelHash("")
	if hashNil != 0 {
		t.Errorf("%d", hashNil)
		t.Error("test func *channelHash* error")
	}
}
