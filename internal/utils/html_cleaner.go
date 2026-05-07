package utils

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const noiseTags = "script,style,header,footer,nav,aside,noscript,iframe,template,canvas,form,svg,link"

func CleanHtml(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	body := doc.Find("body")
	if body.Length() == 0 {
		body = doc.Selection
	}

	body.Find(noiseTags).Remove()

	body.Find("*").Each(func(_ int, sel *goquery.Selection) {
		for _, node := range sel.Nodes {
			node.Attr = nil
		}
	})

	body.Find("*").AddBack().Each(func(_ int, sel *goquery.Selection) {
		if strings.TrimSpace(sel.Text()) == "" {
			sel.Remove()
		}
	})

	cleanedBody, err := body.Html()
	if err != nil {
		return "", err
	}

	return cleanedBody, nil
}
