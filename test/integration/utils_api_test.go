//go:build integration
// +build integration

package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
)

var testHttpClient TestHttpClient = TestHttpClient{}

type UsersApi interface {
	CreateUser(login any, email any, password any, role any, state any) (int, string, error)
	GetUser(id string) (int, string)
	GetUsers(limit any, offset any) (int, string, error)
	UpdateUser(id any, login any, email any, password any, role any, state any) (int, string, error)
	DeleteUser(id any) (int, string, error)
}

type PingApi interface {
	Ping() (int, string, error)
	SafePing() (int, string, error)
}

type TestApi interface {
	UsersApi
	PingApi
}

type TestHttpClient struct {
}

func (p *TestHttpClient) CreateUser(login any, email any, password any, role any, state any) (int, string, error) {
	body, err := CreateUserPutOrPostBody(login, email, password, role, state)
	if err != nil {
		return -1, "", err
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String(), nil
}

func (p *TestHttpClient) GetUser(id string) (int, string) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/"+id, nil)
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func (p *TestHttpClient) GetUsers(limit any, offset any) (int, string, error) {
	queryParams, err := CreateLimitAndOffsetQueryParams(limit, offset)
	if err != nil {
		return -1, "", err
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users"+queryParams, nil)
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String(), nil
}

func (p *TestHttpClient) UpdateUser(id any, login any, email any, password any, role any, state any) (int, string, error) {
	idParam, err := ParseForPathParam("id", id)
	if err != nil {
		return -1, "", err
	}
	body, err := CreateUserPutOrPostBody(login, email, password, role, state)
	if err != nil {
		return -1, "", err
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users"+idParam, bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String(), nil
}

func (p *TestHttpClient) DeleteUser(id any) (int, string, error) {
	idParam, err := ParseForPathParam("id", id)
	if err != nil {
		return -1, "", err
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users"+idParam, nil)
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String(), nil
}

func (p *TestHttpClient) Ping() (int, string, error) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Content-Type", "application/json")
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String(), nil
}

func (p *TestHttpClient) SafePing(accessToken string) (int, string, error) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/safe-ping", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	TestRouter.ServeHTTP(w, req)
	return w.Code, w.Body.String(), nil
}

func ParseForJsonBody(paramName string, paramValue any) (string, error) {
	result := ""
	switch paramType := paramValue.(type) {
	case int:
		result = "\"" + paramName + "\": " + strconv.Itoa(paramValue.(int))
	case string:
		result = "\"" + paramName + "\": \"" + paramValue.(string) + "\""
	case nil:
		result = ""
	default:
		return "", fmt.Errorf("unkown type for '%s': %v", paramName, paramType)
	}
	return result, nil
}

func ParseForPathParam(paramName string, paramValue any) (string, error) {
	result := ""
	switch paramType := paramValue.(type) {
	case int:
		result = "/" + strconv.Itoa(paramValue.(int))
	case string:
		result = "/" + paramValue.(string)
	case nil:
		result = ""
	default:
		return "", fmt.Errorf("unkown type for '%s': %v", paramName, paramType)
	}
	return result, nil
}

func ParseForQueryParam(paramName string, paramValue any) (string, error) {
	result := ""
	switch paramType := paramValue.(type) {
	case int:
		result = paramName + "=" + strconv.Itoa(paramValue.(int))
	case nil:
		result = ""
	default:
		return "", fmt.Errorf("unkown type for '%s': %v", paramName, paramType)
	}
	return result, nil
}

func CreateLimitAndOffsetQueryParams(limit any, offset any) (string, error) {
	limitQueryParam, err := ParseForQueryParam("limit", limit)
	if err != nil {
		return "", err
	}
	offsetQueryParam, err := ParseForQueryParam("offset", offset)
	if err != nil {
		return "", err
	}

	queryParams := ""
	if limitQueryParam != "" && offsetQueryParam != "" {
		queryParams += "?" + limitQueryParam + "&" + offsetQueryParam
	} else if limitQueryParam != "" {
		queryParams += "?" + limitQueryParam
	} else if offsetQueryParam != "" {
		queryParams += "?" + offsetQueryParam
	}

	return queryParams, nil
}

func CreateUserPutOrPostBody(login any, email any, password any, role any, state any) (string, error) {
	loginField, err := ParseForJsonBody("Login", login)
	if err != nil {
		return "", err
	}
	emailField, err := ParseForJsonBody("Email", email)
	if err != nil {
		return "", err
	}
	passwordField, err := ParseForJsonBody("Password", password)
	if err != nil {
		return "", err
	}
	roleField, err := ParseForJsonBody("Role", role)
	if err != nil {
		return "", err
	}
	stateField, err := ParseForJsonBody("State", state)
	if err != nil {
		return "", err
	}

	result := "{"
	if loginField != "" {
		result += loginField + ","
	}
	if emailField != "" {
		result += emailField + ","
	}
	if passwordField != "" {
		result += passwordField + ","
	}
	if roleField != "" {
		result += roleField + ","
	}
	if stateField != "" {
		result += stateField + ","
	}
	if len(result) != 1 {
		result = result[:len(result)-1]
	}
	result += "}"
	return result, nil
}
