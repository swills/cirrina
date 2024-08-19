package vmswitch

import (
	"testing"

	"github.com/google/uuid"
)

func TestSwitch_BeforeCreate(t *testing.T) {
	type fields struct {
		ID   string
		Name string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "SuccessIDNotSet",
			fields: fields{
				ID:   "",
				Name: "bridge0",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDJunk",
			fields: fields{
				ID:   "asdfasdfasdf",
				Name: "bridge0",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDWrongFormat",
			fields: fields{
				ID:   "597b5f690ae9400e96ec779d05694728",
				Name: "bridge0",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDSet",
			fields: fields{
				ID:   "074f8bc5-2c98-4cac-b6f1-ac86af497192",
				Name: "bridge0",
			},
			wantErr: false,
		},
		{
			name: "FailNameNotSet",
			fields: fields{
				ID:   "ad0e6bec-a26c-4657-b9d1-30676a932c23",
				Name: "",
			},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testSwitch := &Switch{
				ID:   testCase.fields.ID,
				Name: testCase.fields.Name,
			}

			err := testSwitch.BeforeCreate(nil)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}

			_, err = uuid.Parse(testSwitch.ID)
			if err != nil {
				t.Fatalf("error parsing uuid: %s", err.Error())
			}
		})
	}
}

func TestSwitch_BeforeCreateNilReceiver(t *testing.T) {
	t.Parallel()

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()

		testISO := (*Switch)(nil)

		err := testISO.BeforeCreate(nil)
		if err == nil {
			t.Errorf("BeforeCreate() nil receiver did not return error, error = %v", err)
		}
	})
}
