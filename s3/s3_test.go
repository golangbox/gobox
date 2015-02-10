package s3

import "testing"

const (
	validS3Key = "test"
)

func TestTestKeyExistence(t *testing.T) {
	var exists bool
	var err error

	exists, err = TestKeyExistence(validS3Key)
	if exists != true {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
	exists, err = TestKeyExistence("notvalid")
	if exists == true {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateSignedUrl(t *testing.T) {

	url, err := GenerateSignedUrl(validS3Key)
	_ = url
	if err != nil {
		t.Error(err)
	}
	// exists, err = TestKeyExistence("notvalid")
	// if exists == true {
	// 	t.Fail()
	// }
	// if err != nil {
	// 	t.Error(err)
	// }
}

func TestFileUpload(t *testing.T) {
	file := []byte("file thing")
	err := UploadFile("test2", file)
	if err != nil {
		t.Error(err)
	}
}
