// 本文件定义平台账号模型 —— 用户绑定的各平台（如抖音）账号及其 Token。
package models

import (
	"time"
)

// Account 表示用户在某个平台上的授权账号，一个用户可绑定多个平台账号。
type Account struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index;not null" json:"user_id"`
	Platform          string    `gorm:"size:32;default:douyin" json:"platform"`
	PlatformAccountID string    `gorm:"size:128;not null" json:"platform_account_id"`
	AccessToken       string    `gorm:"size:512" json:"-"`
	RefreshToken      string    `gorm:"size:512" json:"-"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}
