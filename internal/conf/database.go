package conf

import "runtime"

// DatabaseConfig 保存 SQLite 双连接池配置。
type DatabaseConfig struct {
	Path              string
	BusyTimeoutMS     int
	ReadMaxOpenConns  int
	WriteMaxOpenConns int
}

// SearchConfig 保存搜索索引配置。
type SearchConfig struct {
	BlevePath string
}

// LoadDatabase 从环境变量读取 SQLite 配置，供无需完整服务配置的命令行工具使用。
func LoadDatabase() DatabaseConfig {
	return loadDatabaseConfig()
}

// loadDatabaseConfig 读取 SQLite 配置。
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Path:              envString("SQLITE_PATH", defaultDBPath),
		BusyTimeoutMS:     envInt("SQLITE_BUSY_TIMEOUT_MS", defaultSQLiteTimeout),
		ReadMaxOpenConns:  envIntWithFallback("SQLITE_READ_MAX_OPEN_CONNS", runtime.NumCPU()*4),
		WriteMaxOpenConns: 1,
	}
}
