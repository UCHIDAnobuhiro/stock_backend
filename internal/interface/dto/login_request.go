package dto

// loginReqは/loginのリクエストボディを表す構造体です。
// バリデーションとして必須チェックを行います。
type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
