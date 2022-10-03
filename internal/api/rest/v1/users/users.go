package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	userRoles "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
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
	Id       int     `json:"Id" binding:"required"`
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

	if !app.IsSameUser(c, user.Id) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to update user. User ID from body: %v", user.Id))
		return
	}

	if user.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user state. User ID from body: %v", user.Id))
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

	// TODO: add confirmation flow for chaning emails
	if user.Email != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user email. User ID from body: %v", user.Id))
			return
		}
	}

	if user.Role != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user role. User ID from body: %v", user.Id))
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

	err = services.Instance().Feed().UpdateUser(toFeedUserDTO(&user))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update user")
		log.Error("Unable to update user at feed service", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func toFeedUserDTO(user *UserEditDTO) *feed.FeedUserDTO {
	result := &feed.FeedUserDTO{
		Id: int32(user.Id),
	}

	if user.Login != nil {
		result.Login = *user.Login
	}
	if user.Email != nil {
		result.Email = *user.Email
	}
	if user.Role != nil {
		result.Role = *user.Role
	}
	if user.State != nil {
		result.State = *user.State
	}

	return result
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
	case *UpdateUserPasswordDTO:
		return &queries.UpdateUserParams{
			Id:       t.UserId,
			Password: t.Password,
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
