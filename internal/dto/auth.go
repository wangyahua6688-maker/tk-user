package dto

// SMSResult 短信发送响应。
type SMSResult struct {
	Phone       string `json:"phone"`
	Purpose     string `json:"purpose"`
	ExpiresSec  int    `json:"expires_sec"`
	MockMode    bool   `json:"mock_mode"`
	PreviewCode string `json:"preview_code,omitempty"`
}

// AuthResult 用户登录/注册成功后的统一返回结构。
type AuthResult struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
}
