package domain

// SQLPlan represents a safe SQL query plan
type SQLPlan struct {
	Operation string   `json:"operation"` // SELECT only
	Table     string   `json:"table"`     // users | kpr_applications | approval_workflows
	Columns   []string `json:"columns"`   // optional, default "*"
	Filters   []Filter `json:"filters"`   // equality filters only
	Limit     int      `json:"limit"`     // optional
}

// Filter represents a SQL filter condition
type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"` // '=' only
	Value  string `json:"value"`
}

// SendMessageRequest represents request to send message
type SendMessageRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// SendMessageResponse represents response after sending message
type SendMessageResponse struct {
	Status string `json:"status"`
	Phone  string `json:"phone"`
}
