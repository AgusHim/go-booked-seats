package models

type User struct {
	ID       string `json:"id" gorm:"primaryKey"`
	Name     string `json:"name"`
	Email    string `json:"email" gorm:"unique"`
	Role     string `json:"role"`
	Password string `json:"password,omitempty" gorm:"type:text"`
}
