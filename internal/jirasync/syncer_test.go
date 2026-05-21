package jirasync

import "testing"

func TestShouldSkipReleaseNotes(t *testing.T) {
	tests := []struct {
		name        string
		fields      map[string]interface{}
		typeFieldID string
		want        bool
	}{
		{
			name:        "type field not configured",
			fields:      map[string]interface{}{"cf_10785": map[string]interface{}{"value": releaseNotesNotRequired}},
			typeFieldID: "",
			want:        false,
		},
		{
			name:        "release notes not required",
			fields:      map[string]interface{}{"cf_10785": map[string]interface{}{"value": releaseNotesNotRequired}},
			typeFieldID: "cf_10785",
			want:        true,
		},
		{
			name:        "release notes required",
			fields:      map[string]interface{}{"cf_10785": map[string]interface{}{"value": "Release Notes Required"}},
			typeFieldID: "cf_10785",
			want:        false,
		},
		{
			name:        "type field is nil",
			fields:      map[string]interface{}{"cf_10785": nil},
			typeFieldID: "cf_10785",
			want:        false,
		},
		{
			name:        "type field is wrong shape (string instead of map)",
			fields:      map[string]interface{}{"cf_10785": "some string"},
			typeFieldID: "cf_10785",
			want:        false,
		},
		{
			name:        "type field map has no value key",
			fields:      map[string]interface{}{"cf_10785": map[string]interface{}{"id": "123"}},
			typeFieldID: "cf_10785",
			want:        false,
		},
		{
			name:        "type field not present in fields map",
			fields:      map[string]interface{}{},
			typeFieldID: "cf_10785",
			want:        false,
		},
		{
			name:        "nil fields map",
			fields:      nil,
			typeFieldID: "cf_10785",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipReleaseNotes(tt.fields, tt.typeFieldID)
			if got != tt.want {
				t.Errorf("shouldSkipReleaseNotes() = %v, want %v", got, tt.want)
			}
		})
	}
}
