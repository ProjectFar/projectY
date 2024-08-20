package model

type Blog struct {
	ID     string  `json:"id"`
	User   string  `json:"user"`
	Title  string  `json:"title"`
	Author string  `json:"author"`
	Price  float64 `json:"price"`
}
