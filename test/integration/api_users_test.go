//go:build integration
// +build integration

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/entities"
	"github.com/stretchr/testify/assert"
)

var (
	ERROR_USER_LOGIN_IS_REQUIRED string = "{\"errors\":[" +
		"{\"Field\":\"Login\",\"Msg\":\"This field is required\"}" +
		"]}"
	ERROR_USER_EMAIL_IS_REQUIRED string = "{\"errors\":[" +
		"{\"Field\":\"Email\",\"Msg\":\"This field is required\"}" +
		"]}"
	ERROR_USER_PASSWORD_IS_REQUIRED string = "{\"errors\":[" +
		"{\"Field\":\"Password\",\"Msg\":\"This field is required\"}" +
		"]}"
	ERROR_USER_ROLE_IS_REQUIRED string = "{\"errors\":[" +
		"{\"Field\":\"Role\",\"Msg\":\"This field is required\"}" +
		"]}"
	ERROR_USER_STATE_IS_REQUIRED string = "{\"errors\":[" +
		"{\"Field\":\"State\",\"Msg\":\"This field is required\"}" +
		"]}"
	ERROR_USER_ALL_ARE_REQUIRED string = "{\"errors\":[" +
		"{\"Field\":\"Login\",\"Msg\":\"This field is required\"}," +
		"{\"Field\":\"Email\",\"Msg\":\"This field is required\"}," +
		"{\"Field\":\"Password\",\"Msg\":\"This field is required\"}," +
		"{\"Field\":\"Role\",\"Msg\":\"This field is required\"}," +
		"{\"Field\":\"State\",\"Msg\":\"This field is required\"}" +
		"]}"
	ERROR_USER_EMAIL_WRONG_FORMAT string = "{\"errors\":[" +
		"{\"Field\":\"Email\",\"Msg\":\"Wrong email format\"}" +
		"]}"
	ERROR_USER_CREATE_ROLE_WRONG_VALUE  string = fmt.Sprintf("Unable to create user. Wrong 'Role' value. Possible values: %v", entities.GetPossibleUserRoles())
	ERROR_USER_CREATE_STATE_WRONG_VALUE string = fmt.Sprintf("Unable to create user. Wrong 'State' value. Possible values: %v", entities.GetPossibleUserStates())
	ERROR_USER_UPDATE_ROLE_WRONG_VALUE  string = fmt.Sprintf("Unable to update user. Wrong 'Role' value. Possible values: %v", entities.GetPossibleUserRoles())
	ERROR_USER_UPDATE_STATE_WRONG_VALUE string = fmt.Sprintf("Unable to update user. Wrong 'State' value. Possible values: %v", entities.GetPossibleUserStates())
)

func TestApiUserGet(t *testing.T) {
	t.Run("NotFoundCase", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body := testHttpClient.GetUser("1")

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, "\""+api.PAGE_NOT_FOUND+"\"", body)
	})))
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		id := "1"
		login := "Test user 1"
		email := "user1@somewhere.com"
		password := "Test password 1"
		role := entities.USER_ROLE_OWNER
		state := entities.USER_STATE_NEW
		expectedBody := "{" +
			"\"Id\":" + id + "," +
			"\"Login\":\"" + login + "\"," +
			"\"Email\":\"" + email + "\"," +
			"\"Role\":\"" + role + "\"," +
			"\"State\":\"" + state + "\"" +
			"}"

		httpStatusCode, body, err := testHttpClient.CreateUser(login, email, password, role, state)

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, id, body)

		httpStatusCode, body = testHttpClient.GetUser(id)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, expectedBody, body)
	})))
	t.Run("WrongInput: 'Id' is a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body := testHttpClient.GetUser("text")

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_ID_WRONG_FORMAT+"\"", body)
	})))
	t.Run("WrongInput: 'Id' is a float", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body := testHttpClient.GetUser("2.15")

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_ID_WRONG_FORMAT+"\"", body)
	})))
	t.Run("WrongInput: 'Id' is a empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body := testHttpClient.GetUser("")

		assert.Equal(t, http.StatusMovedPermanently, httpStatusCode)
		assert.Equal(t, "<a href=\"/users\">Moved Permanently</a>.\n\n", body)
	})))
}

