package test

import (
	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v2/model"
)

// WithUser fake an authentication for testing.
func WithUser(ctx *gin.Context, userID uint) {
	ctx.Set("user", &model.User{ID: userID})
	ctx.Set("userid", userID)
}
