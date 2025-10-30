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

// OTPRequest represents request to generate OTP
type OTPRequest struct {
	Phone      string `json:"phone"`
	ExpiryTime int    `json:"expiry_time,omitempty"` // in seconds, optional
}

// OTPResponse represents response after generating OTP
type OTPResponse struct {
	Status    string `json:"status"`
	Phone     string `json:"phone"`
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
	ExpiresIn int    `json:"expires_in"` // seconds remaining
}

// OTPValidateRequest represents request to validate OTP
type OTPValidateRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// OTPValidateResponse represents response after validating OTP
type OTPValidateResponse struct {
	Status  string `json:"status"`
	Phone   string `json:"phone"`
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}
