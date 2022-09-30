package services

import (
	"log"
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	auth "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	notifications "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/notifications"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/whitelist"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

type Services struct {
	auth          *auth.AuthGRPCService
	db            *db.PostgreSQLService
	feed          *feed.FeedBuilderGRPCService
	whitelist     *whitelist.WhiteListService
	notifications *notifications.NotificationsGRPCService
}

var once sync.Once
var instance *Services

func Instance() *Services {
	once.Do(func() {
		if instance == nil {
			instance = createServices()
		}
	})
	return instance
}

func createServices() *Services {
	authcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("AUTH_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials")
	}
	feedcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("FEED_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials")
	}
	notificationscreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("NOTIFICATIONS_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials")
	}

	return &Services{
		auth:          auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &authcreds),
		feed:          feed.CreateFeedBuilderGRPCService(utils.EnvVar("FEED_SERVICE_GRPC_HOST")+":"+utils.EnvVar("FEED_SERVICE_GRPC_PORT"), &feedcreds),
		db:            db.CreatePostgreSQLService(),
		whitelist:     whitelist.CreateWhiteListService(utils.EnvVar("APP_WHITE_LIST_PATH")),
		notifications: notifications.CreateNotificationsGRPCService(utils.EnvVar("NOTIFICATIONS_SERVICE_GRPC_HOST")+":"+utils.EnvVar("NOTIFICATIONS_SERVICE_GRPC_PORT"), &notificationscreds),
	}
}

func (s *Services) Shutdown() {
	s.auth.Shutdown()
	s.db.Shutdown()
	s.feed.Shutdown()
	s.whitelist.Shutdown()
}

func (s *Services) DB() *db.PostgreSQLService {
	return s.db
}

func (s *Services) Auth() *auth.AuthGRPCService {
	return s.auth
}

func (s *Services) Feed() *feed.FeedBuilderGRPCService {
	return s.feed
}

func (s *Services) Whitelist() *whitelist.WhiteListService {
	return s.whitelist
}

func (s *Services) Notifications() *notifications.NotificationsGRPCService {
	return s.notifications
}
