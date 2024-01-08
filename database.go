package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

func CreateDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// 创建一个表来存储用户ID和用户分数
	createTableSQL := `  
	CREATE TABLE IF NOT EXISTS userscores (  
		id TEXT PRIMARY KEY,  
		score INTEGER NOT NULL  
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func Insert(userID string, score int) error {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		fmt.Println("打不开数据库！")
		panic(err.Error())
	}
	defer db.Close()
	insertUserSQL := `  
	INSERT INTO userscores (id, score) VALUES (?, ?);  
	`

	_, errs := db.Exec(insertUserSQL, userID, score)
	if db == nil {
		return fmt.Errorf("database does not exist : %w", errs)
	}

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", errs)
	} else {
		fmt.Println("插入成功！！！")
	}

	return nil
}

func PrintScoreTable() string {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	// 查询并打印所有用户数据
	queryAllUsersSQL := `  
	SELECT id, score FROM userscores;  
	`

	// 执行查询语句
	rows, err := db.Query(queryAllUsersSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	data := ""
	// 逐行读取并打印用户信息
	for rows.Next() {
		var username string
		var score int
		if err := rows.Scan(&username, &score); err != nil {
			log.Fatal(err)
		}
		data += "\n" + username + "   " + strconv.Itoa(score)
	}
	return data
}

func DeleteTable() {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tableName := "userscores"
	sqlStmt := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("Failed to delete table %s: %v", tableName, err)
	}
	fmt.Printf("Table %s deleted successfully.\n", tableName)
}
