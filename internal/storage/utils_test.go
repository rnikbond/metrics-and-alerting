package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleToInt64() {

	someStruct := struct {
		value interface{}
	}{
		value: 1231,
	}

	intVal, _ := ToInt64(someStruct.value)

	fmt.Printf("value: %d\n", intVal)

	// Output:
	// value: 1231
}

func ExampleToFloat64() {

	someStruct := struct {
		value interface{}
	}{
		value: 1231.2222,
	}

	floatVal, _ := ToFloat64(someStruct.value)

	fmt.Printf("value: %.4f\n", floatVal)

	// Output:
	// value: 1231.2222
}

func TestToInt64(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Valid value {int} => [OK]",
			args: args{
				value: 123,
			},
			want:    123,
			wantErr: false,
		},
		{
			name: "Valid value {string} => [OK]",
			args: args{
				value: "123",
			},
			want:    123,
			wantErr: false,
		},
		{
			name: "Valid value {float} => [OK]",
			args: args{
				value: 123.123,
			},
			want:    123,
			wantErr: true,
		},
		{
			name:    "Invalid value {nil} => [ERROR]",
			wantErr: true,
		},
		{
			name: "Invalid value {empty string} => [ERROR]",
			args: args{
				value: "",
			},
			wantErr: true,
		},
		{
			name: "Invalid value {spaces string} => [ERROR]",
			args: args{
				value: "   ",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToInt64(tt.args.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "Valid value {float} => [OK]",
			args: args{
				value: 123.123,
			},
			want:    123.123,
			wantErr: false,
		},
		{
			name: "Valid value {string} => [OK]",
			args: args{
				value: "123.123",
			},
			want:    123.123,
			wantErr: false,
		},
		{
			name: "Valid value {int} => [OK]",
			args: args{
				value: 123,
			},
			want:    123,
			wantErr: false,
		},
		{
			name:    "Invalid value {nil} => [ERROR]",
			wantErr: true,
		},
		{
			name: "Invalid value {empty string} => [ERROR]",
			args: args{
				value: "",
			},
			wantErr: true,
		},
		{
			name: "Invalid value {spaces string} => [ERROR]",
			args: args{
				value: "   ",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFloat64(tt.args.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, got, tt.want)
			}
		})
	}
}
