package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/Jeffail/gabs"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

const geoLocation = "-37;145"

//get your token here: https://aqicn.org/data-platform/token/#/
const token = "YOUR_TOKEN_HERE"

func getAQI() (float64, error) {
	r, err := httpClient.Get(fmt.Sprintf("https://api.waqi.info/feed/geo:%s/?token=%s", geoLocation, token))
	if err != nil {
		return 0.0, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return 0.0, err
	}
	//log.Println(string(body))
	parsed, err := gabs.ParseJSON(body)
	if err != nil {
		return 0.0, err
	}

	value, ok := parsed.Path("status").Data().(string)
	if !ok {
		return 0.0, errors.New("unexpected status")
	}
	if value != "ok" {
		return 0.0, fmt.Errorf("status:%s", value)
	}
	aqi, ok := parsed.Path("data.aqi").Data().(float64)
	if !ok {
		return 0.0, errors.New("cannot find aqi info")
	}
	time, ok := parsed.Path("data.time.s").Data().(string)
	if ok {
		log.Println("Time:", time)
	}
	log.Println("AQI:", aqi)

	// get pollutant
	pollutant := ""
	pm25, ok := parsed.Path("data.iaqi.pm25.v").Data().(float64)
	if ok {
		pollutant += fmt.Sprintf("[PM 2.5:%v] ", pm25)
	}
	pm10, ok := parsed.Path("data.iaqi.pm10.v").Data().(float64)
	if ok {
		pollutant += fmt.Sprintf("[PM 10:%v] ", pm10)
	}
	o3, ok := parsed.Path("data.iaqi.o3.v").Data().(float64)
	if ok {
		pollutant += fmt.Sprintf("[O3:%v] ", o3)
	}
	no2, ok := parsed.Path("data.iaqi.no2.v").Data().(float64)
	if ok {
		pollutant += fmt.Sprintf("[NO2:%v] ", no2)
	}
	so2, ok := parsed.Path("data.iaqi.so2.v").Data().(float64)
	if ok {
		pollutant += fmt.Sprintf("[SO2:%v] ", so2)
	}
	co, ok := parsed.Path("data.iaqi.co.v").Data().(float64)
	if ok {
		pollutant += fmt.Sprintf("[CO:%v] ", co)
	}
	log.Println("Pollutant:", pollutant)

	// get detailed weather
	weather := ""
	dew, ok := parsed.Path("data.iaqi.dew.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Dew:%v] ", dew)
	}
	relativeHumidity, ok := parsed.Path("data.iaqi.h.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Relative Humidity:%v] ", relativeHumidity)
	}
	precipitation, ok := parsed.Path("data.iaqi.r.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Precipitation:%v] ", precipitation)
	}
	wind, ok := parsed.Path("data.iaqi.w.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Wind:%v] ", wind)
	}
	windGust, ok := parsed.Path("data.iaqi.wg.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Wind Gust:%v] ", windGust)
	}
	temperature, ok := parsed.Path("data.iaqi.t.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Temperature:%v] ", temperature)
	}
	pressure, ok := parsed.Path("data.iaqi.p.v").Data().(float64)
	if ok {
		weather += fmt.Sprintf("[Pressure:%v] ", pressure)
	}
	log.Println("Weather:", weather)
	return aqi, nil
}