func TestApiUserGetAll(t *testing.T) {
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		expectedBody := "{"
		expectedBody += "\"Count\":10,"
		expectedBody += "\"Offset\":0,"
		expectedBody += "\"Limit\":50,"
		expectedBody += "\"Data\":["
		for i := 1; i <= 10; i++ {
			id := strconv.Itoa(i)
			login := utils.entityGenerators.GenerateUserLogin(TEST_USER_LOGIN_TEMPLATE, i)
			email := utils.entityGenerators.GenerateUserEmail(TEST_USER_EMAIL_TEMPLATE, i)
			password := utils.entityGenerators.GenerateUserPassword(TEST_USER_PASSORD_TEMPLATE, i)
			role := entities.USER_ROLE_OWNER
			state := entities.USER_STATE_NEW

			testHttpClient.CreateUser(login, email, password, role, state)
			expectedBody += "{" +
				"\"Id\":" + id + "," +
				"\"Login\":\"" + login + "\"," +
				"\"Email\":\"" + email + "\"," +
				"\"Role\":\"" + role + "\"," +
				"\"State\":\"" + state + "\"" +
				"}"
			if i != 10 {
				expectedBody += ","
			} else {
				expectedBody += "]"
			}
		}
		expectedBody += "}"

		httpStatusCode, body, _ := testHttpClient.GetUsers(nil, nil)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, expectedBody, body)
	})))
	t.Run("EmptyResult", RunWithRecreateDB((func(t *testing.T) {
		expectedBody := "{"
		expectedBody += "\"Count\":0,"
		expectedBody += "\"Offset\":0,"
		expectedBody += "\"Limit\":50,"
		expectedBody += "\"Data\":[]"
		expectedBody += "}"
		httpStatusCode, body, _ := testHttpClient.GetUsers(nil, nil)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, expectedBody, body)
	})))
	t.Run("LimitCase", RunWithRecreateDB((func(t *testing.T) {
		expectedBody := "{"
		expectedBody += "\"Count\":5,"
		expectedBody += "\"Offset\":0,"
		expectedBody += "\"Limit\":5,"
		expectedBody += "\"Data\":["
		for i := 1; i <= 5; i++ {
			id := strconv.Itoa(i)
			login := utils.entityGenerators.GenerateUserLogin(TEST_USER_LOGIN_TEMPLATE, i)
			email := utils.entityGenerators.GenerateUserEmail(TEST_USER_EMAIL_TEMPLATE, i)
			role := entities.USER_ROLE_OWNER
			state := entities.USER_STATE_NEW
			expectedBody += "{" +
				"\"Id\":" + id + "," +
				"\"Login\":\"" + login + "\"," +
				"\"Email\":\"" + email + "\"," +
				"\"Role\":\"" + role + "\"," +
				"\"State\":\"" + state + "\"" +
				"}"
			if i != 5 {
				expectedBody += ","
			} else {
				expectedBody += "]"
			}
		}
		expectedBody += "}"

		for i := 1; i <= 10; i++ {
			login := utils.entityGenerators.GenerateUserLogin(TEST_USER_LOGIN_TEMPLATE, i)
			email := utils.entityGenerators.GenerateUserEmail(TEST_USER_EMAIL_TEMPLATE, i)
			password := utils.entityGenerators.GenerateUserPassword(TEST_USER_PASSORD_TEMPLATE, i)
			role := entities.USER_ROLE_OWNER
			state := entities.USER_STATE_NEW

			testHttpClient.CreateUser(login, email, password, role, state)
		}

		httpStatusCode, body, _ := testHttpClient.GetUsers(5, 0)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, expectedBody, body)
	})))
	t.Run("OffsetCase", RunWithRecreateDB((func(t *testing.T) {
		expectedBody := "{"
		expectedBody += "\"Count\":5,"
		expectedBody += "\"Offset\":5,"
		expectedBody += "\"Limit\":50,"
		expectedBody += "\"Data\":["
		for i := 6; i <= 10; i++ {
			id := strconv.Itoa(i)
			login := utils.entityGenerators.GenerateUserLogin(TEST_USER_LOGIN_TEMPLATE, i)
			email := utils.entityGenerators.GenerateUserEmail(TEST_USER_EMAIL_TEMPLATE, i)
			role := entities.USER_ROLE_OWNER
			state := entities.USER_STATE_NEW
			expectedBody += "{" +
				"\"Id\":" + id + "," +
				"\"Login\":\"" + login + "\"," +
				"\"Email\":\"" + email + "\"," +
				"\"Role\":\"" + role + "\"," +
				"\"State\":\"" + state + "\"" +
				"}"
			if i != 10 {
				expectedBody += ","
			} else {
				expectedBody += "]"
			}
		}
		expectedBody += "}"

		for i := 1; i <= 10; i++ {
			login := utils.entityGenerators.GenerateUserLogin(TEST_USER_LOGIN_TEMPLATE, i)
			email := utils.entityGenerators.GenerateUserEmail(TEST_USER_EMAIL_TEMPLATE, i)
			password := utils.entityGenerators.GenerateUserPassword(TEST_USER_PASSORD_TEMPLATE, i)
			role := entities.USER_ROLE_OWNER
			state := entities.USER_STATE_NEW

			testHttpClient.CreateUser(login, email, password, role, state)
		}

		httpStatusCode, body, _ := testHttpClient.GetUsers(50, 5)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, expectedBody, body)
	})))
}

