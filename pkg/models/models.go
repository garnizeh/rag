package models

// Domain models matching the database schema in db/migrations/0001_init.sql

type Engineer struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name" validate:"required"`
	Email        string `json:"email" db:"email" validate:"required,email"`
	Updated      int64  `json:"updated" db:"updated"`
	PasswordHash string `json:"password_hash,omitempty" db:"password_hash"`
}

type Profile struct {
	ID         int64  `json:"id" db:"id"`
	EngineerID int64  `json:"engineer_id" db:"engineer_id"`
	Bio        string `json:"bio,omitempty" db:"bio"`
	Updated    int64  `json:"updated" db:"updated"`
}

type Activity struct {
	ID         int64  `json:"id" db:"id"`
	EngineerID int64  `json:"engineer_id" db:"engineer_id"`
	Activity   string `json:"activity" db:"activity"`
	Created    int64  `json:"created" db:"created"`
}

type Question struct {
	ID         int64  `json:"id" db:"id"`
	EngineerID int64  `json:"engineer_id" db:"engineer_id"`
	Question   string `json:"question" db:"question"`
	Answered   *int64 `json:"answered,omitempty" db:"answered"`
	Created    int64  `json:"created" db:"created"`
}

type Job struct {
	ID      int64  `json:"id" db:"id"`
	Status  string `json:"status" db:"status"`
	Created int64  `json:"created" db:"created"`
}
