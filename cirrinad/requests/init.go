package requests

import "cirrina/cirrinad/util"

func init() {

	util.ValidateDbConfig()
	db := getReqDb()
	err := db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
}
