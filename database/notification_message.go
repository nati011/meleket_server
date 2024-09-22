package database

import (
	"github.com/jinzhu/gorm"
	"github.com/meleket/server/v2/model"
)

func (d *GormDatabase) GetNotificationMessageByID(id uint) (*model.NotificationMessage, error) {
	msg := new(model.NotificationMessage)
	err := d.DB.Find(msg, id).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if msg.ID == id {
		return msg, err
	}
	return nil, err
}

func (d *GormDatabase) CreateNotificationMessage(notification_message *model.NotificationMessage) error {
	return d.DB.Create(notification_message).Error
}

func (d *GormDatabase) GetNotificationMessageByClient(clientID uint) ([]*model.NotificationMessage, error) {
	var messages []*model.NotificationMessage
	err := d.DB.Joins("JOIN applications ON applications.client_id = ?", clientID).
		Where("messages.application_id = applications.id").Order("id desc").Find(&messages).Error

	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

func (d *GormDatabase) DeleteNotificationMessageByID(id uint) error {
	return d.DB.Where("id = ?", id).Delete(&model.Message{}).Error
}
