package users

import (
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
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserDTO struct {
	Uuid  string
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
	Uuid     string  `json:"Uuid" binding:"required"`
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
	Uuid string `json:"Uuid" binding:"required"`
}

type SendEmailDTO struct {
	Sender    string `json:"Sender" binding:"required"`
	Recepient string `json:"Recepient" binding:"required"`
	Body      string `json:"Body" binding:"required"`
}

func GetUsers(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	shardStr := c.Query("shard")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	shard, err := strconv.Atoi(shardStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Missed 'shard' parameter or wrong value")
		log.Error(fmt.Sprintf("Missed 'shard' parameter or wrong value: %v", shardStr), err.Error())
		return
	}

	list, err := services.Instance().Profiles().GetUsers(offset, limit, shard)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get users")
		log.Error("Unable to get users", err.Error())
		return
	}

	result := &UserListDTO{Data: convertUsers(list), Count: len(list), Offset: offset, Limit: limit}
	c.JSON(http.StatusOK, result)
}

func GetMyProfile(c *gin.Context) {
	userUuid, ok := c.Get(app.CTX_TOKEN_ID_KEY)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get to user profile")
		log.Error("Unable to get user profile", "Missed TOKEN ID in gin context")
		return
	}

	user, err := services.Instance().Profiles().GetUser(userUuid.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get to user profile")
			log.Error("Unable to get user profile", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, convertUser(user))
}

func GetUser(c *gin.Context) {
	userUuid := c.Param("uuid")

	if userUuid == "" {
		c.JSON(http.StatusBadRequest, "Missed 'uuid' parameted")
		return
	}

	user, err := services.Instance().Profiles().GetUser(userUuid)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get user")
			log.Error("Unable to get user", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, convertUser(user))
}

func CreateUser(c *gin.Context) {
	var dto UserCreateDTO

	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	possibleUserRoles := utilsEntities.GetPossibleUserRoles()
	if !utils.Contains(possibleUserRoles, dto.Role) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
		return
	}

	possibleUseStates := entities.GetPossibleUserStates()
	if !utils.Contains(possibleUseStates, dto.State) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create user. Wrong 'State' value. Possible values: %v", possibleUseStates))
		return
	}

	hashPassword, err := credentials.HashPassword(dto.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create user")
		log.Error("Unable to create user", err.Error())
		return
	}
	dto.Password = hashPassword

	uuid, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create user")
		log.Error("unable to create uuid for user", err.Error())
		return
	}

	userUuid := uuid.String()

	userId, err := services.Instance().Profiles().CreateUser(userUuid, dto.Login, dto.Email, dto.Password, dto.Role, dto.State)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create user")
		log.Error("Unable to create user", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created user. Id: %v. Uuid: %v", userId, userUuid))

	if err != nil {
		if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
			log.Error("unable to create user", "duplicate user email: "+dto.Email)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to create user")
			log.Error("Unable to create user", err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, userUuid)
}

func UpdateUser(c *gin.Context) {
	var dto UserEditDTO

	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, dto.Uuid) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to update user. User Uuid from body: %v", dto.Uuid))
		return
	}

	if dto.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user state. User Uuid from body: %v", dto.Uuid))
			return
		}
		if dto.Role != nil && *dto.Role == utilsEntities.USER_ROLE_OWNER {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to block owners. User Uuid from body: %v", dto.Uuid))
			return
		}
		if *dto.State == entities.USER_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleUserStates := entities.GetPossibleUserStates()
		if !utils.Contains(possibleUserStates, *dto.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update user. Wrong 'State' value. Possible values: %v", possibleUserStates))
			return
		}
	}

	// TODO: add confirmation flow for changing emails
	if dto.Email != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user email. User Uuid from body: %v", dto.Uuid))
			return
		}
	}

	if dto.Role != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update user role. User Uuid from body: %v", dto.Uuid))
			return
		}
		possibleUserRoles := utilsEntities.GetPossibleUserRoles()
		if !utils.Contains(possibleUserRoles, *dto.Role) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update user. Wrong 'Role' value. Possible values: %v", possibleUserRoles))
			return
		}
	}

	if dto.Password != nil {
		hashPassword, err := credentials.HashPassword(*dto.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Unable to update user")
			log.Error("Unable to update user", err.Error())
			return
		}
		dto.Password = &hashPassword
	}

	err := services.Instance().Profiles().UpdateUser(dto.Uuid, dto.Login, dto.Email, dto.Password, dto.Role, dto.State)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else if err.Error() == queries.ErrorUserDuplicateKey.Error() {
			c.JSON(http.StatusBadRequest, api.DUPLICATE_FOUND)
			log.Error("unable to update user", "duplicate user email: "+*dto.Email)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update user")
			log.Error("Unable to update user", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Updated user. Uuid: %v", dto.Uuid))

	err = services.Instance().Feed().UpdateUser(toFeedUserDTO(&dto))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update user")
		log.Error("Unable to update user at feed service", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func toFeedUserDTO(user *UserEditDTO) *feed.FeedUserDTO {
	result := &feed.FeedUserDTO{
		Uuid: user.Uuid,
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
	var dto UserDeleteDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, dto.Uuid) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to delete user. User UUID from body: %v", dto.Uuid))
		return
	}

	err := services.Instance().Profiles().DeleteUser(dto.Uuid)
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
	return UserDTO{Uuid: user.Uuid, Login: user.Login, Email: user.Email, Role: user.Role, State: user.State}
}
