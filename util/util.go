package util

import (
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"
)

const TAG = "util"

func BytesToUtf8(iso8859_1_buf []byte) []byte {
	buf := make([]rune, len(iso8859_1_buf))
	for i, b := range iso8859_1_buf {
		buf[i] = rune(b)
	}
	return []byte(string(buf))
}

func DumpRequest(req *http.Request, name string) {
	dump1, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.WithField("tag", TAG).Errorf("dump of %v request failed", name)
	}
	ioutil.WriteFile(name, dump1, 0777)
}

func DumpResponse(resp *http.Response, name string) {
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.WithField("tag", TAG).Errorf("log failed for get Request")
	}
	ioutil.WriteFile(name, dump, 0777)
}
