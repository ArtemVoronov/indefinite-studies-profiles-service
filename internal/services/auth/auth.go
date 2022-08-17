package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

var authServiceBaseURL string
var verifyTokenURL string
var once sync.Once

func Setup() {
	once.Do(func() {
		authServiceBaseURL = utils.EnvVar("AUTH_SERVICE_BASE_URL")
		verifyTokenURL = authServiceBaseURL + "/api/v1/auth/verify-token"
	})
}

type VerificationDTO struct {
	AccessToken string `json:"accessToken" binding:"required"`
}

type VerificationResult struct {
	IsValid   bool
	IsExpired bool
}

// TODO: move settings to .env
var client *http.Client = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

func VerifyToken(token string) (*VerificationResult, error) {
	var result *VerificationResult

	body, err := json.Marshal(VerificationDTO{AccessToken: token})
	if err != nil {
		return result, fmt.Errorf("unable to verify token: %s", err)
	}

	req, err := http.NewRequest(http.MethodPost, verifyTokenURL, bytes.NewBuffer(body))
	if err != nil {
		return result, fmt.Errorf("unable to verify token: %s", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("unable to verify token: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return result, fmt.Errorf("unable to verify token, response status code: %v", resp.StatusCode)
	}

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("unable to verify token: %s", err)
	}

	err = json.Unmarshal(resBody, &result)
	if err != nil {
		return result, fmt.Errorf("unable to verify token: %s", err)
	}

	return result, nil
}
