package api

type AuthResponse struct {
	AccessToken string `json:"AccessToken"`
	User        User   `json:"User"`
}

type User struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

type ItemsResponse struct {
	Items            []Item `json:"Items"`
	TotalRecordCount int    `json:"TotalRecordCount"`
}

type Item struct {
	Id                string `json:"Id"`
	Name              string `json:"Name"`
	Type              string `json:"Type"`
	SeriesName        string `json:"SeriesName"`
	IndexNumber       int    `json:"IndexNumber"`
	ParentIndexNumber int    `json:"ParentIndexNumber"`
	ProductionYear    int    `json:"ProductionYear"`
	Path              string `json:"Path"`
}
