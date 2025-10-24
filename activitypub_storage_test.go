package conformance

import "testing"

func TestSuite_RunActivityPubTests(t *testing.T) {
	type fields struct {
		types   TestType
		storage ActivityPubStorage
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name:   "empty",
			fields: fields{},
		},
		{
			name: "memstore",
			fields: fields{
				storage: &memStorage{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Suite{
				types:   tt.fields.types,
				storage: tt.fields.storage,
			}
			//mockTesting := testing.T{}
			s.RunActivityPubTests(t)
			//t.Logf(mockTesting.Output())
		})
	}
}
