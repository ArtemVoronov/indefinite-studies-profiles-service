package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/kafka"
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

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		params, err := toCreateUserParams(&dto)
		if err != nil {
			return nil, err
		}
		result, err := queries.CreateUser(tx, ctx, params)
		return result, err
	})()

	if err != nil || data == -1 {
		if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
			log.Error("unable to sign up", "duplicate user email: "+dto.Email)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to sign up")
			log.Error("unable to sign up", err.Error())
		}
		return
	}

	userId, ok := data.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to create sign up", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	token, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to create token for sign up", err.Error())
		return
	}

	err = services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.CreateRegistrationToken(tx, ctx, userId, token.String())
		return err
	})()

	if err != nil {
		if err.Error() == queries.ErrorRegistrationTokenDuplicateKey.Error() {
			c.JSON(http.StatusInternalServerError, "Unable to sign up")
			log.Error("unable to sign up", "duplicate token: "+token.String())
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to sign up")
			log.Error("unable to sign up", err.Error())
		}
		return
	}

	err = sendEmailWithSignUpConfirmationLink(dto.Email, token.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to send email at sign up", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}

func sendEmailWithSignUpConfirmationLink(email string, token string) error {
	// TODO: use config for link and email generation (appropriate domain name, https, path for link and sender, subject, body for email)
	link := "http://localhost/api/v1/users/signup/" + token

	sendEmailEvent := kafka.SendEmailEvent{
		Sender:    "no-reply@indefinitestudies.ru",
		Recepient: email,
		Subject:   "Registration at indefinitestudies.ru",
		Body:      "Welcome!\n\nUse the following link for finishing registration: " + link + "\n\nBest Regards,\nIndefinite Studies Team",
	}

	// TODO: clean
	log.Info(fmt.Sprintf("sending email %v", sendEmailEvent))

	return services.Instance().Subscriptions().PutSendEmailEvent(sendEmailEvent)
}

func SignUpFinish(c *gin.Context) {
	var dto SignUpFinishDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetRegistrationToken(tx, ctx, dto.Token)
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to finish sign up")
			log.Error("Unable to get registration token", err.Error())
		}
		return
	}

	registrationToken, ok := data.(entities.RegistrationToken)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to finish sign up")
		log.Error("Unable to get registration token", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	if registrationToken.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, "Link has expired")
		log.Error("Unable to get registration token", "token has expired: "+dto.Token)
		return
	}

	err = services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		params, err := toUpdateUserParams(&registrationToken)
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

	log.Info(fmt.Sprintf("confirmed user ID %v", registrationToken.UserId))

	c.JSON(http.StatusOK, api.DONE)
}

func ResendConfirmationLink(c *gin.Context) {
	var dto ResendConfirmationLinkDTO
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
			c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
			log.Error("Unable to resend confirmation link", err.Error())
		}
		return
	}

	user, ok := data.(entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("Unable to resend confirmation link", api.ERROR_ASSERT_RESULT_TYPE)
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

	token, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("unable to create token for resend confirmation link", err.Error())
		return
	}

	err = services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateRegistrationToken(tx, ctx, user.Id, token.String())
		if err == sql.ErrNoRows {
			err = queries.CreateRegistrationToken(tx, ctx, user.Id, token.String())
		}
		return err
	})()

	if err != nil {
		if err.Error() == queries.ErrorRegistrationTokenDuplicateKey.Error() {
			c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
			log.Error("unable to resend confirmation link", "duplicate token: "+token.String())
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
			log.Error("unable to resend confirmation link", err.Error())
		}
		return
	}

	err = sendEmailWithSignUpConfirmationLink(user.Email, token.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("unable to send email at resend confirmation link", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}
