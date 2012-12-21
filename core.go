package GoGrooveShark

// Some of the responses can be condensed down to use embedded structs when
// https://codereview.appspot.com/6460044 - is released in stable go1.1

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	API_HOST = "api.grooveshark.com/ws3.php"
)

type apiRequestPayload struct {
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters"`
	Header     map[string]interface{} `json:"header"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ApiErrorResponse struct {
	Errors []apiError `json:"errors"`
}

func (err *ApiErrorResponse) Error() string {
	strRet := ""

	// Return all of the errors
	for _, element := range err.Errors {
		strRet += element.Message + "\n"
	}

	return strRet
}

type apiResponse struct {
	HttpCode int
	Body     string
}

func (resp *apiResponse) getError() error {
	errResp := &ApiErrorResponse{}

	// If we could unmarshal into the error type, we know the result was an error.
	// GrooveShark claim that this is a RESTFul api, but the http response code is always 200(ok)
	// which makes working out if the response is ok a little harder
	err := json.Unmarshal([]byte(resp.Body), errResp)

	foundError := false
	for _, element := range errResp.Errors {
		if element.Code != 0 {
			foundError = true
			break
		}
	}

	if err == nil && foundError {
		return errResp
	}

	return nil
}

func (apiResp *apiResponse) unmarshal(resp interface{}) {
	r := responseUnmarshaller{}
	r.Result.resp = resp
	json.Unmarshal([]byte(apiResp.Body), &r)
}

type SongInfo struct {
	SongID                int
	SongName              string
	ArtistID              int
	ArtistName            string
	AlbumID               int
	AlbumName             string
	CoverArtFileName      string
	Popularity            string
	IsLowBitrateAvailable bool
	IsVerified            bool
	Flags                 int
}

type Playlist struct {
	PlaylistName        string
	TSModified          int
	UserID              int
	PlaylistDescription string
	CoverArtFilename    string
	Songs               []SongInfo
}

type EmptyResponse struct {
	Success bool `json:"success"`
}

type responseUnmarshaller struct {
	Header map[string]string  `json:"header"`
	Result resultUnmarshaller `json:"result"`
}

type resultUnmarshaller struct {
	resp interface{}
}

func (r *resultUnmarshaller) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, r.resp)
}

type GrooveShark struct {
	secretKey, publicKey string
	sessionId            string
}

func NewGrooveShark(key, secret string) *GrooveShark {
	return &GrooveShark{
		secretKey: secret,
		publicKey: key,
	}
}

func (gs *GrooveShark) GetPlaylist(playlistID string, limit *int) (*Playlist, error) {
	args := make(map[string]interface{})
	args["playlistID"] = playlistID

	if limit != nil {
		args["limit"] = limit
	}

	resp, err := gs.apiCall("getPlaylist", args)
	if err != nil {
		return nil, err
	}

	err = resp.getError()
	if err != nil {
		return nil, err
	}

	playlist := new(Playlist)
	resp.unmarshal(playlist)
	return playlist, nil
}

type SessionResponse struct {
	Success   bool   `json:"success"`
	SessionId string `json:"sessionID"`
}

func (gs *GrooveShark) StartSession() (*string, error) {
	resp, err := gs.apiCallSecure("startSession", nil)

	if err != nil {
		return nil, err
	}

	err = resp.getError()
	if err != nil {
		return nil, err
	}

	session := SessionResponse{}
	resp.unmarshal(&session)

	if !session.Success {
		return nil, errors.New("Error starting session")
	}

	// Set the property
	gs.sessionId = session.SessionId

	return &gs.sessionId, nil
}

type User struct {
	UserID     int
	Email      string
	FName      string
	LName      string
	IsPlus     bool
	IsAnywhere bool
	IsPremium  bool
	Success    bool `json:"success"`
}

func (gs *GrooveShark) Authenticate(login, password string) (*User, error) {
	md5Hasher := md5.New()
	md5Hasher.Write([]byte(password))
	passwordHash := fmt.Sprintf("%x", md5Hasher.Sum(nil))

	// Start a session if we haven't got one currently
	if len(gs.sessionId) == 0 {
		_, err := gs.StartSession()
		if err != nil {
			return nil, err
		}
	}

	args := make(map[string]interface{})
	args["login"] = login
	args["password"] = passwordHash
	resp, err := gs.apiCallSecure("authenticate", args)

	if err != nil {
		return nil, err
	}

	err = resp.getError()
	if err != nil {
		return nil, err
	}

	user := new(User)
	resp.unmarshal(user)

	if !user.Success {
		return nil, errors.New("Error authenticating user")
	}

	return user, nil
}

func (gs *GrooveShark) Logout() error {
	resp, err := gs.apiCallSecure("logout", nil)

	if err != nil {
		return err
	}

	err = resp.getError()
	if err != nil {
		return err
	}

	return nil
}

func (gs *GrooveShark) PingService() (*string, error) {
	resp, err := gs.apiCallSecure("pingService", nil)

	if err != nil {
		return nil, err
	}

	err = resp.getError()
	if err != nil {
		return nil, err
	}

	helloWorld := new(string)
	resp.unmarshal(helloWorld)

	return helloWorld, nil
}

func (gs *GrooveShark) AddUserFavoriteSong(songId int) error {
	args := make(map[string]interface{})
	args["songID"] = songId
	resp, err := gs.apiCall("addUserFavoriteSong", args)

	if err != nil {
		return err
	}

	err = resp.getError()
	if err != nil {
		return err
	}

	respData := EmptyResponse{}
	resp.unmarshal(&respData)

	if !respData.Success {
		return errors.New("Error adding favorite song")
	}

	return nil
}

type PlaylistResponse struct {
	Success             bool `json:"success"`
	PlaylistsTSModified int  `json:"playlistsTSModified"`
	PlaylistID          int  `json:"playlistID"`
}

func (gs *GrooveShark) CreatePlaylist(playlistName string, songIds []int) (*PlaylistResponse, error) {
	args := make(map[string]interface{})
	args["name"] = playlistName
	args["songIDs"] = songIds
	resp, err := gs.apiCall("createPlaylist", args)

	if err != nil {
		return nil, err
	}

	err = resp.getError()
	if err != nil {
		return nil, err
	}

	respData := new(PlaylistResponse)
	resp.unmarshal(respData)

	if !respData.Success {
		return nil, errors.New("Error creating playlist")
	}

	return respData, nil
}

type DeletePlaylistResponse struct {
	Success             bool `json:"success"`
	PlaylistsTSModified int  `json:"playlistsTSModified"`
}

func (gs *GrooveShark) DeletePlaylist(playlistId int) (*DeletePlaylistResponse, error) {
	args := make(map[string]interface{})
	args["playlistID"] = playlistId
	resp, err := gs.apiCall("deletePlaylist", args)

	if err != nil {
		return nil, err
	}

	err = resp.getError()
	if err != nil {
		return nil, err
	}

	respData := new(DeletePlaylistResponse)
	resp.unmarshal(respData)

	if !respData.Success {
		return nil, errors.New("Error deleting playlist")
	}

	return respData, nil
}

func (gs *GrooveShark) apiCall(methodName string, args map[string]interface{}) (*apiResponse, error) {
	return gs.apiCallEx(methodName, args, false)
}

func (gs *GrooveShark) apiCallSecure(methodName string, args map[string]interface{}) (*apiResponse, error) {
	return gs.apiCallEx(methodName, args, true)
}

func (gs *GrooveShark) apiCallEx(methodName string, args map[string]interface{}, secure bool) (*apiResponse, error) {
	// Setup the request payload that will be sent over the wire
	req := apiRequestPayload{Method: methodName}
	req.Parameters = args
	req.Header = make(map[string]interface{})
	req.Header["wsKey"] = gs.publicKey

	if len(gs.sessionId) > 0 {
		req.Header["sessionID"] = gs.sessionId
	}

	postData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	payload := fmt.Sprintf("%s", postData)
	signature := createSignature(payload, gs.secretKey)

	proto := "http"
	if secure {
		proto = "https"
	}

	// Build the URL
	queryStr := "?sig=" + signature
	url := proto + "://" + API_HOST + queryStr

	bodyBuffer := new(bytes.Buffer)
	bodyBuffer.Write([]byte(payload))
	httpReq, err := http.NewRequest("POST", url, bodyBuffer)
	if err != nil {
		return nil, err
	}

	// Set the headers
	httpReq.Header.Set("Content-Type", "text/plain; charset=UTF-8")
	httpReq.Header.Set("User-Agent", "GoGrooveShark-Go")

	// Send off the request
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	response := new(apiResponse)
	response.Body = fmt.Sprintf("%s", body)
	response.HttpCode = resp.StatusCode

	return response, nil
}

// Creates a signature that is required by all API calls to GrooveShark
func createSignature(query string, privateKey string) string {
	h := hmac.New(md5.New, []byte(privateKey))
	h.Write([]byte(query))
	return fmt.Sprintf("%x", h.Sum(nil))
}
