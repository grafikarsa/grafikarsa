package dto

type CaptchaResponse struct {
	ID       string `json:"id"`
	Question string `json:"question"`
}

type CaptchaVerifyRequest struct {
	ID     string `json:"captcha_id" validate:"required"`
	Answer int    `json:"captcha_answer" validate:"required"`
}
