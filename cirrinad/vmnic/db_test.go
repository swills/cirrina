package vmnic

import (
	"testing"

	"github.com/google/uuid"
)

func TestVMNic_BeforeCreate(t *testing.T) {
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
				Name: "test2024081901_int0",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDJunk",
			fields: fields{
				ID:   "asdfasdfasdf",
				Name: "test2024081901_int0",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDWrongFormat",
			fields: fields{
				ID:   "255e2e2e8dd247a394b99a957739cb8d",
				Name: "test2024081901_int0",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDSet",
			fields: fields{
				ID:   "36169edf-22dc-4144-a2db-4496fb46aff9",
				Name: "test2024081901_int0",
			},
			wantErr: false,
		},
		{
			name: "FailNameNotSet",
			fields: fields{
				ID:   "b02a85c5-cde0-4cf8-8ee4-85469a346e63",
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

			testNic := &VMNic{
				ID:   testCase.fields.ID,
				Name: testCase.fields.Name,
			}

			err := testNic.BeforeCreate(nil)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}

			_, err = uuid.Parse(testNic.ID)
			if err != nil {
				t.Fatalf("error parsing uuid: %s", err.Error())
			}
		})
	}
}

func TestVMNic_BeforeCreateNilReceiver(t *testing.T) {
	t.Parallel()

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()

		testISO := (*VMNic)(nil)

		err := testISO.BeforeCreate(nil)
		if err == nil {
			t.Errorf("BeforeCreate() nil receiver did not return error, error = %v", err)
		}
	})
}
