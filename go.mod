module cirrina

go 1.19

require (
	github.com/google/uuid v1.3.0
	github.com/jinzhu/configor v1.2.1
	github.com/kontera-technologies/go-supervisor/v2 v2.1.0
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.28.1
	gorm.io/driver/sqlite v1.4.4
	gorm.io/gorm v1.24.6
)

replace github.com/kontera-technologies/go-supervisor/v2 => ../go-supervisor

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.15 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)
