//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/queries"
	"github.com/stretchr/testify/assert"
)

const (

	TEST_USER_LOGIN_1    string = "Test user 1"
	TEST_USER_EMAIL_1    string = "user1@somewhere.com"
	TEST_USER_PASSWORD_1 string = "Test password1 "
	TEST_USER_ROLE_1     string = entities.USER_ROLE_OWNER
	TEST_USER_STATE_1    string = entities.USER_STATE_NEW
	TEST_USER_LOGIN_2    string = "Test user 2"
	TEST_USER_EMAIL_2    string = "user2@somewhere.com"
	TEST_USER_PASSWORD_2 string = "Tes tpassword 2"
	TEST_USER_ROLE_2     string = entities.USER_ROLE_RESIDENT
	TEST_USER_STATE_2    string = entities.USER_STATE_BLOCKED

	TEST_USER_LOGIN_TEMPLATE   string = "Test user "
	TEST_USER_EMAIL_TEMPLATE   string = "user%v@somewhere.com"
	TEST_USER_PASSORD_TEMPLATE string = "Test password "


type TestAsserts struct {
}

type TestUtilsAsserts interface {
	AssertEqualUsers(t *testing.T, expected entities.User, actual entities.User)
	AssertEqualUserArrays(t *testing.T, expected []entities.User, actual []entities.User)
}

func (p *TestAsserts) AssertEqualUsers(t *testing.T, expected entities.User, actual entities.User) {
	assert.Equal(t, expected.Id, actual.Id)
	assert.Equal(t, expected.Login, actual.Login)
	assert.Equal(t, expected.Email, actual.Email)
	assert.Equal(t, expected.Password, actual.Password)
	assert.Equal(t, expected.State, actual.State)
}

func (p *TestAsserts) AssertEqualUserArrays(t *testing.T, expected []entities.User, actual []entities.User) {
	assert.Equal(t, len(expected), len(actual))

	length := len(expected)
	for i := 0; i < length; i++ {
		utils.asserts.AssertEqualUsers(t, expected[i], actual[i])
	}
}

type TestEntityGenerators struct {
}

type TestUtilsGenerators interface {	
	GenerateUserLogin(template string, id int) string
	GenerateUserPassword(template string, id int) string
	GenerateUserEmail(template string, id int) string
	GenerateUser(id int) entities.User	
}

func (p *TestEntityGenerators) GenerateUserLogin(template string, id int) string {
	return template + strconv.Itoa(id)
}

func (p *TestEntityGenerators) GenerateUserPassword(template string, id int) string {
	return template + strconv.Itoa(id)
}

func (p *TestEntityGenerators) GenerateUserEmail(template string, id int) string {
	return fmt.Sprintf(template, id)
}

func (p *TestEntityGenerators) GenerateUser(id int) entities.User {
	return entities.User{
		Id:       id,
		Login:    utils.entityGenerators.GenerateUserLogin(TEST_USER_LOGIN_TEMPLATE, id),
		Email:    utils.entityGenerators.GenerateUserEmail(TEST_USER_EMAIL_TEMPLATE, id),
		Password: utils.entityGenerators.GenerateUserPassword(TEST_USER_PASSORD_TEMPLATE, id),
		Role:     TEST_USER_ROLE_1,
		State:    TEST_USER_STATE_1,
	}
}

type TestUtilsQueries interface {	
	CreateUserInDB(t *testing.T, tx *sql.Tx, ctx context.Context, login string, email string, password string, role string, state string) (int, error)
	CreateUsersInDB(t *testing.T, tx *sql.Tx, ctx context.Context, count int, loginTemplate string, emailTemplate string, passwordTemplate string, role string, state string) error
}

func CreateUserInDB(t *testing.T, tx *sql.Tx, ctx context.Context, login string, email string, password string, role string, state string) (int, error) {
	userId, err := queries.CreateUser(tx, ctx, login, email, password, role, state)
	assert.Nil(t, err)
	assert.NotEqual(t, userId, -1)
	return userId, err
}

func CreateUsersInDB(t *testing.T, tx *sql.Tx, ctx context.Context, count int, loginTemplate string, emailTemplate string, passwordTemplate string, role string, state string) error {
	var lastErr error
	for i := 1; i <= count; i++ {
		_, err := CreateUserInDB(t, tx, ctx,
			utils.entityGenerators.GenerateUserLogin(loginTemplate, i),
			utils.entityGenerators.GenerateUserEmail(emailTemplate, i),
			utils.entityGenerators.GenerateUserPassword(passwordTemplate, i),
			role,
			state,
		)
		if err != nil {
			lastErr = err
		}
	}
	return lastErr
}