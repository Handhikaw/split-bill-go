package entity

type Category struct {
	ID        int64  `gorm:"id"`
	Name      string `gorm:"name"`
	Icon      string `gorm:"icon"`
	ColorCode string `gorm:"color_code"`
	IsDefault bool   `gorm:"is_default"`
	UserID    *int64 `gorm:"user_id"` // Pointer agar bisa nil
}

func (Category) TableName() string {
	return "category"
}
