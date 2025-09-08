package repository

import (
	"todo_backend/internal/domain"
)

// UserRepositoryはユーザエンティティの永続化をを抽象化したインターフェースです。
// DBの種類や実装に依存せず、ユースケース側から利用されます。
type UserRepository interface {
	// Createは新しいユーザーを永続化します。
	// すでに同じEmailが存在する場合はエラーを返します。
	Create(user *domain.User) error

	// FindByEmailは指定したEmailに一致するユーザーを取得します。
	// ユーザーが存在しない場合はエラーを返します。
	FindByEmail(email string) (*domain.User, error)

	// FindByIDは指定したIDに一致するユーザーを取得します。
	// ユーザーが存在しない場合はエラーを返します。
	FindByID(id uint) (*domain.User, error)
}
