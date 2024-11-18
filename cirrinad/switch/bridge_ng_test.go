package vmswitch

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/go-test/deep"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest,maintidx
func Test_ngGetNodes(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []NgNode
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_ngGetNodesSuccess1",
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "success2",
			mockCmdFunc: "Test_ngGetNodesSuccess2",
			want: []NgNode{{
				NodeName:  "ngctl23503",
				NodeType:  "socket",
				NodeID:    "0000001e",
				NodeHooks: 0,
			}},
			wantErr: false,
		},
		{
			name:        "success3",
			mockCmdFunc: "Test_ngGetNodesSuccess3",
			want: []NgNode{
				{
					NodeName:  "igb0",
					NodeType:  "ether",
					NodeID:    "00000001",
					NodeHooks: 0,
				},
				{
					NodeName:  "ix0",
					NodeType:  "ether",
					NodeID:    "00000002",
					NodeHooks: 2,
				},
				{
					NodeName:  "ue0",
					NodeType:  "ether",
					NodeID:    "00000003",
					NodeHooks: 0,
				},
				{
					NodeName:  "bridge0",
					NodeType:  "ether",
					NodeID:    "00000006",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet0",
					NodeType:  "bridge",
					NodeID:    "0000000b",
					NodeHooks: 2,
				},
				{
					NodeName:  "bridge1",
					NodeType:  "ether",
					NodeID:    "00000014",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet1",
					NodeType:  "bridge",
					NodeID:    "00000018",
					NodeHooks: 0,
				},
				{
					NodeName:  "ngctl23503",
					NodeType:  "socket",
					NodeID:    "0000001e",
					NodeHooks: 0,
				},
			},
			wantErr: false,
		},
		{
			name:        "success4",
			mockCmdFunc: "Test_ngGetNodesSuccess4",
			want: []NgNode{
				{
					NodeName:  "igb0",
					NodeType:  "ether",
					NodeID:    "00000001",
					NodeHooks: 0,
				},
				{
					NodeName:  "ix0",
					NodeType:  "ether",
					NodeID:    "00000002",
					NodeHooks: 2,
				},
				{
					NodeName:  "ue0",
					NodeType:  "ether",
					NodeID:    "00000003",
					NodeHooks: 0,
				},
				{
					NodeName:  "bridge0",
					NodeType:  "ether",
					NodeID:    "00000006",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet0",
					NodeType:  "bridge",
					NodeID:    "0000000b",
					NodeHooks: 2,
				},
				{
					NodeName:  "bridge1",
					NodeType:  "ether",
					NodeID:    "00000014",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet1",
					NodeType:  "bridge",
					NodeID:    "00000018",
					NodeHooks: 0,
				},
				{
					NodeName:  "ngctl23503",
					NodeType:  "socket",
					NodeID:    "0000001e",
					NodeHooks: 0,
				},
			},
			wantErr: false,
		},
		{
			name:        "success5",
			mockCmdFunc: "Test_ngGetNodesSuccess5",
			want: []NgNode{
				{
					NodeName:  "igb0",
					NodeType:  "ether",
					NodeID:    "00000001",
					NodeHooks: 0,
				},
				{
					NodeName:  "ix0",
					NodeType:  "ether",
					NodeID:    "00000002",
					NodeHooks: 2,
				},
				{
					NodeName:  "ue0",
					NodeType:  "ether",
					NodeID:    "00000003",
					NodeHooks: 0,
				},
				{
					NodeName:  "bridge0",
					NodeType:  "ether",
					NodeID:    "00000006",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet0",
					NodeType:  "bridge",
					NodeID:    "0000000b",
					NodeHooks: 2,
				},
				{
					NodeName:  "bridge1",
					NodeType:  "ether",
					NodeID:    "00000014",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet1",
					NodeType:  "bridge",
					NodeID:    "00000018",
					NodeHooks: 0,
				},
				{
					NodeName:  "ngctl23503",
					NodeType:  "socket",
					NodeID:    "0000001e",
					NodeHooks: 0,
				},
			},
			wantErr: false,
		},
		{
			name:        "success6",
			mockCmdFunc: "Test_ngGetNodesSuccess6",
			want: []NgNode{
				{
					NodeName:  "igb0",
					NodeType:  "ether",
					NodeID:    "00000001",
					NodeHooks: 0,
				},
				{
					NodeName:  "ix0",
					NodeType:  "ether",
					NodeID:    "00000002",
					NodeHooks: 2,
				},
				{
					NodeName:  "ue0",
					NodeType:  "ether",
					NodeID:    "00000003",
					NodeHooks: 0,
				},
				{
					NodeName:  "bridge0",
					NodeType:  "ether",
					NodeID:    "00000006",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet0",
					NodeType:  "bridge",
					NodeID:    "0000000b",
					NodeHooks: 2,
				},
				{
					NodeName:  "bridge1",
					NodeType:  "ether",
					NodeID:    "00000014",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet1",
					NodeType:  "bridge",
					NodeID:    "00000018",
					NodeHooks: 0,
				},
				{
					NodeName:  "ngctl23503",
					NodeType:  "socket",
					NodeID:    "0000001e",
					NodeHooks: 0,
				},
			},
			wantErr: false,
		},
		{
			name:        "success7",
			mockCmdFunc: "Test_ngGetNodesSuccess7",
			want: []NgNode{
				{
					NodeName:  "igb0",
					NodeType:  "ether",
					NodeID:    "00000001",
					NodeHooks: 0,
				},
				{
					NodeName:  "ix0",
					NodeType:  "ether",
					NodeID:    "00000002",
					NodeHooks: 2,
				},
				{
					NodeName:  "ue0",
					NodeType:  "ether",
					NodeID:    "00000003",
					NodeHooks: 0,
				},
				{
					NodeName:  "bridge0",
					NodeType:  "ether",
					NodeID:    "00000006",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet0",
					NodeType:  "bridge",
					NodeID:    "0000000b",
					NodeHooks: 2,
				},
				{
					NodeName:  "bridge1",
					NodeType:  "ether",
					NodeID:    "00000014",
					NodeHooks: 0,
				},
				{
					NodeName:  "bnet1",
					NodeType:  "bridge",
					NodeID:    "00000018",
					NodeHooks: 0,
				},
				{
					NodeName:  "ngctl23503",
					NodeType:  "socket",
					NodeID:    "0000001e",
					NodeHooks: 0,
				},
			},
			wantErr: false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_ngGetNodesError1",
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

			got, err := ngGetNodes()
			if (err != nil) != testCase.wantErr {
				t.Errorf("ngGetNodes() error = %v, wantErr %v", err, testCase.wantErr)

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
func TestGetAllNgBridges(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetAllNgBridgesSuccess1",
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "success2",
			mockCmdFunc: "TestGetAllNgBridgesSuccess2",
			want:        []string{"bnet0", "bnet1"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestGetAllNgBridgesError1",
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

			got, err := GetAllNgSwitches()
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetAllNgSwitches() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_getNgBridgeMembers(t *testing.T) {
	type args struct {
		bridge string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        []ngPeer
		wantErr     bool
	}{
		{
			name:        "success1",
			args:        args{bridge: "bnet0"},
			mockCmdFunc: "Test_getNgBridgeMembersSuccess1",
			want: []ngPeer{
				{
					LocalHook: "link1",
					PeerName:  "em0",
					PeerType:  "ether",
					PeerID:    "00000002",
					PeerHook:  "upper",
				},
				{
					LocalHook: "link0",
					PeerName:  "em0",
					PeerType:  "ether",
					PeerID:    "00000002",
					PeerHook:  "lower",
				},
			},
			wantErr: false,
		},
		{
			name:        "error",
			args:        args{bridge: "bnet0"},
			mockCmdFunc: "Test_getNgBridgeMembersError1",
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "error2",
			args:        args{bridge: "bnet0"},
			mockCmdFunc: "Test_getNgBridgeMembersError2",
			want: []ngPeer{
				{
					LocalHook: "link1",
					PeerName:  "em0",
					PeerType:  "ether",
					PeerID:    "00000002",
					PeerHook:  "upper",
				},
				{
					LocalHook: "link0",
					PeerName:  "em0",
					PeerType:  "ether",
					PeerID:    "00000002",
					PeerHook:  "lower",
				},
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := getNgBridgeMembers(testCase.args.bridge)
			if (err != nil) != testCase.wantErr {
				t.Errorf("getNgBridgeMembers() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_ngBridgeNextLink(t *testing.T) {
	type args struct {
		peers []ngPeer
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success1",
			args: args{peers: []ngPeer{
				{
					LocalHook: "link1",
					PeerName:  "em0",
					PeerType:  "ether",
					PeerID:    "00000002",
					PeerHook:  "upper",
				},
				{
					LocalHook: "link0",
					PeerName:  "em0",
					PeerType:  "ether",
					PeerID:    "00000002",
					PeerHook:  "lower",
				}},
			},
			want: "link2",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := ngBridgeNextLink(testCase.args.peers)

			if got != testCase.want {
				t.Errorf("ngBridgeNextLink() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func Test_createNgBridge(t *testing.T) {
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
			args:        args{name: "bnet0"},
			mockCmdFunc: "Test_createNgBridgeSuccess1",
			wantErr:     false,
		},
		{
			name:        "error1",
			args:        args{name: ""},
			mockCmdFunc: "Test_createNgBridgeSuccess1",
			wantErr:     true,
		},
		{
			name:        "error2",
			args:        args{name: "garbage"},
			mockCmdFunc: "Test_createNgBridgeSuccess1",
			wantErr:     true,
		},
		{
			name:        "error3",
			args:        args{name: "bnet0"},
			mockCmdFunc: "Test_createNgBridgeError3",
			wantErr:     true,
		},
		{
			name:        "error4",
			args:        args{name: "bnet0"},
			mockCmdFunc: "Test_createNgBridgeError4",
			wantErr:     true,
		},
		{
			name:        "error5",
			args:        args{name: "bnet0"},
			mockCmdFunc: "Test_createNgBridgeError5",
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := createNgBridge(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("createNgBridge() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_actualNgBridgeCreate(t *testing.T) {
	type args struct {
		netDev string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_actualNgBridgeCreateSuccess1",
			args:        args{netDev: "bnet0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_actualNgBridgeCreateError1",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_actualNgBridgeCreateError2",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "Test_actualNgBridgeCreateError3",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error4",
			mockCmdFunc: "Test_actualNgBridgeCreateError4",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error5",
			mockCmdFunc: "Test_actualNgBridgeCreateError5",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error6",
			mockCmdFunc: "Test_actualNgBridgeCreateError6",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error7",
			mockCmdFunc: "Test_actualNgBridgeCreateError7",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := actualNgBridgeCreate(testCase.args.netDev)
			if (err != nil) != testCase.wantErr {
				t.Errorf("actualNgBridgeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeNgDeletePeer(t *testing.T) {
	type args struct {
		bridgeName string
		hook       string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_bridgeNgDeletePeerSuccess1",
			args:        args{bridgeName: "bnet0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeNgDeletePeerError1",
			args:        args{bridgeName: "bnet0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := bridgeNgDeletePeer(testCase.args.bridgeName, testCase.args.hook)
			if (err != nil) != testCase.wantErr {
				t.Errorf("bridgeNgDeletePeer() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeNgDeleteAllPeers(t *testing.T) {
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
			mockCmdFunc: "Test_bridgeNgDeleteAllPeersSuccess1",
			args:        args{name: "bnet0"},
			wantErr:     false,
		},
		{
			name:        "success2",
			mockCmdFunc: "Test_bridgeNgDeleteAllPeersSuccess2",
			args:        args{name: "bnet0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeNgDeleteAllPeersError1",
			args:        args{name: "bnet0"},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_bridgeNgDeleteAllPeersError2",
			args:        args{name: "bnet0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := bridgeNgDeleteAllPeers(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("bridgeNgDeleteAllPeers() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_createNgBridgeWithMembers(t *testing.T) {
	type args struct {
		bridgeName    string
		bridgeMembers []string
	}

	tests := []struct {
		name            string
		mockCmdFunc     string
		hostIntStubFunc func() ([]net.Interface, error)
		args            args
		wantErr         bool
	}{
		{
			name:            "success1",
			mockCmdFunc:     "Test_createNgBridgeWithMembersSuccess1",
			hostIntStubFunc: StubCreateNgBridgeWithMembersSuccess2,
			args:            args{bridgeName: "bnet0", bridgeMembers: []string{}},
			wantErr:         false,
		},
		{
			name:            "success2",
			mockCmdFunc:     "Test_createNgBridgeWithMembersSuccess2",
			hostIntStubFunc: StubCreateNgBridgeWithMembersSuccess2,
			args:            args{bridgeName: "bnet0", bridgeMembers: []string{"em0"}},
			wantErr:         false,
		},
		{
			name:            "error1",
			mockCmdFunc:     "Test_createNgBridgeWithMembersError1",
			hostIntStubFunc: StubCreateNgBridgeWithMembersSuccess2,
			args:            args{bridgeName: "bnet0", bridgeMembers: []string{"em0"}},
			wantErr:         true,
		},
		{
			name:            "error2",
			mockCmdFunc:     "Test_createNgBridgeWithMembersError2",
			hostIntStubFunc: StubCreateNgBridgeWithMembersSuccess2,
			args:            args{bridgeName: "bnet0", bridgeMembers: []string{"em0"}},
			wantErr:         true,
		},
		{
			name:            "error3",
			mockCmdFunc:     "Test_createNgBridgeWithMembersSuccess1",
			hostIntStubFunc: StubCreateNgBridgeWithMembersError3,
			args:            args{bridgeName: "bnet0", bridgeMembers: []string{"em0"}},
			wantErr:         false,
		},
		{
			name:            "error4",
			mockCmdFunc:     "Test_createNgBridgeWithMembersError4",
			hostIntStubFunc: StubCreateNgBridgeWithMembersSuccess2,
			args:            args{bridgeName: "bnet0", bridgeMembers: []string{"em0"}},
			wantErr:         false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			util.NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { util.NetInterfacesFunc = net.Interfaces })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := createNgBridgeWithMembers(testCase.args.bridgeName, testCase.args.bridgeMembers)
			if (err != nil) != testCase.wantErr {
				t.Errorf("createNgBridgeWithMembers() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeNgRemoveUplink(t *testing.T) {
	type args struct {
		bridgeName string
		peerName   string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_bridgeNgRemoveUplinkSuccess1",
			args:        args{bridgeName: "bnet0", peerName: "em0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeNgRemoveUplinkError1",
			args:        args{bridgeName: "bnet0", peerName: "em0"},
			wantErr:     true,
		},
		{
			name:        "success2",
			mockCmdFunc: "Test_bridgeNgRemoveUplinkSuccess2",
			args:        args{bridgeName: "bnet0", peerName: "em0"},
			wantErr:     false,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_bridgeNgRemoveUplinkError2",
			args:        args{bridgeName: "bnet0", peerName: "em0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := switchNgRemoveUplink(testCase.args.bridgeName, testCase.args.peerName)

			if (err != nil) != testCase.wantErr {
				t.Errorf("switchNgRemoveUplink() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

// test helpers from here down

func StubCreateNgBridgeWithMembersError3() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        0,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
		{
			Index:        1,
			MTU:          1500,
			Name:         "igb0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
	}, nil
}

//nolint:paralleltest
func Test_ngGetNodesSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesSuccess3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesSuccess4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
some garbage for testing field5 field6 field7 field8 field9
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesSuccess5(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
  Name: bnet2  field3 field4 field5 field6 field7 field8 field9
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesSuccess6(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
  Name: bnet1           Type: bridge  field5 field6 field7 field8 field9
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesSuccess7(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   field7 field8 field9
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_ngGetNodesError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestGetAllNgBridgesSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestGetAllNgBridgesSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetAllNgBridgesError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_getNgBridgeMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_getNgBridgeMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_getNgBridgeMembersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
this is some garbage for testing
  link0           em0             ether        00000002        lower          
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeError3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeError4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeError5(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "mkpeer" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for i := 32767; i >= 0; i-- {
		fmt.Printf("bridge%d\n", i) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[2] == "bridge32767" && cmdWithArgs[3] == "create" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateError3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[2] == "bridge32767" && cmdWithArgs[3] == "create" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "mkpeer" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateError4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "create" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "mkpeer" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "name" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateError5(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "create" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "mkpeer" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "name" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "connect" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualNgBridgeCreateError6(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "create" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "mkpeer" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "name" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "connect" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "msg" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest,cyclop
func Test_actualNgBridgeCreateError7(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "create" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "mkpeer" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "name" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "connect" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "msg" {
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "destroy" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgDeletePeerSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgDeletePeerError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_bridgeNgDeleteAllPeersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgDeleteAllPeersSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" {
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
this is some garbage for testing
  link0           em0             ether        00000002        lower          
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgDeleteAllPeersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgDeleteAllPeersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "rmhook" {
		os.Exit(1)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" {
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
this is some garbage for testing
  link0           em0             ether        00000002        lower          
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeWithMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeWithMembersSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func StubCreateNgBridgeWithMembersSuccess2() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        0,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
		{
			Index:        1,
			MTU:          1500,
			Name:         "em0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
	}, nil
}

//nolint:paralleltest
func Test_createNgBridgeWithMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_createNgBridgeWithMembersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createNgBridgeWithMembersError4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "connect" && cmdWithArgs[3] == "em0:" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgRemoveUplinkSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgRemoveUplinkError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgRemoveUplinkSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" {
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
		os.Exit(0)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeNgRemoveUplinkError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" {
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "rmhook" {
		os.Exit(1)
	}

	os.Exit(0)
}
