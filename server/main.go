package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFileName     = "db.sqlite3"
	createPostTable = `
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT
		)
	`
	insertPost  = "INSERT INTO posts (content) VALUES (?)"
	selectPosts = "SELECT id, content FROM posts"
	updatePost = "UPDATE posts SET content = ? WHERE id = ?"
	deletePost = "DELETE FROM posts WHERE id = ?"
)

type Post struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

func init() {
	// データベースとの接続
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// テーブルの作成
	_, err = db.Exec(createPostTable)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

func main() {
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getPosts(w, r, db)
		case http.MethodPost:
			createPost(w, r, db)
		case http.MethodPut:
			UpdatePost(w, r, db)
		case http.MethodDelete:
			DeletePost(w, r, db)
		}
	})

	log.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)
}

func getPosts(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query(selectPosts)
	if err != nil {
		http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Content); err != nil {
			http.Error(w, "Failed to scan post", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	respondJSON(w, http.StatusOK, posts)
}

func createPost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var post Post
	if err := decodeBody(r, &post); err != nil {
		respondJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := db.Exec(insertPost, post.Content)
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to fetch last insert ID", http.StatusInternalServerError)
		return
	}

	post.ID = int(id)
	respondJSON(w, http.StatusCreated, post)
}

func UpdatePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var post Post
    if err := decodeBody(r, &post); err != nil {
        respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    result, err := db.Exec(updatePost, post.Content, post.ID)
    if err != nil {
        http.Error(w, "Failed to update post", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, "Failed to fetch rows affected", http.StatusInternalServerError)
        return
    }

    if rowsAffected == 0 {
        http.Error(w, "No post found to update", http.StatusNotFound)
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{"message": "Post updated successfully"})
}

func DeletePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var post Post
    if err := decodeBody(r, &post); err != nil {
        respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    result, err := db.Exec(deletePost, post.ID)
    if err != nil {
        http.Error(w, "Failed to delete post", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, "Failed to fetch rows affected", http.StatusInternalServerError)
        return
    }

    if rowsAffected == 0 {
        http.Error(w, "No post found to delete", http.StatusNotFound)
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{"message": "Post deleted successfully"})
}

func decodeBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
