package db

import (
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
)

var once sync.Once
var instance *db.PostgreSQLService

func Instance() *db.PostgreSQLService {
	once.Do(func() {
		if instance == nil {
			instance = db.CreatePostgreSQLService()
		}
	})
	return instance
}
