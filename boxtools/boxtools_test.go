package boxtools

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

const (
	email    = "max.t.mcdonnell@gmail.com"
	password = "password"
)

var db gorm.DB

func init() {
	var err error
	db, err = gorm.Open("postgres", "dbname=goboxtest sslmode=disable")
	db.DropTableIfExists(&User{})
	db.DropTableIfExists(&File{})
	db.DropTableIfExists(&JournalEntry{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&File{})
	db.AutoMigrate(&JournalEntry{})
	if err != nil {
		fmt.Println(err)
	}
}

func TestDBFindFileReturnsErrorWhenNotFound(t *testing.T) {
	testMeta := Meta{
		0,
		"commit",
		"what is name for?",
		"854eaaae4dc9ad3eef2fd235587d9d6e71c168e9b7b6624f41aa650fb87d0a87",
		8014,
		"./client.go",
		time.Now(),
		time.Now(),
		time.Now(),
	}
	_, err := DBFindFile(testMeta, db)
	if err == nil {
		t.Fail()
	}
	fmt.Println("passed: TestDBFindFileReturnsErrorWhenNotFound")

}

func TestDBFindFileWorksWhenFileExists(t *testing.T) {
	testMeta := Meta{
		0,
		"commit",
		"what is name for?",
		"854eaaae4dc9ad3eef2fd235587d9d6e71c168e9b7b6624f41aa650fb87d0a87",
		8014,
		"./client.go",
		time.Now(),
		time.Now(),
		time.Now(),
	}
	_, err := DBCreateFileFromMetaStruct(testMeta, db)
	if err != nil {
		t.Log("File was not created successfully")
		t.FailNow()
	}

	_, err1 := DBFindFile(testMeta, db)
	if err1 != nil {
		t.Log("File was created successfully but not found by DBFindFile")
		t.FailNow()
	}
	fmt.Println("passed: TestDBFindFileWorksWhenFileExists")
}

func TestDBCreateFileFromMetaStruct(t *testing.T) {
	testMeta := Meta{
		0,
		"commit",
		"what is name for?",
		"854eaaae4dc9ad3eef2fd235587d9d6e71c168e9b7b6624f41aa650fb87d0a87",
		8014,
		"./client.go",
		time.Now(),
		time.Now(),
		time.Now(),
	}
	_, err := ConvertMetaStructToFileStruct(testMeta)
	if err != nil {
		t.Log("Conversion from Meta to File did not work properly")
		t.FailNow()
	}
	fmt.Println("passed: TestConvertMetaStructToFileStruct")

}

func TestCreateJournalEntryFromMeta(t *testing.T) {
	testMeta := Meta{
		0,
		"commit",
		"what is name for?",
		"854eaaae4dc9ad3eef2fd235587d9d6e71c168e9b7b6624f41aa650fb87d0a87",
		8014,
		"./client.go",
		time.Now(),
		time.Now(),
		time.Now(),
	}
	_, err := CreateJournalEntryFromMeta(testMeta, db)
	if err != nil {
		t.Log("CreateJournalEntry raised an error on creation")
	}

}

func TestDBCreateJournalEntry(t *testing.T) {
	_, err := DBCreateJournalEntry("commit", 1, db)
	if err != nil {
		t.Log("Couldn't properly create a JournalEntry in the database")
		t.FailNow()
	}
}

func TestUserCreation(t *testing.T) {
	user, err := NewUser(email, password, db)
	if err != nil {
		t.Error(err)
	}
	if user.Email != email {
		t.Fail()
	}
	if user.HashedPassword == "" {
		t.Fail()
	}

	fmt.Println("passed: TestUserCreation")
}

func TestPasswordValidation(t *testing.T) {
	user, err := ValidateUserPassword(email, password, db)
	if err != nil {
		t.Error(err)
	}
	if user.Email != email {
		t.Fail()
	}
	// clean up created user
	db.Where("email = ?", email).Delete(User{})
	fmt.Println("passed: TestPasswordValidation")
}

func TestJsonMetaConversion(t *testing.T) {
	// if testing.Short() {
	//     t.Skip("skipping test in short mode.")
	// }
	testJsonString := "{\"Task\":\"upload\",\"File\":{\"Name\":\"client.go\",\"Hash\":\"7f41b055dfd190ab2e7b940171c50689866cd42d318460ca3637ddb27cfad926\",\"Size\":7838,\"Path\":\"./client.go\",\"Modified\":\"2015-02-02T18:14:48-05:00\"}}"
	testMeta := Meta{
		1,
		"upload",
		"client.go",
		"7f41b055dfd190ab2e7b940171c50689866cd42d318460ca3637ddb27cfad926",
		7838,
		"./client.go",
		time.Now(),
		time.Now(),
		time.Now(),
	}
	resultMeta, err := ConvertJsonStringToMetaStruct(testJsonString)
	if err != nil {
		t.Error(err)
	}
	if resultMeta.Task == testMeta.Task &&
		resultMeta.Name == testMeta.Name &&
		resultMeta.Hash == testMeta.Hash &&
		resultMeta.Size == testMeta.Size &&
		resultMeta.Path == testMeta.Path {
	} else {
		t.Fail()
	}
	fmt.Println("passed: TestJsonMetaConversion")
}

func TestRemoveRedundancy(t *testing.T) {
	// only test files in the same directory
	// nothing too complex other than removing matching
	// upload/delete pairs

	testJsons := `{"Task":"upload","File":{"Name":"client.go","Hash":"7f41b055dfd190ab2e7b940171c50689866cd42d318460ca3637ddb27cfad926","Size":7838,"Path":"./client.go","Modified":"2015-02-02T18:14:48-05:00"}}
{"Task":"delete","File":{"Name":"client.go","Hash":"854eaaae4dc9ad3eef2fd235587d9d6e71c168e9b7b6624f41aa650fb87d0a87","Size":8014,"Path":"./client.go","Modified":"2015-01-29T16:57:12-05:00"}}
{"Task":"upload","File":{"Name":"hi","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./hi","Modified":"2015-02-02T18:22:37-05:00"}}
{"Task":"delete","File":{"Name":"hi","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./hi","Modified":"2015-02-02T18:22:37-05:00"}}
{"Task":"upload","File":{"Name":"blah","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./blah","Modified":"2015-02-02T18:24:40-05:00"}}
{"Task":"upload","File":{"Name":"wheee","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./wheee","Modified":"2015-02-02T18:25:23-05:00"}}
{"Task":"delete","File":{"Name":"blah","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./blah","Modified":"2015-02-02T18:24:40-05:00"}}
{"Task":"upload","File":{"Name":"asdfasdfa","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./asdfasdfa","Modified":"2015-02-02T18:26:13-05:00"}}
{"Task":"delete","File":{"Name":"wheee","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./wheee","Modified":"2015-02-02T18:25:23-05:00"}}
{"Task":"upload","File":{"Name":"test","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./test","Modified":"2015-02-02T18:29:19-05:00"}}
{"Task":"upload","File":{"Name":"test","Hash":"9bcbbd9e1305636ccd1088094ba1f255e762f3c84c4f67308355dd4fa7890a6e","Size":89,"Path":"./test","Modified":"2015-02-02T18:29:30-05:00"}}
{"Task":"delete","File":{"Name":"test","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./test","Modified":"2015-02-02T18:29:19-05:00"}}
{"Task":"upload","File":{"Name":"adfasdfasd","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./adfasdfasd","Modified":"2015-02-02T18:30:17-05:00"}}
{"Task":"delete","File":{"Name":"asdfasdfa","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./asdfasdfa","Modified":"2015-02-02T18:26:13-05:00"}}
{"Task":"delete","File":{"Name":"adfasdfasd","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./adfasdfasd","Modified":"2015-02-02T18:30:17-05:00"}}
{"Task":"upload","File":{"Name":"asdfasdfasdkfksa","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./asdfasdfasdkfksa","Modified":"2015-02-02T18:36:59-05:00"}}
{"Task":"upload","File":{"Name":"asdfasd","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./asdfasd","Modified":"2015-02-03T10:53:26-05:00"}}`
	jsonSlice := strings.Split(testJsons, "\n")
	var metaSlice []Meta
	for _, jsonMetaString := range jsonSlice {
		metaFromJson, err := ConvertJsonStringToMetaStruct(jsonMetaString)
		if err != nil {
			t.Error(err)
		}
		metaSlice = append(metaSlice, metaFromJson)
	}
	simplifiedMetadata := RemoveRedundancyFromMetadata(metaSlice)

	// fmt.Println(simplifiedMetadata)
	if len(simplifiedMetadata) != 5 {
		t.Fail()
	}
	var computedResultJson string
	for _, metaStruct := range simplifiedMetadata {
		newString, err := ConvertMetaStructToJsonString(metaStruct)
		if err != nil {
			t.Error(err)
		}
		computedResultJson = computedResultJson + newString + "\n"
	}
	resultJson := `{"Task":"upload","File":{"Name":"client.go","Hash":"7f41b055dfd190ab2e7b940171c50689866cd42d318460ca3637ddb27cfad926","Size":7838,"Path":"./client.go","Modified":"2015-02-02T18:14:48-05:00"}}
{"Task":"delete","File":{"Name":"client.go","Hash":"854eaaae4dc9ad3eef2fd235587d9d6e71c168e9b7b6624f41aa650fb87d0a87","Size":8014,"Path":"./client.go","Modified":"2015-01-29T16:57:12-05:00"}}
{"Task":"upload","File":{"Name":"test","Hash":"9bcbbd9e1305636ccd1088094ba1f255e762f3c84c4f67308355dd4fa7890a6e","Size":89,"Path":"./test","Modified":"2015-02-02T18:29:30-05:00"}}
{"Task":"upload","File":{"Name":"asdfasdfasdkfksa","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./asdfasdfasdkfksa","Modified":"2015-02-02T18:36:59-05:00"}}
{"Task":"upload","File":{"Name":"asdfasd","Hash":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","Size":0,"Path":"./asdfasd","Modified":"2015-02-03T10:53:26-05:00"}}
`
	if computedResultJson != resultJson {
		t.Fail()
	}

	fmt.Println("passed: TestRemoveRedundancy")
}
