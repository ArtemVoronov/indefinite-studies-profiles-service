package users

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
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
		log.Printf("Unable to get to users : %s", err)
		return
	}

	users, ok := data.([]entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Printf("Unable to get to users : %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	result := &UserListDTO{Data: convertUsers(users), Count: len(users), Offset: offset, Limit: limit}
	c.JSON(http.StatusOK, result)
}

func GetMyProfile(c *gin.Context) {
	userId, ok := c.Get(app.CTX_TOKEN_ID_KEY)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Internal Server Error")
		log.Printf("Unauthorized")
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
			c.JSON(http.StatusInternalServerError, "Unable to get user")
			log.Printf("Unable to get to user : %s", err)
		}
		return
	}

	user, ok := data.(entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Printf("Unable to get to users : %s", api.ERROR_ASSERT_RESULT_TYPE)
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
			log.Printf("Unable to get to user : %s", err)
		}
		return
	}

	user, ok := data.(entities.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Printf("Unable to get to users : %s", api.ERROR_ASSERT_RESULT_TYPE)
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

	isAllowed := services.Instance().Whitelist().Contains(user.Email)
	if !isAllowed {
		c.JSON(http.StatusForbidden, "Forbidden")
		return
	}

	possibleUserRoles := entities.GetPossibleUserRoles()
	if !utils.Contains(possibleUserRoles, user.Role) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
		return
	}

	hashPassword, err := credentials.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create user")
		log.Printf("Unable to create user : %s", err)
		return
	}
	user.Password = hashPassword

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		result, err := queries.CreateUser(tx, ctx, toCreateUserParams(&user))
		return result, err
	})()

	if err != nil || data == -1 {
		if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to create user")
			log.Printf("Unable to create user : %s", err)
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
		log.Printf("Forbidden to update user. User ID from body: %v", *user.Id)
		return
	}

	if user.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Printf("Forbidden to change user state. User ID from body: %v", *user.Id)
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
			log.Printf("Forbidden to change user role. User ID from body: %v", *user.Id)
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
			log.Printf("Unable to update user : %s", err)
			return
		}
		user.Password = &hashPassword
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateUser(tx, ctx, toUpdateUserParams(&user))

		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update user")
			log.Printf("Unable to update user : %s", err)
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
		log.Printf("Forbidden to delete user. User ID from body: %v", user.Id)
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
			log.Printf("Unable to delete user: %s", err)
		}
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func SendEmail(c *gin.Context) {
	var dto SendEmailDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Notifications().SendEmail(dto.Sender, dto.Recepient, dto.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to send email")
		log.Printf("Unable to send email: %s", err)
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

func toUpdateUserParams(user *UserEditDTO) *queries.UpdateUserParams {
	return &queries.UpdateUserParams{
		Id:       user.Id,
		Login:    user.Login,
		Email:    user.Email,
		Password: user.Password,
		Role:     user.Role,
		State:    user.State,
	}
}

func toCreateUserParams(user *UserCreateDTO) *queries.CreateUserParams {
	return &queries.CreateUserParams{
		Login:    user.Login,
		Email:    user.Email,
		Password: user.Password,
		Role:     user.Role,
	}
}
