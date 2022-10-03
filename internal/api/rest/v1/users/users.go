package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	userRoles "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/kafka"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserDTO struct {
	Id    int
	Login string
	Email string
	Role  string
	State string
}

type UserListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []UserDTO
}

type UserEditDTO struct {
	Id       *int    `json:"Id" binding:"required"`
	Login    *string `json:"Login,omitempty"`
	Email    *string `json:"Email,omitempty"`
	Password *string `json:"Password,omitempty"`
	Role     *string `json:"Role,omitempty"`
	State    *string `json:"State,omitempty"`
}

type UserCreateDTO struct {
	Login    string `json:"Login" binding:"required"`
	Email    string `json:"Email" binding:"required,email"`
	Password string `json:"Password" binding:"required"`
	Role     string `json:"Role" binding:"required"`
	State    string `json:"State" binding:"required"`
}

type UserDeleteDTO struct {
	Id int `json:"Id" binding:"required"`
}

type SendEmailDTO struct {
	Sender    string `json:"Sender" binding:"required"`
	Recepient string `json:"Recepient" binding:"required"`
	Body      string `json:"Body" binding:"required"`
}

func GetUsers(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		users, err := queries.GetUsers(tx, ctx, limit, offset)
		return users, err
	})()

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Error("Unable to get users", err.Error())
		return
	}

	users, ok := data.([]entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Error("Unable to get users", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	result := &UserListDTO{Data: convertUsers(users), Count: len(users), Offset: offset, Limit: limit}
	c.JSON(http.StatusOK, result)
}

func GetMyProfile(c *gin.Context) {
	userId, ok := c.Get(app.CTX_TOKEN_ID_KEY)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get to user profile")
		log.Error("Unable to get user profile", "Missed TOKEN ID in gin context")
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetUser(tx, ctx, userId.(int))
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get to user profile")
			log.Error("Unable to get user profile", err.Error())
		}
		return
	}

	user, ok := data.(entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Error("Unable to get user profile", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	c.JSON(http.StatusOK, convertUser(user))
}

func GetUser(c *gin.Context) {
	userIdStr := c.Param("id")

	if userIdStr == "" {
		c.JSON(http.StatusBadRequest, "Missed ID")
		return
	}

	var userId int
	var parseErr error
	if userId, parseErr = strconv.Atoi(userIdStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetUser(tx, ctx, userId)
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get user")
			log.Error("Unable to get user", err.Error())
		}
		return
	}

	user, ok := data.(entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Error("Unable to get user", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	c.JSON(http.StatusOK, convertUser(user))
}

func CreateUser(c *gin.Context) {
	var user UserCreateDTO

	if err := c.ShouldBindJSON(&user); err != nil {
		validation.SendError(c, err)
		return
	}

	possibleUserRoles := entities.GetPossibleUserRoles()
	if !utils.Contains(possibleUserRoles, user.Role) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
		return
	}

	possibleUseStates := entities.GetPossibleUserStates()
	if !utils.Contains(possibleUseStates, user.State) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'State' value. Possible values: %v", possibleUseStates))
		return
	}

	hashPassword, err := credentials.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create user")
		log.Error("Unable to create user", err.Error())
		return
	}
	user.Password = hashPassword

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		params, err := toCreateUserParams(&user)
		if err != nil {
			return nil, err
		}
		result, err := queries.CreateUser(tx, ctx, params)
		return result, err
	})()

	if err != nil || data == -1 {
		if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
			log.Error("unable to create user", "duplicate user email: "+user.Email)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to create user")
			log.Error("Unable to create user", err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, data)
}

