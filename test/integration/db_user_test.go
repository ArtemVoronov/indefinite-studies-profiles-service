//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db/queries"
	"github.com/stretchr/testify/assert"
)

func TestDBUserGet(t *testing.T) {
	t.Run("NotFoundCase", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			_, err := queries.GetUser(tx, ctx, 1)

			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		expected := utils.entityGenerators.GenerateUser(1)
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, expected.Login, expected.Email, expected.Password, expected.Role, expected.State)

			assert.Nil(t, err)
			assert.Equal(t, expected.Id, userId)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			actual, err := queries.GetUser(tx, ctx, expected.Id)

			utils.asserts.AssertEqualUsers(t, expected, actual)
			return err
		})()
	})))
	t.Run("TimeoutError", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at loading user by id '%d' from db, case after QueryRow.Scan: %s", 1, "context deadline exceeded")
			_, err := tx.ExecContext(ctx, "SELECT pg_sleep(10)")
			_, err = queries.GetUser(tx, ctx, 1)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
	t.Run("ContextCancelled", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at loading user by id '%d' from db, case after QueryRow.Scan: %s", 1, "context canceled")
			cancel()
			_, err := queries.GetUser(tx, ctx, 1)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
}

func TestDBUserCreate(t *testing.T) {
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Nil(t, err)
			assert.Equal(t, userId, 1)
			return err
		})()
	})))
	t.Run("DuplicateCase", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Nil(t, err)
			assert.NotEqual(t, userId, -1)
			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			_, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Equal(t, db.ErrorUserDuplicateKey, err)
			return err
		})()
	})))
	t.Run("TimeoutError", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at inserting user (Login: '%s', Email: '%s') into db, case after QueryRow.Scan: %s", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, "context deadline exceeded")
			_, err := tx.ExecContext(ctx, "SELECT pg_sleep(10)")
			_, err = queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
	t.Run("ContextCancelled", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at inserting user (Login: '%s', Email: '%s') into db, case after QueryRow.Scan: %s", TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, "context canceled")
			cancel()
			_, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
}

func TestDBUserGetAll(t *testing.T) {
	t.Run("ExpectedEmpty", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			users, err := queries.GetUsers(tx, ctx, 50, 0)

			assert.Nil(t, err)
			assert.Equal(t, 0, len(users))
			return err
		})()
	})))
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		var expectedUsers []entities.User
		for i := 1; i <= 10; i++ {
			expectedUsers = append(expectedUsers, utils.entityGenerators.GenerateUser(i))
		}
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := CreateUsersInDB(t, tx, ctx, 10, TEST_USER_LOGIN_TEMPLATE, TEST_USER_EMAIL_TEMPLATE, TEST_USER_PASSORD_TEMPLATE, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			actualUsers, err := queries.GetUsers(tx, ctx, 50, 0)

			assert.Nil(t, err)
			utils.asserts.AssertEqualUserArrays(t, expectedUsers, actualUsers)
			return err
		})()
	})))
	t.Run("LimitParameterCase", RunWithRecreateDB((func(t *testing.T) {
		var expectedUsers []entities.User
		for i := 1; i <= 5; i++ {
			expectedUsers = append(expectedUsers, utils.entityGenerators.GenerateUser(i))
		}
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := CreateUsersInDB(t, tx, ctx, 10, TEST_USER_LOGIN_TEMPLATE, TEST_USER_EMAIL_TEMPLATE, TEST_USER_PASSORD_TEMPLATE, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			actualUsers, err := queries.GetUsers(tx, ctx, 5, 0)

			assert.Nil(t, err)
			utils.asserts.AssertEqualUserArrays(t, expectedUsers, actualUsers)
			return err
		})()
	})))
	t.Run("OffsetParameterCase", RunWithRecreateDB((func(t *testing.T) {
		var expectedUsers []entities.User
		for i := 6; i <= 10; i++ {
			expectedUsers = append(expectedUsers, utils.entityGenerators.GenerateUser(i))
		}
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := CreateUsersInDB(t, tx, ctx, 10, TEST_USER_LOGIN_TEMPLATE, TEST_USER_EMAIL_TEMPLATE, TEST_USER_PASSORD_TEMPLATE, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			actualUsers, err := queries.GetUsers(tx, ctx, 50, 5)

			assert.Nil(t, err)
			utils.asserts.AssertEqualUserArrays(t, expectedUsers, actualUsers)
			return err
		})()
	})))
	t.Run("TimeoutError", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at loading users from db, case after Query: %s", "context deadline exceeded")
			_, err := tx.ExecContext(ctx, "SELECT pg_sleep(10)")
			_, err = queries.GetUsers(tx, ctx, 50, 0)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
	t.Run("ContextCancelled", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at loading users from db, case after Query: %s", "context canceled")
			cancel()
			_, err := queries.GetUsers(tx, ctx, 50, 0)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
}

func TestDBUserUpdate(t *testing.T) {
	t.Run("NotFoundCase", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.UpdateUser(tx, ctx, 1, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("DeletedCase", RunWithRecreateDB((func(t *testing.T) {
		expectedUserId := 1
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Nil(t, err)
			assert.Equal(t, expectedUserId, userId)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.DeleteUser(tx, ctx, expectedUserId)

			assert.Nil(t, err)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.UpdateUser(tx, ctx, expectedUserId, TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_2)

			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		expected := utils.entityGenerators.GenerateUser(1)

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_2)

			assert.Nil(t, err)
			assert.Equal(t, expected.Id, userId)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.UpdateUser(tx, ctx, expected.Id, expected.Login, expected.Email, expected.Password, expected.Role, expected.State)

			assert.Nil(t, err)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			actual, err := queries.GetUser(tx, ctx, expected.Id)

			utils.asserts.AssertEqualUsers(t, expected, actual)
			return err
		})()
	})))
	t.Run("DuplicateCase", RunWithRecreateDB((func(t *testing.T) {
		expectedUserId1 := 1
		expectedUserId2 := 2
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Nil(t, err)
			assert.Equal(t, expectedUserId1, userId)
			return err
		})()

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_2, TEST_USER_EMAIL_2, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_2)

			assert.Nil(t, err)
			assert.Equal(t, expectedUserId2, userId)
			return err
		})()

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.UpdateUser(tx, ctx, expectedUserId2, TEST_USER_LOGIN_2, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_2, TEST_USER_ROLE_2, TEST_USER_STATE_1)

			assert.Equal(t, db.ErrorUserDuplicateKey, err)
			return err
		})()
	})))
	t.Run("TimeoutError", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at updating user, case after preparing statement: %s", "context deadline exceeded")
			_, err := tx.ExecContext(ctx, "SELECT pg_sleep(10)")
			err = queries.UpdateUser(tx, ctx, 1, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
	t.Run("ContextCancelled", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at updating user, case after preparing statement: %s", "context canceled")
			cancel()
			err := queries.UpdateUser(tx, ctx, 1, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)
			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
}

func TestDBUserDelete(t *testing.T) {
	t.Run("NotFoundCase", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.DeleteUser(tx, ctx, 1)

			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("AlreadyDeletedCase", RunWithRecreateDB((func(t *testing.T) {
		expectedUserId := 1
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, TEST_USER_LOGIN_1, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			assert.Nil(t, err)
			assert.Equal(t, expectedUserId, userId)
			return err
		})()

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.DeleteUser(tx, ctx, expectedUserId)

			assert.Nil(t, err)
			return err
		})()

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.DeleteUser(tx, ctx, expectedUserId)

			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		var expectedUsers []entities.User
		expectedUsers = append(expectedUsers, utils.entityGenerators.GenerateUser(1))
		expectedUsers = append(expectedUsers, utils.entityGenerators.GenerateUser(3))

		userIdToDelete := 2

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := CreateUsersInDB(t, tx, ctx, 3, TEST_USER_LOGIN_TEMPLATE, TEST_USER_EMAIL_TEMPLATE, TEST_USER_PASSORD_TEMPLATE, TEST_USER_ROLE_1, TEST_USER_STATE_1)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			err := queries.DeleteUser(tx, ctx, userIdToDelete)

			assert.Nil(t, err)
			return err
		})()

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			users, err := queries.GetUsers(tx, ctx, 50, 0)

			assert.Nil(t, err)
			utils.asserts.AssertEqualUserArrays(t, expectedUsers, users)
			return err
		})()

		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			_, err := queries.GetUser(tx, ctx, userIdToDelete)

			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("TimeoutError", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at deleting user, case after preparing statement: %s", "context deadline exceeded")
			_, err := tx.ExecContext(ctx, "SELECT pg_sleep(10)")
			err = queries.DeleteUser(tx, ctx, 1)

			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
	t.Run("ContextCancelled", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			expectedError := fmt.Errorf("error at deleting user, case after preparing statement: %s", "context canceled")
			cancel()
			err := queries.DeleteUser(tx, ctx, 1)
			assert.Equal(t, expectedError, err)
			return err
		})()
	})))
}

