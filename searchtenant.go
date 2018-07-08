package main

import (
	"encoding/json"
	"errors"
	"github.com/bingoohuang/go-utils"
	"net/http"
	"strings"
)

type Merchant struct {
	MerchantName string
	MerchantId   string
	MerchantCode string
	HomeArea     string
	Classifier   string
}

func serveSearchDb(w http.ResponseWriter, req *http.Request) {
	go_utils.HeadContentTypeJson(w)
	searchKey := strings.TrimSpace(req.FormValue("searchKey"))
	byTenant := strings.TrimSpace(req.FormValue("byTenant"))
	if searchKey == "" {
		http.Error(w, "searchKey required", 405)
		return
	}

	if searchKey == "trr" || !multiTenants {
		var searchResult [1]Merchant
		searchResult[0] = Merchant{
			MerchantName: "trr",
			MerchantId:   "trr",
			MerchantCode: "trr",
			HomeArea:     "south-center",
			Classifier:   "trr"}
		json.NewEncoder(w).Encode(searchResult)
		return
	}

	searchSql := ""
	if byTenant == "true" {
		searchSql = "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
			"FROM TR_F_MERCHANT WHERE MERCHANT_ID = '" + searchKey + "' OR MERCHANT_CODE = '" + searchKey + "'"
	} else {
		searchSql = "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
			"FROM TR_F_MERCHANT WHERE MERCHANT_ID = '" + searchKey +
			"' OR MERCHANT_CODE = '" + searchKey + "' OR MERCHANT_NAME LIKE '%" + searchKey + "%'"
	}
	_, data, _, _, err, _ := executeQuery(searchSql, gDatasource, maxRows)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	searchResult := make([]Merchant, len(data))
	for i, v := range data {
		searchResult[i] = Merchant{
			MerchantName: v[1], MerchantId: v[2], MerchantCode: v[3], HomeArea: v[4], Classifier: v[5]}
	}

	json.NewEncoder(w).Encode(searchResult)
}

type MerchantDb struct {
	MerchantId string
	Username   string
	Password   string
	Host       string
	Port       string
	Database   string
}

func searchMerchantDb(tid string, ds string) (*MerchantDb, error) {
	queryDbSql := "SELECT MERCHANT_ID, DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME " +
		"FROM TR_F_DB WHERE MERCHANT_ID = '" + tid + "'"

	_, data, _, _, err, _ := executeQuery(queryDbSql, ds, maxRows)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("none or more than one found for tid:" + tid)
	}
	v := data[0]

	return &MerchantDb{MerchantId: v[1], Username: v[2], Password: v[3], Host: v[4], Port: v[5], Database: v[6]}, nil
}

func searchMerchant(tid string) (*Merchant, error) {
	searchSql := "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
		"FROM TR_F_MERCHANT WHERE MERCHANT_ID = '" + tid + "'"

	return searchMerchantBySql(searchSql)
}

func searchMerchantByTcode(tcode string) (*Merchant, error) {
	searchSql := "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
		"FROM TR_F_MERCHANT WHERE MERCHANT_CODE = '" + tcode + "'"

	return searchMerchantBySql(searchSql)
}

func searchMerchantBySql(searchSql string) (*Merchant, error) {
	_, data, _, _, err, _ := executeQuery(searchSql, gDatasource, maxRows)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("merchant query result 0 or more than 1")
	}

	v := data[0]

	return &Merchant{MerchantName: v[1], MerchantId: v[2], MerchantCode: v[3], HomeArea: v[4], Classifier: v[5]}, nil
}
