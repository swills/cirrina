module cirrina

go 1.21

require (
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/color v1.16.0
	github.com/gdamore/tcell/v2 v2.7.1
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-version v1.6.0
	github.com/jedib0t/go-pretty/v6 v6.5.4
	github.com/kontera-technologies/go-supervisor/v2 v2.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/rivo/tview v0.0.0-20240225120200-5605142ca62e
	github.com/rxwycdh/rxhash v0.0.0-20230131062142-10b7a38b400d
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
	golang.org/x/sys v0.17.0
	golang.org/x/term v0.17.0
	google.golang.org/grpc v1.62.0
	google.golang.org/protobuf v1.32.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/sqlite v1.5.5
	gorm.io/gorm v1.25.7
)

replace github.com/kontera-technologies/go-supervisor/v2 => gitlab.com/swills/go-supervisor/v2 v2.1.1-0.20230409155131-5a8ec9493b2d

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240221002015-b0ce06bbee7c // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
