package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/meleket/server/v2/auth"
	"github.com/meleket/server/v2/model"
)

type NotificationMessageDatabase interface {
	CreateNotificationMessage(notification_message *model.NotificationMessage) error
	GetApplicationByToken(token string) (*model.Application, error)
}

type NotificationMessageNotifier interface {
	NotifyClient(userID uint, clientToken string, message *model.NotificationMessageExternal)
}

type NotificationMessageAPI struct {
	DB       NotificationMessageDatabase
	ClientDB ClientDatabase
	Notifier NotificationMessageNotifier
}

func (a *NotificationMessageAPI) CreateNotificationMessage(ctx *gin.Context) {
	notificationMessage := model.NotificationMessageExternal{}
	if err := ctx.Bind(&notificationMessage); err == nil {
		application, err := a.DB.GetApplicationByToken(auth.GetTokenID(ctx))
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		notificationMessage.ApplicationID = application.ID

		client, clientErr := a.ClientDB.GetClientByID(notificationMessage.ClientID)
		if clientSuccess := successOrAbort(ctx, 500, clientErr); !clientSuccess {
			return
		}
		if client == nil {
			ctx.AbortWithError(404, fmt.Errorf("client with id %d doesn't exists", notificationMessage.ClientID))
			return
		} else {
			notificationMessage.ClientID = client.ID
		}

		if strings.TrimSpace(notificationMessage.Title) == "" {
			notificationMessage.Title = application.Name
		}

		if notificationMessage.Priority == nil {
			notificationMessage.Priority = &application.DefaultPriority
		}

		notificationMessage.Date = timeNow()
		notificationMsgInternal := toInternalNotificationMessage(&notificationMessage)
		if success := successOrAbort(ctx, 500, a.DB.CreateNotificationMessage(notificationMsgInternal)); !success {
			return
		}
		a.Notifier.NotifyClient(auth.GetUserID(ctx), client.Token, toExternalNotificationMessage(notificationMsgInternal))
		ctx.JSON(200, toExternalNotificationMessage(notificationMsgInternal))
	}
}

func toInternalNotificationMessage(msg *model.NotificationMessageExternal) *model.NotificationMessage {
	res := &model.NotificationMessage{
		ID:            msg.ID,
		ClientID:      msg.ClientID,
		ApplicationID: msg.ApplicationID,
		Message:       msg.Message,
		Title:         msg.Title,
		Date:          msg.Date,
	}
	if msg.Priority != nil {
		res.Priority = *msg.Priority
	}

	if msg.Extras != nil {
		res.Extras, _ = json.Marshal(msg.Extras)
	}
	return res
}

func toExternalNotificationMessage(msg *model.NotificationMessage) *model.NotificationMessageExternal {
	res := &model.NotificationMessageExternal{
		ID:            msg.ID,
		ClientID:      msg.ClientID,
		ApplicationID: msg.ApplicationID,
		Message:       msg.Message,
		Title:         msg.Title,
		Priority:      &msg.Priority,
		Date:          msg.Date,
	}
	if len(msg.Extras) != 0 {
		res.Extras = make(map[string]interface{})
		json.Unmarshal(msg.Extras, &res.Extras)
	}
	return res
}
