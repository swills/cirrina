package requests

import (
	"cirrina/cirrinad/vm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"time"
)

func GetUnStartedReq() Request {
	db := GetReqDB()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)
	return rs
}

func StartReq(rs Request) {
	db := vm.GetVMDB()
	rs.StartedAt.Time = time.Now()
	rs.StartedAt.Valid = true
	db.Model(&rs).Limit(1).Updates(rs)
}

func MarkReqSuccessful(rs *Request) *gorm.DB {
	db := GetReqDB()
	return db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func MarkReqFailed(rs *Request) *gorm.DB {
	db := GetReqDB()
	return db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}

func GetReqDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
	return db

}
