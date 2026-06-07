package manifest

import "testing"

func TestEntity_ValidateKind(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		wantErr bool
	}{
		{"valid policy", "policy", false},
		{"valid procedure", "procedure", false},
		{"valid capability", "capability", false},
		{"valid guard", "guard", false},
		{"valid worker", "worker", false},
		{"valid bundle", "bundle", false},
		{"valid target", "target", false},
		{"valid skip", "skip", false},
		{"invalid kind", "unknown", true},
		{"empty kind", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Entity{Kind: tt.kind}
			err := e.ValidateKind()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryDoc_ValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"valid current", "current", false},
		{"valid target", "target", false},
		{"valid transitional", "transitional", false},
		{"valid question", "question", false},
		{"valid deprecated", "deprecated", false},
		{"invalid status", "invalid-status", true},
		{"empty status", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := MemoryDoc{Status: tt.status}
			err := d.ValidateStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "with line number",
			err:  ValidationError{File: "test.yaml", Line: 10, Message: "missing field"},
			want: "test.yaml:10: ERROR: missing field",
		},
		{
			name: "without line number",
			err:  ValidationError{File: "test.yaml", Line: 0, Message: "file error"},
			want: "test.yaml: ERROR: file error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
