// 本文件定义用户模型，包含密码的 bcrypt 加密与校验逻辑。
package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"` // json:"-" 防止序列化时泄露密码
	CreatedAt time.Time `json:"created_at"`
}

// SetPassword 使用 bcrypt 加密密码并存储。
func (u *User) SetPassword(raw string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword 校验密码是否匹配。
func (u *User) CheckPassword(raw string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(raw))
	return err == nil
}

// BeforeCreate GORM 钩子 —— 创建用户前确保密码已加密（兜底保护）。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 如果密码已经是 bcrypt hash（以 $2a$ 开头），则不重复加密
	if len(u.Password) > 0 && len(u.Password) < 60 {
		return u.SetPassword(u.Password)
	}
	return nil
}
