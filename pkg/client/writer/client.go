package writer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"openpitrix.io/scheduler/pkg/logger"
)

var client = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   10000,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

const (
	DefaultScheme = "http"
)

func WriteAPIServer(server string, resource string, name string, content string) string {
	url := fmt.Sprintf("%s/%s/%s", server, resource, name)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(content)))
	if err != nil {
		logger.Error(nil, "WriteAPIServer NewRequest error:", err)
		return ""
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		logger.Error(nil, "WriteAPIServer get error:", err)
	} else {
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)

		if err != nil {
			logger.Error(nil, "WriteAPIServer read error:", err.Error())
		}

		return string(body)
	}
	return ""
}
