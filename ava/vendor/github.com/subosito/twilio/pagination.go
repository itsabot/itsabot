package twilio

type Pagination struct {
	Page            int    `json:"page"`
	NumPages        int    `json:"num_pages"`
	PageSize        int    `json:"page_size"`
	Total           int    `json:"total"`
	Start           int    `json:"start"`
	End             int    `json:"end"`
	Uri             string `json:"uri"`
	FirstPageUri    string `json:"first_page_uri"`
	PreviousPageUri string `json:"previous_page_uri"`
	NextPageUri     string `json:"next_page_uri"`
	LastPageUri     string `json:"last_page_uri"`
}
