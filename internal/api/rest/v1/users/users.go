package users

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserDTO struct {
	Id    int
	Login string
	Email string
	Role  string
	State string
}

type CredentialsDTO struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CredentialsValidationResult struct {
	UserId  int  `json:"userId" binding:"required,email"`
	IsValid bool `json:"isValid" binding:"required"`
}

type UserListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []UserDTO
}

type UserEditDTO struct {
	Login    string `json:"login" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
	State    string `json:"state" binding:"required"`
}

type UserCreateDTO struct {
	Login    string `json:"login" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
	State    string `json:"state" binding:"required"`
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

	data, err := db.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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

	data, err := db.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
		validation.ProcessAndSendValidationErrorMessage(c, err)
		return
	}

	possibleUserRoles := entities.GetPossibleUserRoles()
	if !utils.Contains(possibleUserRoles, user.Role) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
		return
	}

	possibleUserStates := entities.GetPossibleUserStates()
	if !utils.Contains(possibleUserStates, user.State) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'State' value. Possible values: %v", possibleUserStates))
		return
	}

	if user.State == entities.USER_STATE_DELETED {
		c.JSON(http.StatusBadRequest, api.DELETE_VIA_POST_REQUEST_IS_FODBIDDEN)
		return
	}

	hashPassword, err := hashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create user")
		log.Printf("Unable to create user : %s", err)
		return
	}

	data, err := db.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		result, err := queries.CreateUser(tx, ctx, user.Login, user.Email, string(hashPassword), user.Role, user.State)
		return result, err
	})()

	if err != nil || data == -1 {
		if err.Error() == db.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to create user")
			log.Printf("Unable to create user : %s", err)
		}
		return
	}

	c.JSON(http.StatusCreated, data)
}

// TODO: add optional field updating (field is not reqired and missed -> do not update it)
func UpdateUser(c *gin.Context) {
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

	var user UserEditDTO

	if err := c.ShouldBindJSON(&user); err != nil {
		validation.ProcessAndSendValidationErrorMessage(c, err)
		return
	}

	if user.State == entities.USER_STATE_DELETED {
		c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
		return
	}

	possibleUserRoles := entities.GetPossibleUserRoles()
	if !utils.Contains(possibleUserRoles, user.Role) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
		return
	}

	possibleUserStates := entities.GetPossibleUserStates()
	if !utils.Contains(possibleUserStates, user.State) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update user. Wrong 'State' value. Possible values: %v", possibleUserStates))
		return
	}

	hashPassword, err := hashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update user")
		log.Printf("Unable to update user : %s", err)
		return
	}

	// TODO: check password hash
	// TODO: add route for changing password
	// TODO: add route for restoring password
	err = db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateUser(tx, ctx, userId, user.Login, user.Email, string(hashPassword), user.Role, user.State)
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else if err.Error() == db.ErrorUserDuplicateKey.Error() {
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
	idStr := c.Param("id")

	if idStr == "" {
		c.JSON(http.StatusBadRequest, "Missed ID")
		return
	}

	var id int
	var parseErr error
	if id, parseErr = strconv.Atoi(idStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	err := db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteUser(tx, ctx, id)
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

func IsValidCredentials(c *gin.Context) {
	var credentials CredentialsDTO

	if err := c.ShouldBindJSON(&credentials); err != nil {
		validation.ProcessAndSendValidationErrorMessage(c, err)
		return
	}

	// TODO: add counter of invalid athorizations, then use it for temporary blocking access
	validatoionResult, err := checkCredentials(credentials.Email, credentials.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Internal server error")
		log.Printf("error during authenication: %v\n", err)
		return
	}

	c.JSON(http.StatusOK, validatoionResult)
}

func UpdateCredentials(c *gin.Context) {
	// TODO
	c.JSON(http.StatusNotImplemented, "Not Implemented")
}

func checkCredentials(email string, password string) (*CredentialsValidationResult, error) {
	var result *CredentialsValidationResult

	data, err := db.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetUserByEmail(tx, ctx, email)
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			return &CredentialsValidationResult{UserId: -1, IsValid: false}, nil
		}
		return result, fmt.Errorf("unable to check credentials : %s", err)
	}

	user, ok := data.(entities.User)
	if !ok {
		return result, fmt.Errorf("unable to check credentials : %s", api.ERROR_ASSERT_RESULT_TYPE)
	}

	if isValidPassword(user.Password, password) {
		result = &CredentialsValidationResult{UserId: user.Id, IsValid: true}
	} else {
		result = &CredentialsValidationResult{UserId: -1, IsValid: false}
	}

	return result, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("unable to create password hash: %v", err.Error())
	}
	return string(hash), nil
}

func isValidPassword(hashedPassword string, rawPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(rawPassword))
	return err == nil
}
