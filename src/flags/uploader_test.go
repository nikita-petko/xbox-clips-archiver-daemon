package flags

import (
	"flag"
	"testing"
)

func TestUploaderGroupingFlags(t *testing.T) {
	if flag.Lookup("uploader-group-by-user-mode").DefValue != "none" ||
		flag.Lookup("uploader-group-by-title-mode").DefValue != "name" {
		t.Fatal("unexpected grouping defaults")
	}

	oldUser, oldTitle := *UploaderGroupByUserMode, *UploaderGroupByTitleMode
	t.Cleanup(func() {
		*UploaderGroupByUserMode, *UploaderGroupByTitleMode = oldUser, oldTitle
	})
	t.Setenv("UPLOADER_GROUP_BY_USER_MODE", "gamertag")
	t.Setenv("UPLOADER_GROUP_BY_TITLE_MODE", "id")
	applyEnvironmentVariableFlags()
	if *UploaderGroupByUserMode != "gamertag" || *UploaderGroupByTitleMode != "id" {
		t.Fatalf("environment values: %q, %q", *UploaderGroupByUserMode, *UploaderGroupByTitleMode)
	}
	if err := ValidateUploaderGrouping(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"", "ID", "bad"} {
		*UploaderGroupByUserMode = mode
		if err := ValidateUploaderGrouping(); err == nil {
			t.Errorf("expected invalid user mode %q to fail", mode)
		}
		*UploaderGroupByUserMode, *UploaderGroupByTitleMode = "none", mode
		if err := ValidateUploaderGrouping(); err == nil {
			t.Errorf("expected invalid title mode %q to fail", mode)
		}
	}
}

func TestUploadMediaFileNameMode(t *testing.T) {
	if flag.Lookup("upload-media-file-name-mode").DefValue != "captured" {
		t.Fatal("unexpected filename mode default")
	}

	old := *UploadMediaFileNameMode
	t.Cleanup(func() { *UploadMediaFileNameMode = old })
	t.Setenv("UPLOAD_MEDIA_FILE_NAME_MODE", "id")
	applyEnvironmentVariableFlags()
	if *UploadMediaFileNameMode != "id" {
		t.Fatalf("environment value = %q", *UploadMediaFileNameMode)
	}
	for _, tc := range []struct {
		mode  string
		valid bool
	}{{"id", true}, {"captured", true}, {"", false}, {"CAPTURED", false}, {"bad", false}} {
		*UploadMediaFileNameMode = tc.mode
		if err := ValidateUploadMediaFileNameMode(); (err == nil) != tc.valid {
			t.Errorf("mode %q: error = %v", tc.mode, err)
		}
	}

}

func TestVerifyUploadIntegrityFlag(t *testing.T) {
	if flag.Lookup("should-verify-upload-integrity").DefValue != "true" {
		t.Fatal("integrity verification should default to true")
	}
	old := *ShouldVerifyUploadIntegrity
	t.Cleanup(func() { *ShouldVerifyUploadIntegrity = old })
	t.Setenv("SHOULD_VERIFY_UPLOAD_INTEGRITY", "false")
	applyEnvironmentVariableFlags()
	if *ShouldVerifyUploadIntegrity {
		t.Fatal("expected environment variable to disable verification")
	}
	t.Setenv("SHOULD_VERIFY_UPLOAD_INTEGRITY", "invalid")
	defer func() {
		if recover() == nil {
			t.Error("expected invalid environment value to fail")
		}
	}()
	applyEnvironmentVariableFlags()
}

func TestAlertURLBase(t *testing.T) {
	if flag.Lookup("alert-url-base").DefValue != "" {
		t.Fatal("alert URL base should be unset by default")
	}
	old := *AlertURLBase
	t.Cleanup(func() { *AlertURLBase = old })
	t.Setenv("ALERT_URL_BASE", "https://cdn.example.com/assets")
	applyEnvironmentVariableFlags()
	if *AlertURLBase != "https://cdn.example.com/assets" {
		t.Fatalf("environment value = %q", *AlertURLBase)
	}
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"", true}, {"https://cdn.example.com/assets/", true},
		{"https://cdn.example.com/a%2Fb", true},
		{"http://cdn.example.com", false}, {"cdn.example.com", false},
		{"https:///missing-host", false}, {"https://user:pass@cdn.example.com", false},
		{"https://cdn.example.com/?q=1", false}, {"https://cdn.example.com?", false},
		{"https://cdn.example.com/#fragment", false}, {"https://cdn.example.com/#", false},
		{"https://cdn.example.com/%zz", false},
	} {
		*AlertURLBase = tc.value
		_, err := ParseAlertURLBase()
		if (err == nil) != tc.valid {
			t.Errorf("base %q: error = %v", tc.value, err)
		}
	}
}
