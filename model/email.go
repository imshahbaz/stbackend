package model

import (
	"fmt"
)

type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type BrevoEmailRequest struct {
	Sender      Recipient   `json:"sender"`
	To          []Recipient `json:"to"`
	Subject     string      `json:"subject"`
	HTMLContent string      `json:"htmlContent"`
}

const SignupTemplate = `
<table style="max-width:400px;margin:auto;padding:20px;border:1px solid #ddd;border-radius:8px;">
  <tr><td style="font-family:Arial, sans-serif;">
    <p>Verification code: <h2 style="color:#1a73e8;">%s</h2></p>
    <p>Valid for %d minutes.</p>
  </td></tr>
</table>`

func (r *BrevoEmailRequest) Signup(otp string, validity int) {
	r.Subject = "Signup Verification Code"
	r.HTMLContent = fmt.Sprintf(SignupTemplate, otp, 5)
}

type SendEmailRequest struct {
	Body BrevoEmailRequest
}
