package utils

import (
	"context"
	"errors"
	"github.com/SEC-Jobstreet/backend-candidate-service/api/models"
)

func GetCurrentUser(ctx context.Context) (models.AuthClaim, *models.AppError) {
	userCtx := ctx.Value(CurrentUser)
	currentUser, ok := userCtx.(models.AuthClaim)
	if !ok {
		return models.AuthClaim{}, &models.AppError{IsError: true, Message: "Cannot parse to type User", Error: errors.New("cannot parse to type User")}
	}

	return currentUser, nil
}