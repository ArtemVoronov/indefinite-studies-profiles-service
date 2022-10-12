package users

import (
	"database/sql"
	"fmt"
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
	userRoles "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SignUpStartDTO struct {
	Login    string `json:"Login" binding:"required"`
	Email    string `json:"Email" binding:"required,email"`
	Password string `json:"Password" binding:"required"`
}

type SignUpFinishDTO struct {
	Token string `json:"Token" binding:"required"`
}

type ResendConfirmationLinkDTO struct {
	Email string `json:"Email" binding:"required"`
}

func SignUpStart(c *gin.Context) {
	var dto SignUpStartDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	hashPassword, err := credentials.HashPassword(dto.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to create sign up", err.Error())
		return
	}
	dto.Password = hashPassword

	uuid, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to create uuid for user", err.Error())
		return
	}

	userUuid := uuid.String()

	role := userRoles.USER_ROLE_GI
	if services.Instance().Whitelist().Contains(dto.Email) {
		role = userRoles.USER_ROLE_OWNER
	}

	userId, err := services.Instance().Profiles().CreateUser(userUuid, dto.Login, dto.Email, dto.Password, role, entities.USER_STATE_NEW)

	if err != nil {
		if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
			log.Error("unable to sign up", "duplicate user email: "+dto.Email)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to sign up")
			log.Error("unable to sign up", err.Error())
		}
		return
	}

	token, err := tokens.CreateToken(userUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to create token for sign up", err.Error())
		return
	}

	err = services.Instance().Profiles().CreateRegistrationToken(userUuid, userId, token)
	if err != nil {
		if err.Error() == queries.ErrorRegistrationTokenDuplicateKey.Error() {
			c.JSON(http.StatusInternalServerError, "Unable to sign up")
			log.Error("unable to sign up", "duplicate token: "+token)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to sign up")
			log.Error("unable to sign up", err.Error())
		}
		return
	}

	err = sendEmailWithSignUpConfirmationLink(dto.Email, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to send email at sign up", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}

func sendEmailWithSignUpConfirmationLink(email string, token string) error {
	sendEmailEvent := services.Instance().Templates().GetEmailSignUpConfirmationLink(email, token)
	return services.Instance().Subscriptions().PutSendEmailEvent(sendEmailEvent)
}

func SignUpFinish(c *gin.Context) {
	var dto SignUpFinishDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	userUuid, err := tokens.ExtractUserUuid(dto.Token)

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to restore finish sign up")
		log.Error("Unable to extract user uuid from registration token", err.Error())
	}

	registrationToken, err := services.Instance().Profiles().GetRegistrationToken(userUuid, dto.Token)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to finish sign up")
			log.Error("Unable to get registration token", err.Error())
		}
		return
	}

	if registrationToken.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, "Link has expired")
		log.Error("Unable to get registration token", "token has expired: "+dto.Token)
		return
	}

	confirmedState := entities.USER_STATE_CONFRIMED
	err = services.Instance().Profiles().UpdateUser(userUuid, nil, nil, nil, nil, &confirmedState)

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to finish sign up")
		log.Error("Unable to finish sign up", err.Error())
		return
	}

	log.Info(fmt.Sprintf("confirmed user Uuid %v", userUuid))

	c.JSON(http.StatusOK, api.DONE)
}

func ResendConfirmationLink(c *gin.Context) {
	var dto ResendConfirmationLinkDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	user, err := services.Instance().Profiles().GetUserByEmail(dto.Email)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
			log.Error("Unable to resend confirmation link", err.Error())
		}
		return
	}

	if user.State == entities.USER_STATE_CONFRIMED {
		c.JSON(http.StatusBadRequest, "User is confirmed already")
		log.Error("Unable to resend confirmation link", "User is confirmed already")
		return
	}

	if user.State == entities.USER_STATE_BLOCKED || user.State == entities.USER_STATE_DELETED {
		c.JSON(http.StatusBadRequest, "User is blocked")
		log.Error("Unable to resend confirmation link", "User is blocked")
		return
	}

	token, err := tokens.CreateToken(user.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("unable to create token for resend confirmation link", err.Error())
		return
	}

	err = services.Instance().Profiles().UpdsertRegistrationToken(user.Uuid, user.Id, token)

	if err != nil {
		if err.Error() == queries.ErrorRegistrationTokenDuplicateKey.Error() {
			c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
			log.Error("unable to resend confirmation link", "duplicate token: "+token)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
			log.Error("unable to resend confirmation link", err.Error())
		}
		return
	}

	err = sendEmailWithSignUpConfirmationLink(user.Email, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("unable to send email at resend confirmation link", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}
