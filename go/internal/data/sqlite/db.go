package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB 初始化数据库连接并创建表结构（返回数据库连接）
func InitDB(dbPath string) (*sql.DB, error) {
	// 1. 打开数据库连接（文件不存在会自动创建）
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 2. 测试连接有效性
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 启用外键约束（SQLite默认禁用）
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("启用外键约束失败: %w", err)
	}

	// 3. 创建必要的表结构
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("创建表结构失败: %w", err)
	}

	return db, nil // 返回初始化好的数据库连接
}

// createTables 创建所有业务表（接收db参数，不再依赖全局变量）
func createTables(db *sql.DB) error {
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
	if _, err := db.Exec(assistantTableSQL); err != nil {
		return fmt.Errorf("创建assistants表失败: %w", err)
	}

	// 历史记录表（与助手关联，级联删除）
	historyTableSQL := `
	CREATE TABLE IF NOT EXISTS histories (
		assistant_id TEXT PRIMARY KEY,  -- 直接用助手ID作为主键
		messages TEXT NOT NULL,         -- JSON格式的消息列表
		FOREIGN KEY(assistant_id) REFERENCES assistants(id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(historyTableSQL); err != nil {
		return fmt.Errorf("创建histories表失败: %w", err)
	}

	return nil
}
