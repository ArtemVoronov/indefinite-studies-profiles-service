package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/lib/pq"
)

var ErrorUserDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"users_email_unique\"")

type CreateUserParams struct {
	Login    interface{}
	Email    interface{}
	Password interface{}
	Role     interface{}
	State    interface{}
}

type UpdateUserParams struct {
	Id       interface{}
	Login    interface{}
	Email    interface{}
	Password interface{}
	Role     interface{}
	State    interface{}
}

const (
	GET_USERS_QUERY = `SELECT 
		id, login, email, password, role, state, create_date, last_update_date
	FROM users 
	WHERE state != $3 
	LIMIT $1 OFFSET $2`

	GET_USER_QUERY_BY_ID = `SELECT 
		id, login, email, password, role, state, create_date, last_update_date
	FROM users 
	WHERE id = $1 and state != $2`

	GET_USER_QUERY_BY_EMAIL = `SELECT 
		id, login, email, password, role, state, create_date, last_update_date
	FROM users 
	WHERE email = $1 and state != $2`

	CREATE_USER_QUERY = `INSERT INTO users
		(login, email, password, role, state, create_date, last_update_date)
		VALUES($1, $2, $3, $4, $5, $6, $7)
	RETURNING id`

	UPDATE_USER_QUERY = `UPDATE users
	SET login = COALESCE($2, login),
		email = COALESCE($3, email),
		password = COALESCE($4, password),
		role = COALESCE($5, role),
		state = COALESCE($6, state),
		last_update_date = $7
	WHERE id = $1 and state != $8`

	DELETE_USER_QUERY = `DELETE from users WHERE id = $1`

	GET_USERS_BY_IDS_QUERY = `SELECT 
	id, login, email, password, role, state, create_date, last_update_date
	FROM users 
	WHERE state != $4 AND id = ANY($1)
	LIMIT $2 OFFSET $3`
)

func GetUsers(tx *sql.Tx, ctx context.Context, limit int, offset int) ([]entities.User, error) {
	var user []entities.User
	var (
		id             int
		login          string
		email          string
		password       string
		role           string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)

	rows, err := tx.QueryContext(ctx, GET_USERS_QUERY, limit, offset, entities.USER_STATE_DELETED)
	if err != nil {
		return user, fmt.Errorf("error at loading users from db, case after Query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &login, &email, &password, &role, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return user, fmt.Errorf("error at loading users from db, case iterating and using rows.Scan: %v", err)
		}
		user = append(user, entities.User{Id: id, Login: login, Email: email, Password: password, Role: role, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return user, fmt.Errorf("error at loading user from db, case after iterating: %v", err)
	}

	return user, nil
}

func GetUser(tx *sql.Tx, ctx context.Context, id int) (entities.User, error) {
	var user entities.User

	err := tx.QueryRowContext(ctx, GET_USER_QUERY_BY_ID, id, entities.USER_STATE_DELETED).
		Scan(&user.Id, &user.Login, &user.Email, &user.Password, &user.Role, &user.State, &user.CreateDate, &user.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, err
		} else {
			return user, fmt.Errorf("error at loading user by id '%v' from db, case after QueryRow.Scan: %v", id, err)
		}
	}

	return user, nil
}

func GetUsersByIds(tx *sql.Tx, ctx context.Context, ids []int, limit int, offset int) ([]entities.User, error) {
	var users []entities.User
	var (
		id             int
		login          string
		email          string
		password       string
		role           string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)

	rows, err := tx.QueryContext(ctx, GET_USERS_BY_IDS_QUERY, pq.Array(ids), limit, offset, entities.USER_STATE_DELETED)
	if err != nil {
		return users, fmt.Errorf("error at loading users by ids, case after Query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &login, &email, &password, &role, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return users, fmt.Errorf("error at loading users by ids from db, case iterating and using rows.Scan: %v", err)
		}
		users = append(users, entities.User{Id: id, Login: login, Email: email, Password: password, Role: role, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return users, fmt.Errorf("error at loading user by ids from db, case after iterating: %v", err)
	}

	return users, nil
}

func GetUserByEmail(tx *sql.Tx, ctx context.Context, email string) (entities.User, error) {
	var user entities.User

	err := tx.QueryRowContext(ctx, GET_USER_QUERY_BY_EMAIL, email, entities.USER_STATE_DELETED).
		Scan(&user.Id, &user.Login, &user.Email, &user.Password, &user.Role, &user.State, &user.CreateDate, &user.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, err
		} else {
			return user, fmt.Errorf("error at loading user by email '%v' from db, case after QueryRow.Scan: %v", email, err)
		}
	}

	return user, nil
}

func CreateUser(tx *sql.Tx, ctx context.Context, params *CreateUserParams) (int, error) {
	lastInsertId := -1

	createDate := time.Now()
	lastUpdateDate := time.Now()

	err := tx.QueryRowContext(ctx, CREATE_USER_QUERY,
		params.Login, params.Email, params.Password, params.Role, params.State, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		if err.Error() == ErrorUserDuplicateKey.Error() {
			return -1, ErrorUserDuplicateKey
		}
		return -1, fmt.Errorf("error at inserting user (Login: '%v', Email: '%v') into db, case after QueryRow.Scan: %v", params.Login, params.Email, err)
	}

	return lastInsertId, nil
}

func UpdateUser(tx *sql.Tx, ctx context.Context, params *UpdateUserParams) error {
	lastUpdateDate := time.Now()
	stmt, err := tx.PrepareContext(ctx, UPDATE_USER_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating user, case after preparing statement: %v", err)
	}
	res, err := stmt.ExecContext(ctx, params.Id, params.Login, params.Email, params.Password, params.Role, params.State, lastUpdateDate, entities.USER_STATE_DELETED)
	if err != nil {
		if err.Error() == ErrorUserDuplicateKey.Error() {
			return ErrorUserDuplicateKey
		}
		return fmt.Errorf("error at updating user (Id: %v), case after executing statement: %v", params.Id, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating user (Id: %v), case after counting affected rows: %v", params.Id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteUser(tx *sql.Tx, ctx context.Context, id int) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_USER_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting user, case after preparing statement: %v", err)
	}
	res, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("error at deleting user by id '%v', case after executing statement: %v", id, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting user by id '%v', case after counting affected rows: %v", id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
