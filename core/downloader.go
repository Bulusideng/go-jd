package core

import (
	"compress/gzip"
	"fmt"

	"io"
	"io/ioutil"

	"net/http"
)

type Downloader struct {
	*http.Client
}

func (dl *Downloader) GetResponse(method, URL string, queryFun func(URL string) string) ([]byte, error) {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	queryURL := URL
	if queryFun != nil {
		queryURL = queryFun(URL)
	}

	if req, err = http.NewRequest(method, queryURL, nil); err != nil {
		return nil, err
	}
	applyCustomHeader(req, DefaultHeaders)
	cnt := 0

	for cnt < 3 {
		if resp, err = dl.Do(req); err != nil {
			fmt.Printf("Error[%d]%s get %s\n", cnt, err.Error(), req.URL)
			cnt++
			if cnt > 3 {
				return nil, err
			}
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	var reader io.Reader

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(resp.Body)
	default:
		reader = resp.Body
	}

	return ioutil.ReadAll(reader)
}
