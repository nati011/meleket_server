package model

import "time"

type NotificationMessage struct {
	ID            uint `gorm:"AUTO_INCREMENT;primary_key;index"`
	ClientID      uint
	ApplicationID uint
	Message       string `gorm:"type:text"`
	Title         string `gorm:"type:text"`
	Priority      int
	Extras        []byte
	Date          time.Time
	Seen          bool
}

type NotificationMessageExternal struct {
	ID            uint                   `json:"id"`
	ClientID      uint                   `form:"clientid" query:"clientid" json:"clientid"`
	ApplicationID uint                   `form:"appid" query:"appid" json:"appid"`
	Message       string                 `form:"message" query:"message" json:"message" binding:"required"`
	Title         string                 `form:"title" query:"title" json:"title"`
	Priority      *int                   `form:"priority" query:"priority" json:"priority"`
	Extras        map[string]interface{} `form:"-" query:"-" json:"extras,omitempty"`
	Date          time.Time              `json:"date"`
}
