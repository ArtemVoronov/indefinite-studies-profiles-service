package credentials

import (
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"golang.org/x/crypto/bcrypt"
)

type CredentialsDTO struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CredentialsValidationResult struct {
	UserUuid string `json:"userUuid" binding:"required,email"`
	IsValid  bool   `json:"isValid" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

func CheckCredentials(email string, password string) (*CredentialsValidationResult, error) {
	var result *CredentialsValidationResult

	user, err := services.Instance().Profiles().GetUserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return &CredentialsValidationResult{UserUuid: "", IsValid: false, Role: ""}, nil
		}
		return result, fmt.Errorf("unable to check credentials : %s", err)
	}

	if user.State == entities.USER_STATE_CONFRIMED && isValidPassword(user.Password, password) {
		result = &CredentialsValidationResult{UserUuid: user.Uuid, IsValid: true, Role: user.Role}
	} else {
		result = &CredentialsValidationResult{UserUuid: "", IsValid: false, Role: ""}
	}

	return result, nil
}

func HashPassword(password string) (string, error) {
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
