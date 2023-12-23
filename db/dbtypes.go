package db

import (
	"github.com/pocketbase/pocketbase/tools/types"
)

type Source struct {
	Name        string `db:"name" json:"name"`
	Url         string `db:"url" json:"url"`
	BaseUrl     string `db:"base_url" json:"base_url"`
	Description string `db:"description" json:"description"`
	Type        string `db:"type" json:"type"`
}

type UserSource struct {
	Name        string `db:"name" json:"name"`
	Publication string `db:"publication" json:"publication"`
}

type CombSource struct {
	Name        string `db:"name" json:"name"`
	Publication string `db:"publication" json:"publication"`
	BaseUrl     string `db:"base_url" json:"base_url"`
}

type Post struct {
	Title       string         `db:"title" json:"title"`
	Description string         `db:"description" json:"description"`
	Url         string         `db:"url" json:"url"`
	Published   types.DateTime `db:"published" json:"published"`
	Image       string         `db:"image" json:"image"`
	Source      CombSource     `db:"source" json:"source"`
}
