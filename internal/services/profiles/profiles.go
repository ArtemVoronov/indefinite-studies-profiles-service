package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/shard"
)

type ProfilesService struct {
	clientShards []*db.PostgreSQLService
	ShardsNum    int
	shardService *shard.ShardService
}

func CreateProfilesService(clients []*db.PostgreSQLService) *ProfilesService {
	return &ProfilesService{
		clientShards: clients,
		ShardsNum:    len(clients),
		shardService: shard.CreateShardService(len(clients)),
	}
}

func (s *ProfilesService) Shutdown() error {
	result := []error{}
	l := len(s.clientShards)
	for i := 0; i < l; i++ {
		err := s.clientShards[i].Shutdown()
		if err != nil {
			result = append(result, err)
		}
	}
	if len(result) > 0 {
		errors.Join(result...)
	}
	return nil
}

func (s *ProfilesService) client(userUuid string) *db.PostgreSQLService {
	bucketIndex := s.shardService.GetBucketIndex(userUuid)
	bucket := s.shardService.GetBucketByIndex(bucketIndex)
	log.Info(fmt.Sprintf("bucket: %v\tbucketIndex: %v", bucket, bucketIndex))
	return s.clientShards[bucket]
}

func (s *ProfilesService) CreateUser(userUuid string, login string, email string, password string, role string, state string) (int, error) {
	var userId int = -1
	data, err := s.client(userUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		params := &queries.CreateUserParams{
			Uuid:     userUuid,
			Login:    login,
			Email:    email,
			Password: password,
			Role:     role,
			State:    state,
		}
		result, err := queries.CreateUser(tx, ctx, params)
		return result, err
	})()

	if err != nil || data == -1 {
		return userId, err
	}

	userId, ok := data.(int)
	if !ok {
		return userId, fmt.Errorf("unable to convert result into int")
	}
	return userId, nil
}

func (s *ProfilesService) UpdateUser(userUuid string, login *string, email *string, password *string, role *string, state *string) error {
	return s.client(userUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		params := &queries.UpdateUserParams{
			Uuid:     userUuid,
			Login:    login,
			Email:    email,
			Password: password,
			Role:     role,
			State:    state,
		}
		err := queries.UpdateUser(tx, ctx, params)
		return err
	})()
}

func (s *ProfilesService) DeleteUser(userUuid string) error {
	return s.client(userUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteUser(tx, ctx, userUuid)
		return err
	})()
}

func (s *ProfilesService) GetUser(userUuid string) (entities.User, error) {
	var result entities.User

	data, err := s.client(userUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetUser(tx, ctx, userUuid)
		return user, err
	})()
	if err != nil {
		return result, err
	}

	result, ok := data.(entities.User)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.User")
	}

	return result, nil
}

func (s *ProfilesService) GetUserByEmail(email string) (entities.User, error) {
	var result entities.User

	for shard := range s.clientShards {
		data, err := s.clientShards[shard].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
			user, err := queries.GetUserByEmail(tx, ctx, email)
			return user, err
		})()
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return result, err
		}

		result, ok := data.(entities.User)
		if !ok {
			return result, fmt.Errorf("unable to convert result into entities.User")
		}
		return result, nil
	}

	return result, sql.ErrNoRows
}

func (s *ProfilesService) GetUsers(offset int, limit int, shard int) ([]entities.User, error) {
	if shard > s.ShardsNum || shard < 0 {
		return nil, fmt.Errorf("unexpected shard number: %v", shard)
	}
	data, err := s.clientShards[shard].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		users, err := queries.GetUsers(tx, ctx, limit, offset)
		return users, err
	})()
	if err != nil {
		return nil, err
	}

	result, ok := data.([]entities.User)
	if !ok {
		return nil, fmt.Errorf("unable to convert result into []entities.User")
	}
	return result, nil
}

func (s *ProfilesService) CreateRegistrationToken(userUuid string, userId int, token string) error {
	return s.client(userUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		return queries.CreateRegistrationToken(tx, ctx, userId, token)
	})()
}

func (s *ProfilesService) UpsertRegistrationToken(userUuid string, userId int, token string) error {
	return s.client(userUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateRegistrationToken(tx, ctx, userId, token)
		if err == sql.ErrNoRows {
			err = queries.CreateRegistrationToken(tx, ctx, userId, token)
		}
		return err
	})()
}

func (s *ProfilesService) GetRegistrationToken(userUuid string, token string) (entities.RegistrationToken, error) {
	var result entities.RegistrationToken

	data, err := s.client(userUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		res, err := queries.GetRegistrationToken(tx, ctx, token)
		return res, err
	})()
	if err != nil {
		return result, err
	}

	result, ok := data.(entities.RegistrationToken)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.RegistrationToken")
	}

	return result, nil
}

func (s *ProfilesService) UpsertRestorePasswordToken(userUuid string, userId int, token string) error {
	return s.client(userUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateRestorePasswordToken(tx, ctx, userId, token)
		if err == sql.ErrNoRows {
			err = queries.CreateRestorePasswordToken(tx, ctx, userId, token)
		}
		return err
	})()
}

func (s *ProfilesService) GetRestorePasswordToken(userUuid string, token string) (entities.RestorePasswordToken, error) {
	var result entities.RestorePasswordToken

	data, err := s.client(userUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		res, err := queries.GetRestorePasswordToken(tx, ctx, token)
		return res, err
	})()
	if err != nil {
		return result, err
	}

	result, ok := data.(entities.RestorePasswordToken)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.RestorePasswordToken")
	}

	return result, nil
}
