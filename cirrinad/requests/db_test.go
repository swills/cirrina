package requests

import (
	"testing"

	"github.com/google/uuid"
)

func TestRequest_BeforeCreate(t *testing.T) {
	type fields struct {
		ID string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "SuccessIDNotSet",
			fields: fields{
				ID: "",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDJunk",
			fields: fields{
				ID: "782jgkfd189vjn",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDWrongFormat",
			fields: fields{
				ID: "e48088639d884f36a1994cc3d057c542",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDSet",
			fields: fields{
				ID: "544ab6b7-0df0-41e1-a841-afa5b0972b6c",
			},
			wantErr: false,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testReq := &Request{
				ID: testCase.fields.ID,
			}

			err := testReq.BeforeCreate(nil)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}

			_, err = uuid.Parse(testReq.ID)
			if err != nil {
				t.Fatalf("error parsing uuid: %s", err.Error())
			}
		})
	}
}

func TestRequest_BeforeCreateNilReceiver(t *testing.T) {
	t.Parallel()

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()

		testISO := (*Request)(nil)

		err := testISO.BeforeCreate(nil)
		if err == nil {
			t.Errorf("BeforeCreate() nil receiver did not return error, error = %v", err)
		}
	})
}
