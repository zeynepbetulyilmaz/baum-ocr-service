package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	TokenVersion int       `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Document struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	OriginalFilename string    `json:"original_filename"`
	StoredFilename   string    `json:"-"`
	Lang             string    `json:"lang"`
	Status           string    `json:"status"`
	PageCount        int       `json:"page_count"`
	TextPath         string    `json:"-"`
	PDFPath          string    `json:"-"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}