// Package usecase はauthフィーチャーのビジネスロジックを実装します。
package usecase

import "errors"

var (
	// ErrUserNotFound はメールアドレスまたはIDでユーザーが見つからない場合に返されます。
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists は既に存在するメールアドレスでユーザーを作成しようとした場合に返されます。
	ErrEmailAlreadyExists = errors.New("email already exists")
)
