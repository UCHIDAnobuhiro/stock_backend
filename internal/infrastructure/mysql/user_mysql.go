package mysql

import (
	"todo_backend/internal/domain"
	"todo_backend/internal/interface/repository"

	"gorm.io/gorm"
)

// userMySQLはUserRepositoryインターフェースのMySQLの実装です。
// GORMを使用してデータベースの操作を行います。
type userMySQL struct {
	db *gorm.DB
}

// コンパイル時にuserMySQLがUserRepositoryを実装していることを確認する。
var _ repository.UserRepository = (*userMySQL)(nil)

// NewUserMysqlは、指定された gorm.DB接続を使用するUserMysqlの
// 新しいインスタンスを返します（DI用のコンストラクタ）
func NewUserMySQL(db *gorm.DB) *userMySQL {
	return &userMySQL{db: db}
}

// CreateはユーザをDBに追加します。
func (r *userMySQL) Create(u *domain.User) error {
	return r.db.Create(u).Error
}

// FindByEmailはEmailをキーにユーザを検索します。
// 該当するユーザが存在しない場合はエラーを返します。
func (r *userMySQL) FindByEmail(email string) (*domain.User, error) {
	var u domain.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByIDはIDをキーにユーザを検索します。
// 該当するユーザが存在しない場合、エラーを返します。
func (r *userMySQL) FindByID(id uint) (*domain.User, error) {
	var u domain.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
