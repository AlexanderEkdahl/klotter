package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// Message ...
type Message struct {
	ID        int       `json:"id"`
	Message   string    `json:"message"`
	UserID    string    `json:"-"`
	X         float32   `json:"x"`
	Y         float32   `json:"y"`
	CreatedAt time.Time `json:"created_at"`
	Comments  []Comment
}

// Comment ...
type Comment struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	UserID    string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// RepoCreateComment ...
func RepoCreateComment(messageID int, c Comment) (Comment, error) {
	var id int
	var createdAt time.Time
	err := db.QueryRow("INSERT INTO comments(content, message_id, user_id) VALUES ($1, $2, $3) RETURNING id, created_at", c.Content, messageID, c.UserID).Scan(&id, &createdAt)
	if err != nil {
		return Comment{}, err
	}
	c.ID = id
	c.CreatedAt = createdAt
	return c, nil
}

// RepoCreateMessage ...
func RepoCreateMessage(m Message) (Message, error) {
	query := fmt.Sprintf("INSERT INTO messages(message, location, user_id) VALUES ($1, ST_GeographyFromText('SRID=4326;POINT(%v %v)'), $2) RETURNING id, created_at", m.X, m.Y)
	var id int
	var createdAt time.Time
	err := db.QueryRow(query, m.Message, m.UserID).Scan(&id, &createdAt)
	if err != nil {
		return Message{}, err
	}
	m.ID = id
	m.CreatedAt = createdAt
	return m, nil
}

// RepoFindComments ...
func RepoFindComments(id int) ([]Comment, error) {
	rows, err := db.Query("SELECT id, content, created_at FROM comments WHERE message_id = $1", id)
	if err != nil {
		return nil, err
	}

	comments := []Comment{}
	for rows.Next() {
		var r Comment
		if err := rows.Scan(&r.ID, &r.Content, &r.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

// RepoFindMessage find all messages and their respective comments. Warning: N + 1 queries
func RepoFindMessage(x float32, y float32) ([]Message, error) {
	query := fmt.Sprintf("SELECT id, message, ST_X(location::geometry) as x, ST_Y(location::geometry) as y, created_at FROM messages WHERE ST_DWithin(location, ST_GeographyFromText('SRID=4326;POINT(%v %v)'), 10000) ORDER BY messages.created_at DESC", x, y)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	messages := []Message{}
	for rows.Next() {
		var r Message
		if err := rows.Scan(&r.ID, &r.Message, &r.X, &r.Y, &r.CreatedAt); err != nil {
			return nil, err
		}

		comments, err := RepoFindComments(r.ID)
		if err != nil {
			return nil, err
		}
		r.Comments = comments

		messages = append(messages, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// RepoFindMessageWithUser find all messages that a user participated in and their respective comments. Warning: N + 1 queries
func RepoFindMessageWithUser(userID string) ([]Message, error) {
	query := "SELECT messages.id, messages.message, ST_X(messages.location::geometry) as x, ST_Y(messages.location::geometry) as y, messages.created_at FROM messages LEFT JOIN comments ON comments.message_id = messages.id WHERE messages.user_id = $1 OR comments.user_id = $2  ORDER BY messages.created_at DESC"
	rows, err := db.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}

	messages := []Message{}
	for rows.Next() {
		var r Message
		if err := rows.Scan(&r.ID, &r.Message, &r.X, &r.Y, &r.CreatedAt); err != nil {
			return nil, err
		}

		comments, err := RepoFindComments(r.ID)
		if err != nil {
			return nil, err
		}
		r.Comments = comments

		messages = append(messages, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func init() {
	var err error

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("You need to set DATABASE_URL environement variable")
	}

	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
}
