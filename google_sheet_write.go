package main

import (
	"io/ioutil"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"gopkg.in/Iwark/spreadsheet.v2"
)

// modify to whatever sheet you want to write
const SheetId = "15Zyy0_swv2YazuL9UdZ4YYkPfaIwTpPNtPHLAlsLtcY"

var nextRow int = 1

func InitGoogleSheet(serviceAccountJson string) (*spreadsheet.Sheet, error) {
	data, err := ioutil.ReadFile(serviceAccountJson)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
	if err != nil {
		return nil, err
	}
	client := conf.Client(context.TODO())

	service := spreadsheet.NewServiceWithClient(client)
	spreadsheet, err := service.FetchSpreadsheet(SheetId)
	if err != nil {
		return nil, err
	}
	sheet, err := spreadsheet.SheetByIndex(0)
	if err != nil {
		return nil, err
	}
	// find next row to write
	for i, row := range sheet.Rows {
		if len(row) >= 1 && row[0].Value == "" {
			nextRow = i
			break
		}
		nextRow = i + 1
	}
	return sheet, nil
}

func WriteRowToSheet(sheet *spreadsheet.Sheet, rowContent []string) error {
	//fmt.Printf("Write to next row:%v\n", nextRow)
	// add a row
	for i, s := range rowContent {
		sheet.Update(nextRow, i, s)
	}
	nextRow += 1
	return sheet.Synchronize()
}

func test() {
	sheet, err := InitGoogleSheet("client_secret.json")
	if err != nil {
		panic(err.Error())
	}
	WriteRowToSheet(sheet, []string{"data1", "data2", "data3"})
}
