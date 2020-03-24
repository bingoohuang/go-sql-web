package sqlweb

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bingoohuang/gou/htt"
)

type Merchant struct {
	MerchantName string
	MerchantId   string
	MerchantCode string
	HomeArea     string
	Classifier   string
}

func ServeSearchDb(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", htt.ContentTypeJSON)
	searchKey := strings.TrimSpace(req.FormValue("searchKey"))
	byTenant := strings.TrimSpace(req.FormValue("byTenant"))
	if searchKey == "" {
		http.Error(w, "searchKey required", 405)
		return
	}

	if searchKey == "trr" || !AppConf.MultiTenants {
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
	_, data, _, _, err, _ := executeQuery(searchSql, AppConf.DataSource, 0)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	searchResult := make([]Merchant, 0, len(data)+1)

	if len(data) == 0 {
		searchResult = append(searchResult,
			Merchant{MerchantName: "trr", MerchantId: "trr", MerchantCode: "trr", HomeArea: "south-center", Classifier: "trr"})
	} else {
		for _, v := range data {
			tid := v[2]
			if tid != "trr" {
				searchResult = append(searchResult,
					Merchant{MerchantName: v[1], MerchantId: tid, MerchantCode: v[3], HomeArea: v[4], Classifier: v[5]})
			}
		}
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
	sql := "SELECT MERCHANT_ID, DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME " +
		"FROM TR_F_DB WHERE MERCHANT_ID = '" + tid + "'"

	_, data, _, _, err, _ := executeQuery(sql, ds, 1)
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
	if tid == "trr" {
		return &Merchant{MerchantName: tid, MerchantId: tid, MerchantCode: tid, HomeArea: AppConf.TrrHomeArea, Classifier: tid}, nil
	}

	sql := "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
		"FROM TR_F_MERCHANT WHERE MERCHANT_ID = '" + tid + "'"

	return searchMerchantBySql(sql, 1)
}

func searchMerchantByTcode(tcode string) (*Merchant, error) {
	sql := "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
		"FROM TR_F_MERCHANT WHERE MERCHANT_CODE = '" + tcode + "'"

	return searchMerchantBySql(sql, 1)
}

func searchMerchantBySql(searchSql string, maxRows int) (*Merchant, error) {
	_, data, _, _, err, _ := executeQuery(searchSql, AppConf.DataSource, maxRows)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("merchant query result " + strconv.Itoa(len(data)) + " other than 1")
	}

	v := data[0]

	return &Merchant{MerchantName: v[1], MerchantId: v[2], MerchantCode: v[3], HomeArea: v[4], Classifier: v[5]}, nil
}
