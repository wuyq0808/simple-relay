package services

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/go-sql-driver/mysql"
)

type DatabaseService struct {
	db *sql.DB
}

type DatabaseConfig struct {
	User                   string
	Password               string
	Database               string
	InstanceConnectionName string
	UsePrivateIP           bool
}

func NewDatabaseService(config DatabaseConfig) (*DatabaseService, error) {
	d, err := cloudsqlconn.NewDialer(context.Background(), cloudsqlconn.WithLazyRefresh())
	if err != nil {
		return nil, fmt.Errorf("cloudsqlconn.NewDialer: %w", err)
	}

	var opts []cloudsqlconn.DialOption
	if config.UsePrivateIP {
		opts = append(opts, cloudsqlconn.WithPrivateIP())
	}

	mysql.RegisterDialContext("cloudsqlconn",
		func(ctx context.Context, addr string) (net.Conn, error) {
			return d.Dial(ctx, config.InstanceConnectionName, opts...)
		})

	dbURI := fmt.Sprintf("%s:%s@cloudsqlconn(localhost:3306)/%s?parseTime=true",
		config.User, config.Password, config.Database)

	db, err := sql.Open("mysql", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(7)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return &DatabaseService{db: db}, nil
}

func (ds *DatabaseService) Close() error {
	return ds.db.Close()
}

func (ds *DatabaseService) CreateTokensTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS oauth_tokens (
		id INT AUTO_INCREMENT PRIMARY KEY,
		client_id VARCHAR(255) NOT NULL,
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		scope VARCHAR(255),
		organization_uuid VARCHAR(255),
		organization_name VARCHAR(255),
		account_uuid VARCHAR(255),
		account_email VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE(client_id)
	)`
	
	_, err := ds.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tokens table: %w", err)
	}
	return nil
}