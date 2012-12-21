package GoGrooveShark

import (
	"fmt"
	"testing"
)

// Various things used during testing
var (
	GS_SAMPLE_SONG_ID       = 30717514
	GS_SAMPLE_ALBUM_ID      = 3857559
	GS_SAMPLE_PLAYLIST_NAME = "_TEST_PLAYLIST_"
)

func ExampleCreateSignature() {
	fmt.Println(createSignature("{'x': 'some json value'}", "1234567890abcdefghijklmnopqrstuv"))
	// Output: 9bdf0ebbb4cb2c945022670a16c6b5dc
}

func TestInitGrooveShark(t *testing.T) {
	gs := NewGrooveShark("pub", "secrets")

	if gs.publicKey != "pub" {
		t.Fail()
	}

	if gs.secretKey != "secrets" {
		t.Fail()
	}
}

func CreateGrooveShark() *GrooveShark {
	return NewGrooveShark(API_KEY, API_SECRET)
}

func TestGoodApiCall(t *testing.T) {
	out, err := CreateGrooveShark().apiCall("pingService", make(map[string]interface{}))
	if err != nil {
		t.Fatalf("Error param is set")
	}

	if out == nil {
		t.Fatalf("out param is not set")
	}

	if out.getError() != nil {
		t.Fatalf("Response is not ok")
	}
}

func TestBadApiCall(t *testing.T) {
	out, err := CreateGrooveShark().apiCall("pingServiceNOTAMETHOD", make(map[string]interface{}))
	if err != nil {
		t.Fatalf("Error param is set")
	}

	if out == nil {
		t.Fatalf("out param is not set")
	}

	if out.getError() == nil {
		t.Fatalf("Response is ok")
	}
}

func TestGetPlaylist(t *testing.T) {
	playlist, err := CreateGrooveShark().GetPlaylist("52262304", nil)
	if err != nil {
		t.Fatal(err)
	}

	if playlist.UserID == 0 {
		t.Fatal("Empty response")
	}
}

func TestStartSession(t *testing.T) {
	gs := CreateGrooveShark()
	sessionId, err := gs.StartSession()
	if err != nil {
		t.Fatal(err)
	}

	if sessionId == nil {
		t.Fatal("No session id returned")
	}

	if gs.sessionId != *sessionId {
		t.Fatal("GS session does not match.")
	}

	t.Logf("SessionID: %s", gs.sessionId)
}

func TestAuthenticate(t *testing.T) {
	gs := CreateGrooveShark()
	user, err := gs.Authenticate(GS_LOGIN, GS_PASSWORD)
	if err != nil {
		t.Fatal(err)
	}

	if user == nil {
		t.Fatal("No user returned")
	}

	t.Logf("%+v", user)
}

func TestAddUserFavoriteSong(t *testing.T) {
	gs := CreateGrooveShark()
	_, err := gs.Authenticate(GS_LOGIN, GS_PASSWORD)
	if err != nil {
		t.Fatal(err)
	}

	err = gs.AddUserFavoriteSong(GS_SAMPLE_SONG_ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPingService(t *testing.T) {
	gs := CreateGrooveShark()
	msg, err := gs.PingService()
	if err != nil {
		t.Fatal(err)
	}

	if msg == nil {
		t.Fatal("No hello world message returned")
	}

	t.Logf(*msg)
}

func TestCreatePlaylist(t *testing.T) {
	gs := CreateGrooveShark()

	_, err := gs.Authenticate(GS_LOGIN, GS_PASSWORD)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := gs.CreatePlaylist(GS_SAMPLE_PLAYLIST_NAME, []int{GS_SAMPLE_SONG_ID})
	if err != nil {
		t.Fatal(err)
	}

	if msg == nil {
		t.Fatal("No response returned")
	}

	if !msg.Success {
		t.Fatal("Error returned")
	}

	t.Logf("%+v", msg)
}

func TestDeletePlaylist(t *testing.T) {
	gs := CreateGrooveShark()

	_, err := gs.Authenticate(GS_LOGIN, GS_PASSWORD)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := gs.DeletePlaylist(80882182)
	if err != nil {
		t.Fatal(err)
	}

	if msg == nil {
		t.Fatal("No response returned")
	}

	if !msg.Success {
		t.Fatal("Error returned")
	}

	t.Logf("%+v", msg)
}
