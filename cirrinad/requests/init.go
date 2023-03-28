package requests

func init() {
	db := getReqDb()
	err := db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
}
