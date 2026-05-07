package yandex

type SearchQuery struct {
	SearchType string `json:"searchType"`
	QueryText  string `json:"queryText"`
	FamilyMode string `json:"familyMode"`
	Page       int    `json:"page"`
}
type SearchDTO struct {
	Query          SearchQuery `json:"query"`
	FolderId       string `json:"folderId"`
	ResponseFormat string `json:"responseFormat"`
}

type SearchResponse struct {
	RawData string `json:"rawData"`
}

type SearchYandexDTO struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
}
