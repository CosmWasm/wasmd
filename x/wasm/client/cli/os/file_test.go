package os

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFileWithSizeLimit(t *testing.T) {
	filename := "file.go"
	file, err := os.ReadFile(filename)
	require.NoError(t, err)

	type args struct {
		name      string
		sizeLimit int64
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"cannot open error", args{"", 0}, nil, true},
		{"size limit over error", args{filename, 0}, nil, true},
		{"simple reading file success", args{filename, 100000}, file, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadFileWithSizeLimit(tt.args.name, tt.args.sizeLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFileWithSizeLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadFileWithSizeLimit() got = %v, want %v", got, tt.want)
			}
		})
	}
}
