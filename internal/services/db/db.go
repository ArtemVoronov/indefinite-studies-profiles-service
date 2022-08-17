package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	_ "github.com/lib/pq"
)

// TODO: reorder funcs, types, vars in 'db' namespace

type Singleton interface {
	GetDB() *sql.DB
	GetTimeout() time.Duration
}

type singleton struct {
	rwmutex sync.RWMutex
	db      *sql.DB
	timeout time.Duration
}

var once sync.Once
var instance *singleton

func GetInstance() Singleton {
	once.Do(func() {
		if instance == nil {
			instance = new(singleton)
			instance.setDB(createDatabase())
			instance.setTimeout(getDefaultQueryTimeout())
		}
	})
	return instance
}

func (s *singleton) setDB(sqldb *sql.DB) {
	s.rwmutex.Lock()
	defer s.rwmutex.Unlock()
	s.db = sqldb
}

func (s *singleton) setTimeout(timeout time.Duration) {
	s.rwmutex.Lock()
	defer s.rwmutex.Unlock()
	s.timeout = timeout
}

func (s *singleton) GetDB() *sql.DB {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()
	return s.db
}

func (s *singleton) GetTimeout() time.Duration {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()
	return s.timeout
}

var ErrorTaskDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"tasks_name_state_unique\"")
var ErrorTagDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"tags_name_state_unique\"")
var ErrorUserDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"users_email_state_unique\"")

func createDatabase() *sql.DB {
	dbEnvVars := [6]string{"DATABASE_HOST", "DATABASE_PORT", "DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_SSL_MODE"}
	var variables []string
	for _, element := range dbEnvVars {
		value := utils.EnvVar(element)
		variables = append(variables, value)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", variables[0], variables[1], variables[2], variables[3], variables[4], variables[5])
	result, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to database : %s", err)
	}

	// log.Printf("----- Database service setup succeed. Database name: %s -----", variables[4])

	return result
}

func getDefaultQueryTimeout() time.Duration {
	valueStr := utils.EnvVarDefault("DATABASE_QUERY_TIMEOUT_IN_SECONDS", "30")

	valueInt, err := strconv.Atoi(valueStr)

	if err != nil {
		log.Printf("Unable to read 'DATABASE_QUERY_TIMEOUT_IN_SECONDS' from config, using default value for 30 seconds")
		return 30 * time.Second
	}

	return time.Duration(valueInt) * time.Second
}

type SqlQueryFunc func(transaction *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error)
type SqlQueryFuncVoid func(transaction *sql.Tx, ctx context.Context, cancel context.CancelFunc) error

func Tx(f SqlQueryFunc) func() (any, error) {
	database := GetInstance().GetDB()
	timeout := GetInstance().GetTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return func() (any, error) {
		defer cancel()
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return -1, fmt.Errorf("error at creating tx: %s", err)
		}
		defer tx.Rollback()
		result, err := f(tx, ctx, cancel)
		if err != nil {
			return result, err
		}
		err = tx.Commit()
		if err != nil {
			return -1, fmt.Errorf("error at commiting tx: %s", err)
		}
		return result, err
	}
}

func TxVoid(f SqlQueryFuncVoid) func() error {
	database := GetInstance().GetDB()
	timeout := GetInstance().GetTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return func() error {
		defer cancel()
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("error at creating tx: %s", err)
		}
		defer tx.Rollback()
		err = f(tx, ctx, cancel)
		if err != nil {
			return err
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("error at commiting tx: %s", err)
		}
		return err
	}
}
