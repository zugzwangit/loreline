package platform

import "time"

type Knowledge struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Source     string    `json:"source"`
	Owner      string    `json:"owner"`
	Status     string    `json:"status"`
	Confidence int       `json:"confidence"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Decision struct {
	Action  string `json:"action"`
	Content string `json:"content,omitempty"`
}
type AskRequest struct {
	Question string `json:"question"`
}
type Answer struct {
	Answer     string     `json:"answer"`
	Confidence float64    `json:"confidence"`
	Citations  []Citation `json:"citations"`
}
type Citation struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Source string `json:"source"`
}
