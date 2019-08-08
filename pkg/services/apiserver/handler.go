package apiserver

import (
	"fmt"

	"github.com/emicklei/go-restful"

	"openpitrix.io/newbilling/pkg/logger"
)

func parseBool(input string) bool {
	if input == "true" {
		return true
	} else {
		return false
	}
}

func DescribeNodes(request *restful.Request, response *restful.Response) {
	watch := parseBool(request.QueryParameter("watch"))

	logger.Debug(nil, "DescribeNodes success")

	response.Write([]byte(fmt.Sprintf("test %v", watch)))
}
