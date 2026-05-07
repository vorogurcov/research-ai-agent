package yandex

import (
	"encoding/xml"
	"strings"
)

func mapXMLToResultString(xmlBytes []byte) (string, error) {

	type RequestInfo struct {
		Query string `xml:"query"`
		Page  int    `xml:"page"`
	}
	type Group struct {
		Url      string   `xml:"doc>url"`
		Passages []string `xml:"doc>passages>passage"`
	}

	type ResponseInfo struct {
		Groups []Group `xml:"results>grouping>group"`
	}

	type Result struct {
		Request  RequestInfo  `xml:"request"`
		Response ResponseInfo `xml:"response"`
	}
	var result Result
	err := xml.Unmarshal(xmlBytes, &result)
	if err != nil {
		return "", nil
	}

	for i := range result.Response.Groups {
		for j := range result.Response.Groups[i].Passages {
			result.Response.Groups[i].Passages[j] = normalizeSpace(result.Response.Groups[i].Passages[j])
		}
	}

	r, err := xml.Marshal(result)
	if err != nil {
		return "", nil
	}
	return string(r), nil
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
