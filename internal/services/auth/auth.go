package auth

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

var once sync.Once
var instance *auth.AuthService

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

func Instance() *auth.AuthService {
	once.Do(func() {
		if instance == nil {
			instance = auth.CreateAuthService(client, utils.EnvVar("AUTH_SERVICE_BASE_URL"))
		}
	})
	return instance
}