func TestApiUserCreate(t *testing.T) {
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, "1", body)
	})))
	t.Run("WrongInput: Missed 'Login'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(nil, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_LOGIN_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'Email'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, nil, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_EMAIL_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'Password'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, nil, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_PASSWORD_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'Role'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, nil, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_ROLE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'State'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, nil)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_STATE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed all reqired", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(nil, nil, nil, nil, nil)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_ALL_ARE_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Login' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser("", TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_LOGIN_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Email' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, "", TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_EMAIL_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Password' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, "", TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_PASSWORD_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Role' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, "", TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_ROLE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'State' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, "")
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_STATE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Login' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Email' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, 1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Password' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, 1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Role' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, 1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'State' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, 1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Role' has a value that not from enum", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, "MISSED TEST ROLE", TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+ERROR_USER_CREATE_ROLE_WRONG_VALUE+"\"", body)
	})))
	t.Run("WrongInput: 'State' has a value that not from enum", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, "MISSED TEST STATE")

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+ERROR_USER_CREATE_STATE_WRONG_VALUE+"\"", body)
	})))
	t.Run("WrongInput: 'Email' wrong format", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, "user1somewhere.com", TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_EMAIL_WRONG_FORMAT, body)
	})))
	t.Run("DuplicateCase", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, "1", body)

		httpStatusCode, body, _ = testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.DUPLICATE_FOUND+"\"", body)
	})))
	t.Run("DeletedCase: try to create as deleted", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, entities.USER_STATE_DELETED)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.DELETE_VIA_POST_REQUEST_IS_FODBIDDEN+"\"", body)
	})))
}

