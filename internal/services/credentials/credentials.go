package credentials

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"golang.org/x/crypto/bcrypt"
)

type CredentialsDTO struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CredentialsValidationResult struct {
	UserId  int  `json:"userId" binding:"required,email"`
	IsValid bool `json:"isValid" binding:"required"`
}

func CheckCredentials(email string, password string) (*CredentialsValidationResult, error) {
	var result *CredentialsValidationResult

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
