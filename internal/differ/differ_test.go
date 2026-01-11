package differ

import (
	"testing"

	wetwire "github.com/lex00/wetwire-aws-go"
)

func TestCompare(t *testing.T) {
	t1 := &wetwire.Template{
		Resources: map[string]wetwire.ResourceDef{
			"Bucket1": {Type: "AWS::S3::Bucket", Properties: map[string]any{"BucketName": "bucket1"}},
			"Bucket2": {Type: "AWS::S3::Bucket", Properties: map[string]any{"BucketName": "bucket2"}},
		},
	}

	t2 := &wetwire.Template{
		Resources: map[string]wetwire.ResourceDef{
			"Bucket1": {Type: "AWS::S3::Bucket", Properties: map[string]any{"BucketName": "bucket1-modified"}},
			"Bucket3": {Type: "AWS::S3::Bucket", Properties: map[string]any{"BucketName": "bucket3"}},
		},
	}

	result, err := Compare(t1, t2, Options{})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Bucket2 was removed
	if len(result.Diff.Removed) != 1 {
		t.Errorf("Removed = %d, want 1", len(result.Diff.Removed))
	} else if result.Diff.Removed[0].Resource != "Bucket2" {
		t.Errorf("Removed[0].Resource = %s, want Bucket2", result.Diff.Removed[0].Resource)
	}

	// Bucket3 was added
	if len(result.Diff.Added) != 1 {
		t.Errorf("Added = %d, want 1", len(result.Diff.Added))
	} else if result.Diff.Added[0].Resource != "Bucket3" {
		t.Errorf("Added[0].Resource = %s, want Bucket3", result.Diff.Added[0].Resource)
	}

	// Bucket1 was modified
	if len(result.Diff.Modified) != 1 {
		t.Errorf("Modified = %d, want 1", len(result.Diff.Modified))
	} else if result.Diff.Modified[0].Resource != "Bucket1" {
		t.Errorf("Modified[0].Resource = %s, want Bucket1", result.Diff.Modified[0].Resource)
	}

	// Summary
	if result.Summary.Total != 3 {
		t.Errorf("Summary.Total = %d, want 3", result.Summary.Total)
	}
}

func TestCompareIdentical(t *testing.T) {
	template := &wetwire.Template{
		Resources: map[string]wetwire.ResourceDef{
			"Bucket": {Type: "AWS::S3::Bucket", Properties: map[string]any{"BucketName": "test"}},
		},
	}

	result, err := Compare(template, template, Options{})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("Summary.Total = %d, want 0 for identical templates", result.Summary.Total)
	}
}

func TestCompareEmpty(t *testing.T) {
	t1 := &wetwire.Template{Resources: map[string]wetwire.ResourceDef{}}
	t2 := &wetwire.Template{Resources: map[string]wetwire.ResourceDef{}}

	result, err := Compare(t1, t2, Options{})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("Summary.Total = %d, want 0", result.Summary.Total)
	}
}

func TestCompareTypeChange(t *testing.T) {
	t1 := &wetwire.Template{
		Resources: map[string]wetwire.ResourceDef{
			"Resource1": {Type: "AWS::S3::Bucket"},
		},
	}

	t2 := &wetwire.Template{
		Resources: map[string]wetwire.ResourceDef{
			"Resource1": {Type: "AWS::S3::AccessPoint"},
		},
	}

	result, err := Compare(t1, t2, Options{})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if len(result.Diff.Modified) != 1 {
		t.Fatalf("Modified = %d, want 1", len(result.Diff.Modified))
	}

	found := false
	for _, change := range result.Diff.Modified[0].Changes {
		if change == "Type changed: AWS::S3::Bucket → AWS::S3::AccessPoint" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected type change to be detected")
	}
}

func TestCompareProperties(t *testing.T) {
	tests := []struct {
		name    string
		props1  map[string]any
		props2  map[string]any
		wantLen int
	}{
		{
			name:    "identical",
			props1:  map[string]any{"Key": "value"},
			props2:  map[string]any{"Key": "value"},
			wantLen: 0,
		},
		{
			name:    "added property",
			props1:  map[string]any{},
			props2:  map[string]any{"Key": "value"},
			wantLen: 1,
		},
		{
			name:    "removed property",
			props1:  map[string]any{"Key": "value"},
			props2:  map[string]any{},
			wantLen: 1,
		},
		{
			name:    "modified property",
			props1:  map[string]any{"Key": "value1"},
			props2:  map[string]any{"Key": "value2"},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := compareProperties("", tt.props1, tt.props2, Options{})
			if len(changes) != tt.wantLen {
				t.Errorf("compareProperties() returned %d changes, want %d", len(changes), tt.wantLen)
			}
		})
	}
}

func TestEqualStringSlices(t *testing.T) {
	tests := []struct {
		a, b []string
		want bool
	}{
		{nil, nil, true},
		{[]string{}, []string{}, true},
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"a"}, []string{"b"}, false},
		{[]string{"a"}, []string{"a", "b"}, false},
	}

	for _, tt := range tests {
		got := equalStringSlices(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("equalStringSlices(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
