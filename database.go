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
	createTableSQL = `  
		CREATE TABLE IF NOT EXISTS users (  
			userid INTEGER PRIMARY KEY  
		);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
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
	createTableSQL := `  
	CREATE TABLE IF NOT EXISTS userscores (  
		id TEXT PRIMARY KEY,  
		score INTEGER NOT NULL  
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		panic(err.Error())
	}
}

func UpdateScore(username string, score int) error {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var exists bool
	errs := db.QueryRow("SELECT EXISTS(SELECT 1 FROM userscores WHERE id = ? AND score > ?)", username, score).Scan(&exists)
	if errs != nil {
		return errs
	}

	// 如果ID存在，则更新score
	if exists {
		// 准备更新语句
		_, err = db.Exec("UPDATE userscores SET score = ? WHERE id = ?", score, username)
		if err != nil {
			return err
		}
	} else {
		Insert(username, score)
	}

	return nil
}

func AddUser(userid string) error {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		fmt.Println("打不开数据库！")
		panic(err.Error())
	}
	defer db.Close()
	insertUserSQL := `  
	INSERT INTO users (id) VALUES (?);  
	`

	_, errs := db.Exec(insertUserSQL, userid)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", errs)
	} else {
		fmt.Println("插入成功！！！")
	}

	return nil
}

func DeleteUser(userid string) error {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	deleteStmt, err := db.Prepare("DELETE FROM user WHERE id = ?")
	if err != nil {
		return err
	}
	defer deleteStmt.Close()

	// 执行删除操作
	_, err = deleteStmt.Exec(userid)
	if err != nil {
		return err
	}

	return nil
}

func QueryUser(userid string) bool {
	db, err := sql.Open("sqlite3", "./userscores.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var exists bool
	errs := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userid).Scan(&exists)
	if errs != nil {
		return false
	}
	return exists
}