func TestDBUserCredentials(t *testing.T) {
	t.Run("NotFoundCase", RunWithRecreateDB((func(t *testing.T) {
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, isValid, err := queries.IsValidCredentials(tx, ctx, TEST_USER_EMAIL_1, TEST_USER_PASSWORD_1)

			assert.Equal(t, -1, userId)
			assert.Equal(t, false, isValid)
			assert.Equal(t, sql.ErrNoRows, err)
			return err
		})()
	})))
	t.Run("BasicCase", RunWithRecreateDB((func(t *testing.T) {
		user := utils.entityGenerators.GenerateUser(1)
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, user.Login, user.Email, user.Password, user.Role, user.State)

			assert.Nil(t, err)
			assert.Equal(t, user.Id, userId)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, isValid, err := queries.IsValidCredentials(tx, ctx, user.Email, user.Password)

			assert.Equal(t, user.Id, userId)
			assert.Equal(t, true, isValid)
			return err
		})()
	})))
	t.Run("WrongPassword", RunWithRecreateDB((func(t *testing.T) {
		user := utils.entityGenerators.GenerateUser(1)
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, err := queries.CreateUser(tx, ctx, user.Login, user.Email, user.Password, user.Role, user.State)

			assert.Nil(t, err)
			assert.Equal(t, user.Id, userId)

			return err
		})()
		db.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
			userId, isValid, err := queries.IsValidCredentials(tx, ctx, user.Email, user.Password+"some_suffix")

			assert.Equal(t, user.Id, userId)
			assert.Equal(t, false, isValid)
			return err
		})()
	})))
}
