package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// copied from https://developers.google.com/sheets/api/quickstart/go

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	user, err := user.Current()
	tokFile := ".google_sheet_token.json"
	if err != nil {
		log.Println(err)
	} else {
		tokFile = path.Join(user.HomeDir, tokFile)
	}
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// ref: https://gist.github.com/bati11/b2ba535f74c7bcb723a2ee46585814d8
func insertRow(sheetService *sheets.Service, spreadsheetID string, sheetID, startID, endID int64) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	req := sheets.Request{
		InsertDimension: &sheets.InsertDimensionRequest{
			InheritFromBefore: false,
			Range: &sheets.DimensionRange{
				Dimension:  "ROWS",
				StartIndex: startID,
				EndIndex:   endID,
				SheetId:    sheetID,
			},
		},
	}
	insertRowReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}
	return sheetService.Spreadsheets.BatchUpdate(spreadsheetID, &insertRowReq).Do()
}

// PrependRow will prepend a row on spreat sheets
func PrependRow(service *sheets.Service, spreadsheetID, rangeA1 string, row []interface{}) error {

	a1 := strings.Split(rangeA1, "!")
	if len(a1) != 2 {
		return errors.New("unable to parse A1 notation " + rangeA1)
	}
	sheetID, err := getSpreadsheet(service, spreadsheetID, a1[0])
	if err != nil {
		return err
	}

	_, err = insertRow(service, spreadsheetID, sheetID, 1, 2)
	if err != nil {
		log.Printf("sheet insert failed: %v", err)
		return err
	}

	valueRange := &sheets.ValueRange{
		MajorDimension: "ROWS",
		Values: [][]interface{}{
			row,
		},
	}
	_, err = service.Spreadsheets.Values.Update(spreadsheetID, rangeA1, valueRange).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Printf("sheet update failed: %v", err)
		return err
	}
	return nil
}

func sheetID(s *sheets.Spreadsheet, sheetName string) (int64, error) {
	for _, sheet := range s.Sheets {
		if sheet.Properties.Title == sheetName {
			return sheet.Properties.SheetID, nil
		}
	}
	return 0, errors.New("couldn't find sheet:" + sheetName)
}

// InitGoogleSheet initialise connection to google sheet service
func InitGoogleSheet(credentialFilename string) (*sheets.Service, error) {
	b, err := ioutil.ReadFile(credentialFilename)
	if err != nil {
		log.Fatalf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Printf("unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Printf("unable to retrieve Sheets client: %v", err)
		return nil, err
	}
	return srv, nil
}

func getSpreadsheet(srv *sheets.Service, id string, sheetName string) (int64, error) {
	spreadsheet, err := srv.Spreadsheets.Get(id).Do()
	if err != nil {
		log.Printf("unable to get sheet %v: %v", sheetName, err)
		return 0, err
	}

	sheetID, err := sheetID(spreadsheet, sheetName)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return sheetID, nil
}
