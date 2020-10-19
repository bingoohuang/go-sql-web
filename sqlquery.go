package sqlweb

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bingoohuang/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

func selectDb(tid string) (string, string, error) {
	if tid == "" || tid == "trr" {
		_, rows, _, _, err, _ := executeQuery("SELECT DATABASE()", AppConf.DSN, 0)
		if err != nil {
			return "", "", err
		}

		return AppConf.DSN, rows[0][1], nil
	}

	if !strings.HasPrefix(tid, "sdb-") {
		return selectDbByTid(tid, AppConf.DSN)
	}

	dsnConfig, err := mysql.ParseDSN(AppConf.DSN)
	if err != nil {
		return "", "", err
	}

	dsnConfig.DBName = strings.TrimPrefix(tid, "sdb-")
	return dsnConfig.FormatDSN(), dsnConfig.DBName, nil
}

func selectDbByTid(tid string, ds string) (string, string, error) {
	queryDbSql := "SELECT DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME " +
		"FROM sqlweb WHERE MERCHANT_ID = '" + tid + "'"

	_, data, _, _, err, _ := executeQuery(queryDbSql, ds, 1)
	if err != nil {
		return "", "", err
	}

	if len(data) == 0 {
		return "", "", errors.New("no db found for tid:" + tid)
	} else if len(data) > 1 {
		log.Println("data", data)
		return "", "", errors.New("more than one db found")
	}

	row := data[0]

	// user:pass@tcp(127.0.0.1:3306)/db?charset=utf8
	return row[1] + ":" + row[2] + "@tcp(" + row[3] + ":" + row[4] + ")/" + row[5] +
		"?charset=utf8mb4,utf8&timeout=30s", row[5], nil
}

func executeQuery(querySql, dataSource string, max int) (
	[]string /*header*/, [][]string, /*data*/
	string /*executionTime*/, string /*costTime*/, error, string /* msg */) {
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, nil, "", "", err, ""
	}
	defer db.Close()

	return query(db, querySql, max)
}

func query(db *sql.DB, query string, maxRows int) ([]string, [][]string, string, string, error, string) {
	executionTime := time.Now().Format("2006-01-02 15:04:05.000")

	sqlResult := sqlx.ExecSQL(db, query, maxRows, "(null)")
	data := addRowsSeq(&sqlResult)

	msg := ""
	if !sqlResult.IsQuerySQL {
		msg = strconv.FormatInt(sqlResult.RowsAffected, 10) + " rows were affected"
	}

	return sqlResult.Headers, data, executionTime, sqlResult.CostTime.String(), sqlResult.Error, msg
}

func addRowsSeq(sqlResult *sqlx.ExecResult) [][]string {
	data := make([][]string, 0)
	if sqlResult.Rows != nil {
		for index, row := range sqlResult.Rows {
			r := make([]string, len(row)+1)
			r[0] = strconv.Itoa(index + 1)
			for j, cell := range row {
				r[j+1] = cell
			}
			data = append(data, r)
		}
	}
	return data
}
