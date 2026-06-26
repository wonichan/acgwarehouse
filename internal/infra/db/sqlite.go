package db

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

const (
	sqliteDriverName = "sqlite"
	sqliteConnMaxAge = time.Hour
)

// SQLite 保存 SQLite 读写双连接池。
type SQLite struct {
	Read  *gorm.DB
	Write *gorm.DB
}

// NewSQLite 创建 WAL 模式 SQLite 读写双连接池。
func NewSQLite(cfg conf.DatabaseConfig) (*SQLite, error) {
	if err := ensureParentDir(cfg.Path); err != nil {
		return nil, pkgerrors.WithMessage(err, "ensure sqlite parent dir")
	}

	writeDB, err := openPool(cfg, cfg.WriteMaxOpenConns)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "open sqlite write pool")
	}

	readDB, err := openPool(cfg, cfg.ReadMaxOpenConns)
	if err != nil {
		closeGORM(writeDB)
		return nil, pkgerrors.WithMessage(err, "open sqlite read pool")
	}

	if err := configurePragmas(writeDB); err != nil {
		closeGORM(readDB)
		closeGORM(writeDB)
		return nil, pkgerrors.WithMessage(err, "configure sqlite pragmas")
	}

	if err := AutoMigrate(writeDB); err != nil {
		closeGORM(readDB)
		closeGORM(writeDB)
		return nil, pkgerrors.WithMessage(err, "auto migrate sqlite")
	}

	return &SQLite{Read: readDB, Write: writeDB}, nil
}

// AutoMigrate 执行当前阶段已存在持久化对象的迁移。
func AutoMigrate(database *gorm.DB) error {
	if database == nil {
		return pkgerrors.New("sqlite database is nil")
	}
	if err := database.AutoMigrate(&po.User{}, &po.Image{}); err != nil {
		return pkgerrors.WithMessage(err, "migrate models")
	}
	return nil
}

// Close 关闭 SQLite 读写连接池。
func (s *SQLite) Close() error {
	if s == nil {
		return nil
	}
	return pkgerrors.WithStack(closeBoth(s.Read, s.Write))
}

// openPool 按指定连接数打开 SQLite GORM 连接池。
func openPool(cfg conf.DatabaseConfig, maxOpenConns int) (*gorm.DB, error) {
	sqlDB, err := sql.Open(sqliteDriverName, sqliteDSN(cfg))
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "open database sql pool")
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(sqliteConnMaxAge)

	database, err := gorm.Open(&sqlite.Dialector{Conn: sqlDB}, &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		_ = sqlDB.Close()
		return nil, pkgerrors.WithMessage(err, "open gorm sqlite")
	}
	return database, nil
}

// sqliteDSN 构造带 busy_timeout 和外键约束的 SQLite DSN。
func sqliteDSN(cfg conf.DatabaseConfig) string {
	values := url.Values{}
	values.Set("_pragma", "busy_timeout("+strconv.Itoa(cfg.BusyTimeoutMS)+")")
	values.Add("_pragma", "foreign_keys(1)")
	return cfg.Path + "?" + values.Encode()
}

// configurePragmas 在写池启用 WAL 与基础一致性配置。
func configurePragmas(database *gorm.DB) error {
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		if err := database.Exec(statement).Error; err != nil {
			return pkgerrors.WithMessage(err, statement)
		}
	}
	return nil
}

// closeBoth 关闭两个 GORM 连接池。
func closeBoth(readDB *gorm.DB, writeDB *gorm.DB) error {
	readErr := closeGORM(readDB)
	writeErr := closeGORM(writeDB)
	if readErr != nil {
		return readErr
	}
	return writeErr
}

// closeGORM 关闭单个 GORM 连接池。
func closeGORM(database *gorm.DB) error {
	if database == nil {
		return nil
	}
	sqlDB, err := database.DB()
	if err != nil {
		return pkgerrors.WithMessage(err, "get sql db")
	}
	if err := sqlDB.Close(); err != nil {
		return pkgerrors.WithMessage(err, "close sql db")
	}
	return nil
}

// ensureParentDir 确保 SQLite 数据库父目录存在。
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return pkgerrors.WithMessage(err, "make sqlite dir")
	}
	return nil
}
