package users

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/tokens"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/gin-gonic/gin"
)

type RestorePasswordStartDTO struct {
	Email string `json:"Email" binding:"required,email"`
}

type RestorePasswordFinishdDTO struct {
	Token    string `json:"Token" binding:"required"`
	Password string `json:"Password" binding:"required"`
}

func RestorePasswordStart(c *gin.Context) {
	var dto RestorePasswordStartDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	user, err := services.Instance().Profiles().GetUserByEmail(dto.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("Unable to restore password", err.Error())
		}
		return
	}

	if user.State == entities.USER_STATE_BLOCKED || user.State == entities.USER_STATE_DELETED {
		c.JSON(http.StatusBadRequest, "User is blocked")
		log.Error("Unable to restore password", "User is blocked")
		return
	}

	token, err := tokens.CreateToken(user.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to restore password")
		log.Error("unable to create token for restore password", err.Error())
		return
	}

	err = services.Instance().Profiles().UpdsertRestorePasswordToken(user.Uuid, user.Id, token)

	if err != nil {
		if err.Error() == queries.ErrorRegistrationTokenDuplicateKey.Error() {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("unable to restore password", "duplicate token: "+token)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("unable to restore password", err.Error())
		}
		return
	}

	err = sendEmailWithRestorePasswordConfirmationLink(user.Email, token)
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

	userUuid, err := tokens.ExtractUserUuid(dto.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to restore password")
		log.Error("Unable to extract user uuid from restore password token", err.Error())
	}

	restorePasswordToken, err := services.Instance().Profiles().GetRestorePasswordToken(userUuid, dto.Token)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to restore password")
			log.Error("Unable to get restore password token", err.Error())
		}
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

	err = services.Instance().Profiles().UpdateUser(userUuid, nil, nil, &hashPassword, nil, nil)
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
