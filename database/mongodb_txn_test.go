package database

import "testing"

func TestMongoSupportsTransactionsFromHello(t *testing.T) {
	tests := []struct {
		name    string
		setName string
		msg     string
		want    bool
	}{
		{
			name:    "replica set supports transactions",
			setName: "rs0",
			msg:     "",
			want:    true,
		},
		{
			name:    "mongos supports transactions",
			setName: "",
			msg:     "isdbgrid",
			want:    true,
		},
		{
			name:    "standalone does not support transactions",
			setName: "",
			msg:     "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mongoSupportsTransactionsFromHello(tt.setName, tt.msg)
			if got != tt.want {
				t.Fatalf("mongoSupportsTransactionsFromHello(%q, %q) = %v, want %v", tt.setName, tt.msg, got, tt.want)
			}
		})
	}
}
