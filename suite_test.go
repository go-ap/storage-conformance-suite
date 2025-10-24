package conformance

import (
	"reflect"
	"testing"
)

func TestInit(t *testing.T) {
	type args struct {
		storage ActivityPubStorage
		tests   TestType
	}
	tests := []struct {
		name string
		args args
		want Suite
	}{
		{
			name: "empty",
			want: Suite{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Init(tt.args.storage, tt.args.tests); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Init() = %v, want %v", got, tt.want)
			}
		})
	}
}
