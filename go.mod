module cirrina

go 1.19

require (
	github.com/gdamore/tcell/v2 v2.6.0
	github.com/google/uuid v1.3.0
	github.com/jinzhu/configor v1.2.1
	github.com/kontera-technologies/go-supervisor/v2 v2.1.0
	github.com/rivo/tview v0.0.0-20230621164836-6cc0565babaf
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
	golang.org/x/term v0.8.0
	google.golang.org/grpc v1.57.0
	google.golang.org/protobuf v1.30.0
	gorm.io/driver/sqlite v1.4.4
	gorm.io/gorm v1.24.6
)

replace github.com/kontera-technologies/go-supervisor/v2 => ./go-supervisor

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.15 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230525234030-28d5490b6b19 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)
