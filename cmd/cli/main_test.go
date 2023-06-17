package main

import (
	"reflect"
	"testing"
)

func Test_getBoardsToExtract(t *testing.T) {
	type args struct {
		str   string
		start int
		end   int
	}

	tests := []struct {
		name    string
		args    args
		want    [][2]int
		wantErr bool
	}{
		{
			name: "empty string",
			args: args{
				str:   "",
				start: 1,
				end:   10,
			},
			want: [][2]int{{1, 10}},
		},
		{
			name: "single Board",
			args: args{
				str:   "1",
				start: 1,
				end:   10,
			},
			want: [][2]int{{1, 1}},
		},
		{
			name: "single Board out of range",
			args: args{
				str:   "11",
				start: 1,
				end:   10,
			},
			wantErr: true,
		},
		{
			name: "multiple boards",
			args: args{
				str:   "1,2,3",
				start: 1,
				end:   10,
			},
			want: [][2]int{{1, 1}, {2, 2}, {3, 3}},
		},
		{
			name: "multiple boards out of range",
			args: args{
				str:   "1,2,11",
				start: 1,
				end:   10,
			},
			wantErr: true,
		},
		{
			name: "multiple boards out of order",
			args: args{
				str:   "1,5,3,2",
				start: 1,
				end:   10,
			},
			wantErr: true,
		},
		{
			name: "multiple boards out of range on the bottom",
			args: args{
				str:   "1,2,3,11",
				start: 3,
				end:   15,
			},
			wantErr: true,
		},
		{
			name: "multiple boards with range",
			args: args{
				str:   "1,2,4-7",
				start: 1,
				end:   10,
			},
			want: [][2]int{{1, 1}, {2, 2}, {4, 7}},
		},
		{
			name: "multiple boards with range out of range",
			args: args{
				str:   "1,2,4-11",
				start: 1,
				end:   10,
			},
			wantErr: true,
		},
		{
			name: "multiple boards with range out of order",
			args: args{
				str:   "1,2,7-4",
				start: 1,
				end:   10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBoardsToExtract(tt.args.str, tt.args.start, tt.args.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBoardsToExtract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBoardsToExtract() got = %v, want %v", got, tt.want)
			}
		})
	}
}
