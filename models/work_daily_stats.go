// 本文件定义作品每日统计模型 —— 记录作品逐日的粉丝、播放、互动等指标。
package models

import "time"

// WorkDailyStats 记录作品在某一天的数据表现。
// WorkID + StatDate 组成联合唯一索引，同一天同一作品只有一条记录。
type WorkDailyStats struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	WorkID       string `gorm:"size:128;not null;uniqueIndex:idx_work_date" json:"work_id"`
	StatDate     string `gorm:"size:10;not null;uniqueIndex:idx_work_date" json:"stat_date"` // 格式: 2006-01-02
	FansAdded    int    `gorm:"default:0" json:"fans_added"`
	PlayCount    int    `gorm:"default:0" json:"play_count"`
	LikeCount    int    `gorm:"default:0" json:"like_count"`
	CommentCount int    `gorm:"default:0" json:"comment_count"`
	ShareCount   int    `gorm:"default:0" json:"share_count"`

	CreatedAt time.Time `json:"created_at"`
}
