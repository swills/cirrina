package vmswitch

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-test/deep"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest
func TestGetAllIfBridges(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetAllIfBridgesSuccess1",
			want:        []string{"bridge0", "bridge1"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestGetAllIfBridgesError1",
			want:        nil,
			wantErr:     true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := GetAllIfBridges()
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetAllIfBridges() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_getIfBridgeMembers(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_getIfBridgeMembersSuccess1",
			args:        args{name: "bridge0"},
			want:        []string{"em0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_getIfBridgeMembersError1",
			args:        args{name: "bridge0"},
			want:        nil,
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := getIfBridgeMembers(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("getIfBridgeMembers() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_createIfBridge(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_createIfBridgeSuccess1",
			args:        args{name: "bridge0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_createIfBridgeSuccess1",
			args:        args{name: ""},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_createIfBridgeSuccess1",
			args:        args{name: "garbage"},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "Test_createIfBridgeError1",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
		{
			name:        "error4",
			mockCmdFunc: "Test_createIfBridgeError4",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
		{
			name:        "error5",
			mockCmdFunc: "Test_createIfBridgeError5",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := createIfBridge(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("createIfBridge() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_actualIfBridgeCreate(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_actualIfBridgeCreateSuccess1",
			args:        args{name: "bridge0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_actualIfBridgeCreateError1",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := actualIfBridgeCreate(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("actualIfBridgeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembers(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_bridgeIfDeleteAllMembersSuccess1",
			args:        args{name: "bridge0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeIfDeleteAllMembersError1",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_bridgeIfDeleteAllMembersError2",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := bridgeIfDeleteAllMembers(testCase.args.name)

			if (err != nil) != testCase.wantErr {
				t.Errorf("bridgeIfDeleteAllMembers() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeIfDeleteMember(t *testing.T) {
	type args struct {
		bridgeName string
		memberName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_bridgeIfDeleteMemberSuccess1",
			args:        args{bridgeName: "bridge0", memberName: "em0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeIfDeleteMemberError1",
			args:        args{bridgeName: "bridge0", memberName: "em0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := bridgeIfDeleteMember(testCase.args.bridgeName, testCase.args.memberName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("bridgeIfDeleteMember() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembers(t *testing.T) {
	type args struct {
		bridgeName    string
		bridgeMembers []string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestCreateIfBridgeWithMembersSuccess1",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError1",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError2",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError3",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
		{
			name:        "emptyBridgeName",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError3",
			args:        args{bridgeName: "", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := CreateIfBridgeWithMembers(testCase.args.bridgeName, testCase.args.bridgeMembers)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CreateIfBridgeWithMembers() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestGetDummyBridgeName(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        string
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetDummyBridgeNameSuccess1",
			want:        "bridge32767",
		},
		{
			name:        "success2",
			mockCmdFunc: "TestGetDummyBridgeNameSuccess2",
			want:        "bridge32765",
		},
		{
			name:        "error1",
			mockCmdFunc: "TestGetDummyBridgeNameError1",
			want:        "",
		},
		{
			name:        "error2",
			mockCmdFunc: "TestGetDummyBridgeNameError2",
			want:        "",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := GetDummyBridgeName()
			if got != testCase.want {
				t.Errorf("GetDummyBridgeName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// test helpers from here down

//nolint:paralleltest
func TestGetAllIfBridgesSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("bridge0\nbridge1\nthis is test garbage\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetAllIfBridgesError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_getIfBridgeMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`

	fmt.Print(ifconfigOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_getIfBridgeMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_createIfBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createIfBridgeError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_createIfBridgeError4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("bridge0\nbridge1\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_createIfBridgeError5(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[2] == "create" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualIfBridgeCreateSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualIfBridgeCreateError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[2] == "create" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "deletem" {
		os.Exit(1)
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_bridgeIfDeleteMemberSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeIfDeleteMemberError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "deletem" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersError3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "addm" {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyBridgeNameSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyBridgeNameSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("bridge32767\nbridge32766\nbridge1\nbridge0\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyBridgeNameError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestGetDummyBridgeNameError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for i := 32767; i >= 0; i-- {
		fmt.Printf("bridge%d\n", i) //nolint:forbidigo
	}

	os.Exit(0)
}
