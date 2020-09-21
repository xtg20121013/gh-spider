package entity

import "gorm.io/gorm"

type UserInfo struct {
	gorm.Model
	Name string
	Index string
	Company string
	Location string
	Mail string
	Website string
	From string
 }