package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
)

var ErrorRegistrationTokenDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"registration_tokens_token_unique\"")
var ErrorRestorePasswordTokenDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"restore_password_tokens_token_unique\"")

const (
	GET_REGISTRATION_TOKEN_QUERY_BY_TOKEN = `SELECT 
		user_id, token, expire_at, create_date, last_update_date
	FROM registration_tokens 
	WHERE token = $1`

	CREATE_REGISTRATION_TOKEN_QUERY = `INSERT INTO registration_tokens
		(user_id, token, expire_at, create_date, last_update_date)
		VALUES($1, $2, $3, $4, $5)`

	UPDATE_REGISTRATION_TOKEN_QUERY = `UPDATE registration_tokens
	SET token = $2,		
		expire_at = $3,
		last_update_date = $4
	WHERE user_id = $1`

	DELETE_REGISTRATION_TOKEN_QUERY = `DELETE from registration_tokens WHERE token = $1`

	GET_RESTORE_PASSWORD_TOKEN_QUERY_BY_TOKEN = `SELECT 
		user_id, token, expire_at, create_date, last_update_date
	FROM restore_password_tokens 
	WHERE token = $1`

	CREATE_RESTORE_PASSWORD_TOKEN_QUERY = `INSERT INTO restore_password_tokens
		(user_id, token, expire_at, create_date, last_update_date)
		VALUES($1, $2, $3, $4, $5)`

	UPDATE_RESTORE_PASSWORD_TOKEN_QUERY = `UPDATE restore_password_tokens
	SET token = $2,		
		expire_at = $3,
		last_update_date = $4
	WHERE user_id = $1`

	DELETE_RESTORE_PASSWORD_TOKEN_QUERY = `DELETE from restore_password_tokens WHERE token = $1`
)

func GetRegistrationToken(tx *sql.Tx, ctx context.Context, token string) (entities.RegistrationToken, error) {
	var result entities.RegistrationToken

	err := tx.QueryRowContext(ctx, GET_REGISTRATION_TOKEN_QUERY_BY_TOKEN, token).
		Scan(&result.UserId, &result.Token, &result.ExpireAt, &result.CreateDate, &result.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, err
		} else {
			return result, fmt.Errorf("error at loading registration token by token '%v' from db, case after QueryRow.Scan: %v", token, err)
		}
	}

	return result, nil
}

func CreateRegistrationToken(tx *sql.Tx, ctx context.Context, userId int, token string) error {
	createDate := time.Now()
	expireAt := createDate.Add(24 * time.Hour) // TODO: make as param?

	stmt, err := tx.PrepareContext(ctx, CREATE_REGISTRATION_TOKEN_QUERY)
	if err != nil {
		return fmt.Errorf("error at inserting registration token (UserId: '%v', Token: '%v'), case after preparing statement: %s", userId, token, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userId, token, expireAt, createDate, createDate)
	if err != nil {
		if err.Error() == ErrorRegistrationTokenDuplicateKey.Error() {
			return ErrorRegistrationTokenDuplicateKey
		}
		return fmt.Errorf("error at inserting registration token (UserId: '%v', Token: '%v'), case after ExecContext: %s", userId, token, err)
	}

	return nil
}

func UpdateRegistrationToken(tx *sql.Tx, ctx context.Context, userId int, token string) error {
	lastUpdateDate := time.Now()
	expireAt := lastUpdateDate.Add(24 * time.Hour) // TODO: make as param?
	stmt, err := tx.PrepareContext(ctx, UPDATE_REGISTRATION_TOKEN_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating registration token, case after preparing statement: %v", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, userId, token, expireAt, lastUpdateDate)
	if err != nil {
		if err.Error() == ErrorUserDuplicateKey.Error() {
			return ErrorUserDuplicateKey
		}
		return fmt.Errorf("error at updating registration token (Token: '%v'), case after executing statement: %v", token, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating registration token (Token: '%v'), case after counting affected rows: %v", token, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteRegistrationToken(tx *sql.Tx, ctx context.Context, token string) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_REGISTRATION_TOKEN_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting registration token, case after preparing statement: %v", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, token)
	if err != nil {
		return fmt.Errorf("error at deleting registration token by token '%v', case after executing statement: %v", token, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting registration token by token '%v', case after counting affected rows: %v", token, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetRestorePasswordToken(tx *sql.Tx, ctx context.Context, token string) (entities.RestorePasswordToken, error) {
	var result entities.RestorePasswordToken

	err := tx.QueryRowContext(ctx, GET_RESTORE_PASSWORD_TOKEN_QUERY_BY_TOKEN, token).
		Scan(&result.UserId, &result.Token, &result.ExpireAt, &result.CreateDate, &result.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, err
		} else {
			return result, fmt.Errorf("error at loading restore password token by token '%v' from db, case after QueryRow.Scan: %v", token, err)
		}
	}

	return result, nil
}

func CreateRestorePasswordToken(tx *sql.Tx, ctx context.Context, userId int, token string) error {
	createDate := time.Now()
	expireAt := createDate.Add(24 * time.Hour) // TODO: make as param?

	stmt, err := tx.PrepareContext(ctx, CREATE_RESTORE_PASSWORD_TOKEN_QUERY)
	if err != nil {
		return fmt.Errorf("error at inserting restore password token (UserId: '%v', Token: '%v'), case after preparing statement: %s", userId, token, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userId, token, expireAt, createDate, createDate)
	if err != nil {
		if err.Error() == ErrorRestorePasswordTokenDuplicateKey.Error() {
			return ErrorRestorePasswordTokenDuplicateKey
		}
		return fmt.Errorf("error at inserting restore password token (UserId: '%v', Token: '%v'), case after ExecContext: %s", userId, token, err)
	}

	return nil
}

func UpdateRestorePasswordToken(tx *sql.Tx, ctx context.Context, userId int, token string) error {
	lastUpdateDate := time.Now()
	expireAt := lastUpdateDate.Add(24 * time.Hour) // TODO: make as param?
	stmt, err := tx.PrepareContext(ctx, UPDATE_RESTORE_PASSWORD_TOKEN_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating restore password token, case after preparing statement: %v", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, userId, token, expireAt, lastUpdateDate)
	if err != nil {
		if err.Error() == ErrorUserDuplicateKey.Error() {
			return ErrorUserDuplicateKey
		}
		return fmt.Errorf("error at updating restore password token (Token: '%v'), case after executing statement: %v", token, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating restore password token (Token: '%v'), case after counting affected rows: %v", token, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteRestorePasswordToken(tx *sql.Tx, ctx context.Context, token string) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_RESTORE_PASSWORD_TOKEN_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting restore password token, case after preparing statement: %v", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, token)
	if err != nil {
		return fmt.Errorf("error at deleting restore password token by token '%v', case after executing statement: %v", token, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting restore password token by token '%v', case after counting affected rows: %v", token, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
