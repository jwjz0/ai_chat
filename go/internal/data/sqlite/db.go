package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB 全局数据库连接（初始化后可用）
var DB *sql.DB

// InitDB 初始化数据库连接并创建表结构（核心函数）
// InitDB 初始化数据库连接并创建表结构（核心函数）
func InitDB(dbPath string) error {
	var err error
	// 1. 打开数据库连接（文件不存在会自动创建）
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 2. 测试连接有效性
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 新增：启用外键约束（SQLite默认禁用）
	if _, err := DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("启用外键约束失败: %w", err)
	}

	// 3. 创建必要的表结构
	if err := createTables(); err != nil {
		return fmt.Errorf("创建表结构失败: %w", err)
	}

	return nil
}

// createTables 创建所有业务表（含外键关联）
func createTables() error {
	// 助手表（核心表）
	assistantTableSQL := `
	CREATE TABLE IF NOT EXISTS assistants (
		id TEXT PRIMARY KEY,                   -- 助手唯一标识
		name TEXT,                             -- 助手名称
		description TEXT,                      -- 助手描述
		prompt TEXT,                           -- 提示词
		gmt_create TEXT,                       -- 创建时间
		gmt_modified TEXT,                     -- 修改时间
		time_stamp TEXT                        -- 时间戳
	);`
	if _, err := DB.Exec(assistantTableSQL); err != nil {
		return fmt.Errorf("创建assistants表失败: %w", err)
	}

	// 历史记录表（与助手关联，级联删除）
	historyTableSQL := `
		CREATE TABLE IF NOT EXISTS histories (
		assistant_id TEXT PRIMARY KEY,  -- 直接用助手ID作为主键
		messages TEXT NOT NULL,         -- JSON格式的消息列表
		FOREIGN KEY(assistant_id) REFERENCES assistants(id) ON DELETE CASCADE
	);`
	if _, err := DB.Exec(historyTableSQL); err != nil {
		return fmt.Errorf("创建histories表失败: %w", err)
	}

	return nil
}
