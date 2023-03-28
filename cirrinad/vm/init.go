package vm

func init() {
	db := getVmDb()
	err := db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
	}
	err = db.AutoMigrate(&Config{})
	if err != nil {
		panic("failed to auto-migrate Configs")
	}
}
