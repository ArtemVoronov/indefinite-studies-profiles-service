package users

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RestorePasswordStartDTO struct {
	Email string `json:"Email" binding:"required,email"`
}

type RestorePasswordFinishdDTO struct {
	Token    string `json:"Token" binding:"required"`
	Password string `json:"Password" binding:"required"`
}

type UpdateUserPasswordDTO struct {
	UserId   int    `json:"UserId" binding:"required"`
	Password string `json:"Password" binding:"required"`
}

func RestorePasswordStart(c *gin.Context) {
	var dto RestorePasswordStartDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetUserByEmail(tx, ctx, dto.Email)
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("Unable to restore password", err.Error())
		}
		return
	}

	user, ok := data.(entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to restore password")
		log.Error("Unable to restore password", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	if user.State == entities.USER_STATE_BLOCKED || user.State == entities.USER_STATE_DELETED {
		c.JSON(http.StatusBadRequest, "User is blocked")
		log.Error("Unable to restore password", "User is blocked")
		return
	}

	token, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to restore password")
		log.Error("unable to create token for restore password", err.Error())
		return
	}

	err = services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateRestorePasswordToken(tx, ctx, user.Id, token.String())
		if err == sql.ErrNoRows {
			err = queries.CreateRestorePasswordToken(tx, ctx, user.Id, token.String())
		}
		return err
	})()

	if err != nil {
		if err.Error() == queries.ErrorRegistrationTokenDuplicateKey.Error() {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("unable to restore password", "duplicate token: "+token.String())
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("unable to restore password", err.Error())
		}
		return
	}

	err = sendEmailWithRestorePasswordConfirmationLink(user.Email, token.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("unable to send email at resend confirmation link", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}

func RestorePasswordFinish(c *gin.Context) {
	var dto RestorePasswordFinishdDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetRestorePasswordToken(tx, ctx, dto.Token)
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("Unable to get restore password token", err.Error())
		}
		return
	}

	restorePasswordToken, ok := data.(entities.RestorePasswordToken)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to restore password")
		log.Error("Unable to get restore password token", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	if restorePasswordToken.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, "Link has expired")
		log.Error("Unable to get restore password", "token has expired: "+dto.Token)
		return
	}

	hashPassword, err := credentials.HashPassword(dto.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to restore password")
		log.Error("Unable to get restore password token", err.Error())
		return
	}

	user := UpdateUserPasswordDTO{UserId: restorePasswordToken.UserId, Password: hashPassword}

	err = services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		params, err := toUpdateUserParams(&user)
		if err != nil {
			return err
		}
		err = queries.UpdateUser(tx, ctx, params)
		return err
	})()

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to finish sign up")
		log.Error("Unable to finish sign up", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func sendEmailWithRestorePasswordConfirmationLink(email string, token string) error {
	sendEmailEvent := services.Instance().Templates().GetEmailRestorePasswordLink(email, token)
	return services.Instance().Subscriptions().PutSendEmailEvent(sendEmailEvent)
}