func TestApiUserUpdate(t *testing.T) {
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		id := "1"
		login := TEST_USER_LOGIN_2
		email := TEST_USER_EMAIL_2
		password := TEST_USER_PASSWORD_2
		role := TEST_USER_ROLE_2
		state := TEST_USER_STATE_2
		expectedBody := "{" +
			"\"Id\":" + id + "," +
			"\"Login\":\"" + login + "\"," +
			"\"Email\":\"" + email + "\"," +
			"\"Role\":\"" + role + "\"," +
			"\"State\":\"" + state + "\"" +
			"}"

		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, id, body)

		httpStatusCode, body, _ = testHttpClient.UpdateUser(id, login, email, password, role, state)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, "\""+api.DONE+"\"", body)

		httpStatusCode, body = testHttpClient.GetUser(id)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, expectedBody, body)

	})))
	t.Run("WrongInput: 'Id' is a empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, api.PAGE_NOT_FOUND, body)
	})))
	t.Run("WrongInput: 'Id' is a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("text", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_ID_WRONG_FORMAT+"\"", body)
	})))
	t.Run("WrongInput: 'Id' is a float", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("2.15", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_ID_WRONG_FORMAT+"\"", body)
	})))
	t.Run("WrongInput: Missed 'Login'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", nil, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_LOGIN_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'Email'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, nil, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_EMAIL_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'Password'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, nil, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_PASSWORD_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'Role'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, nil, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_ROLE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed 'State'", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, nil)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_STATE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: Missed all reqired", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", nil, nil, nil, nil, nil)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_ALL_ARE_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Login' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", "", TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_LOGIN_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Email' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, "", TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_EMAIL_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Password' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, "", TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_PASSWORD_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Role' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, "", TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_ROLE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'State' is empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, "")
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_STATE_IS_REQUIRED, body)
	})))
	t.Run("WrongInput: 'Login' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", 1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Email' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, 1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Password' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, 1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Role' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, 1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'State' is not a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, 1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_MESSAGE_PARSING_BODY_JSON+"\"", body)
	})))
	t.Run("WrongInput: 'Role' has a value that not from enum", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, "MISSED TEST ROLE", TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+ERROR_USER_UPDATE_ROLE_WRONG_VALUE+"\"", body)
	})))
	t.Run("WrongInput: 'State' has a value that not from enum", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, "MISSED TEST STATE")

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+ERROR_USER_UPDATE_STATE_WRONG_VALUE+"\"", body)
	})))
	t.Run("WrongInput: 'Email' wrong format", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, "user1somewhere.com", TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, ERROR_USER_EMAIL_WRONG_FORMAT, body)
	})))
	t.Run("NotFoundCase", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.UpdateUser("1", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, "\""+api.PAGE_NOT_FOUND+"\"", body)
	})))
	t.Run("DeletedCase: find deleted", RunWithRecreateDB((func(t *testing.T) {
		expectedId := "1"

		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, expectedId, body)

		httpStatusCode, body, _ = testHttpClient.DeleteUser(expectedId)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, "\""+api.DONE+"\"", body)

		httpStatusCode, body, _ = testHttpClient.UpdateUser(expectedId, TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_2)

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, "\""+api.PAGE_NOT_FOUND+"\"", body)
	})))
	t.Run("DeletedCase: try to mark as deleted", RunWithRecreateDB((func(t *testing.T) {
		expectedId := "1"

		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, expectedId, body)

		httpStatusCode, body, _ = testHttpClient.UpdateUser(expectedId, TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, entities.USER_STATE_DELETED)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN+"\"", body)
	})))

	t.Run("DuplicateCase", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, "1", body)

		httpStatusCode, body, _ = testHttpClient.CreateUser(TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_2)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, "2", body)

		httpStatusCode, body, _ = testHttpClient.UpdateUser("2", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.DUPLICATE_FOUND+"\"", body)
	})))
	t.Run("MultipleUpdateCase", RunWithRecreateDB((func(t *testing.T) {
		id := "1"
		expectedBody := "{" +
			"\"Id\":" + id + "," +
			"\"Login\":\"" + TEST_USER_LOGIN_2 + "\"," +
			"\"Email\":\"" + TEST_USER_EMAIL_2 + "\"," +
			"\"Role\":\"" + TEST_USER_ROLE_2 + "\"," +
			"\"State\":\"" + TEST_USER_STATE_2 + "\"" +
			"}"

		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, id, body)

		for i := 1; i <= 3; i++ {
			httpStatusCode, body, _ = testHttpClient.UpdateUser(id, TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_2)

			assert.Equal(t, http.StatusOK, httpStatusCode)
			assert.Equal(t, "\""+api.DONE+"\"", body)

			httpStatusCode, body = testHttpClient.GetUser(id)

			assert.Equal(t, http.StatusOK, httpStatusCode)
			assert.Equal(t, expectedBody, body)
		}
	})))
}

func TestApiUserDelete(t *testing.T) {
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		expectedId := "1"

		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, expectedId, body)

		httpStatusCode, body, _ = testHttpClient.DeleteUser(expectedId)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, "\""+api.DONE+"\"", body)

		httpStatusCode, body = testHttpClient.GetUser(expectedId)

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, "\""+api.PAGE_NOT_FOUND+"\"", body)
	})))
	t.Run("WrongInput: 'Id' is a empty string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.DeleteUser("")

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, api.PAGE_NOT_FOUND, body)
	})))
	t.Run("WrongInput: 'Id' is a string", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.DeleteUser("text")

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_ID_WRONG_FORMAT+"\"", body)
	})))
	t.Run("WrongInput: 'Id' is a float", RunWithRecreateDB((func(t *testing.T) {
		httpStatusCode, body, _ := testHttpClient.DeleteUser("2.15")

		assert.Equal(t, http.StatusBadRequest, httpStatusCode)
		assert.Equal(t, "\""+api.ERROR_ID_WRONG_FORMAT+"\"", body)
	})))
	t.Run("MultipleDeleteCase", RunWithRecreateDB((func(t *testing.T) {
		expectedId := "1"

		httpStatusCode, body, _ := testHttpClient.CreateUser(TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

		assert.Equal(t, http.StatusCreated, httpStatusCode)
		assert.Equal(t, expectedId, body)

		httpStatusCode, body, _ = testHttpClient.DeleteUser(expectedId)

		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Equal(t, "\""+api.DONE+"\"", body)

		httpStatusCode, body, _ = testHttpClient.DeleteUser(expectedId)

		assert.Equal(t, http.StatusNotFound, httpStatusCode)
		assert.Equal(t, "\""+api.PAGE_NOT_FOUND+"\"", body)
	})))
}