func UpdateUser(c *gin.Context) {
	var user UserEditDTO

	if err := c.ShouldBindJSON(&user); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, *user.Id) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to update user. User ID from body: %v", *user.Id))
		return
	}

	if user.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user state. User ID from body: %v", *user.Id))
			return
		}
		if *user.State == entities.USER_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleUserStates := entities.GetPossibleUserStates()
		if !utils.Contains(possibleUserStates, *user.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update user. Wrong 'State' value. Possible values: %v", possibleUserStates))
			return
		}
	}

	if user.Role != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user role. User ID from body: %v", *user.Id))
			return
		}
		possibleUserRoles := entities.GetPossibleUserRoles()
		if !utils.Contains(possibleUserRoles, *user.Role) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
			return
		}
	}

	if user.Password != nil {
		hashPassword, err := credentials.HashPassword(*user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Unable to update user")
			log.Error("Unable to update user", err.Error())
			return
		}
		user.Password = &hashPassword
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		params, err := toUpdateUserParams(&user)
		if err != nil {
			return err
		}
		err = queries.UpdateUser(tx, ctx, params)
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
			log.Error("unable to update user", "duplicate user email: "+*user.Email)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update user")
			log.Error("Unable to update user", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func DeleteUser(c *gin.Context) {
	var user UserDeleteDTO
	if err := c.ShouldBindJSON(&user); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, user.Id) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to delete user. User ID from body: %v", user.Id))
		return
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteUser(tx, ctx, user.Id)
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete user")
			log.Error("Unable to delete user", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func convertUsers(users []entities.User) []UserDTO {
	if users == nil {
		return make([]UserDTO, 0)
	}
	var result []UserDTO
	for _, user := range users {
		result = append(result, convertUser(user))
	}
	return result
}

func convertUser(user entities.User) UserDTO {
	return UserDTO{Id: user.Id, Login: user.Login, Email: user.Email, Role: user.Role, State: user.State}
}

func toUpdateUserParams(dto any) (*queries.UpdateUserParams, error) {
	switch t := dto.(type) {
	case *UserEditDTO:
		return &queries.UpdateUserParams{
			Id:       t.Id,
			Login:    t.Login,
			Email:    t.Email,
			Password: t.Password,
			Role:     t.Role,
			State:    t.State,
		}, nil
	case *entities.RegistrationToken:
		return &queries.UpdateUserParams{
			Id:    t.UserId,
			State: entities.USER_STATE_CONFRIMED,
		}, nil
	default:
		return nil, fmt.Errorf("unkown type: %T", t)
	}
}

func toCreateUserParams(dto any) (*queries.CreateUserParams, error) {
	switch t := dto.(type) {
	case *UserCreateDTO:
		return &queries.CreateUserParams{
			Login:    t.Login,
			Email:    t.Email,
			Password: t.Password,
			Role:     t.Role,
			State:    t.State,
		}, nil
	case *SignUpStartDTO:
		role := userRoles.USER_ROLE_GI
		if services.Instance().Whitelist().Contains(t.Email) {
			role = userRoles.USER_ROLE_OWNER
		}
		return &queries.CreateUserParams{
			Login:    t.Login,
			Email:    t.Email,
			Password: t.Password,
			Role:     role,
			State:    entities.USER_STATE_NEW,
		}, nil
	default:
		return nil, fmt.Errorf("unkown type: %T", t)
	}
}

type SignUpStartDTO struct {
	Login    string `json:"Login" binding:"required"`
	Email    string `json:"Email" binding:"required,email"`
	Password string `json:"Password" binding:"required"`
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

	err = sendEmailWithConfirmationLink(dto.Email, token.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to sign up")
		log.Error("unable to send email at sign up", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}

func sendEmailWithConfirmationLink(email string, token string) error {
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
	token := c.Param("token")

	if token == "" {
		c.JSON(http.StatusBadRequest, "Missed 'token' param")
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetRegistrationToken(tx, ctx, token)
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
		log.Error("Unable to get registration token", "token has expired: "+token)
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

	err = sendEmailWithConfirmationLink(user.Email, token.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to resend confirmation link")
		log.Error("unable to send email at resend confirmation link", err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}
