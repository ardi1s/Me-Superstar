// 本文件定义作品模型 —— 账号下发布的内容。
package models

import (
	"time"
)

// Work 表示某个平台账号下发布的一件作品。
// AccountID + WorkID 组成联合唯一索引，保证同一账号下 WorkID 不重复。
type Work struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AccountID   uint      `gorm:"uniqueIndex:idx_account_work;not null" json:"account_id"`
	WorkID      string    `gorm:"uniqueIndex:idx_account_work;size:128;not null" json:"work_id"`
	Title       string    `gorm:"size:255" json:"title"`
	CoverURL    string    `gorm:"size:512" json:"cover_url"`
	PublishTime time.Time `json:"publish_time"`
	CreatedAt   time.Time `json:"created_at"`

	Account Account `gorm:"foreignKey:AccountID" json:"-"`
}
