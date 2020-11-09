package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bingoohuang/gou/ran"
	"github.com/skratchdot/open-golang/open"

	"github.com/bingoohuang/gou/htt"

	"github.com/bingoohuang/sqlweb"
	"github.com/bingoohuang/statiq/fs"
	"github.com/gorilla/mux"

	_ "github.com/bingoohuang/sqlweb/statiq"
)

func init() {
	sqlweb.StaticFs, _ = fs.New()
	sqlweb.MustAsset = func(name string) []byte {
		return sqlweb.StaticFs.Files["/"+name].Data
	}

	sqlweb.AssetNames = make([]string, 0, len(sqlweb.StaticFs.Files))
	for k := range sqlweb.StaticFs.Files {
		sqlweb.AssetNames = append(sqlweb.AssetNames, k[1:])
	}

	sqlweb.AssetInfo = func(name string) (info os.FileInfo, e error) {
		f, err := sqlweb.StaticFs.Open("/" + name)
		if err != nil {
			return nil, err
		}

		return f.Stat()
	}

	sqlweb.InitConf()
}

func main() {
	r := mux.NewRouter().StrictSlash(true)
	handleFunc(r, "/", sqlweb.ServeHome, false, true)
	handleFunc(r, "/home", sqlweb.ServeHome, true, true)
	handleFunc(r, "/query", sqlweb.ServeQuery, true, true)
	handleFunc(r, "/downloadColumn", sqlweb.DownloadColumn, true, true)
	handleFunc(r, "/tablesByColumn", sqlweb.ServeTablesByColumn, false, true)
	handleFunc(r, "/loadLinksConfig", sqlweb.ServeLoadLinksConfig, false, true)
	handleFunc(r, "/saveLinksConfig", sqlweb.ServeSaveLinksConfig, false, true)
	handleFunc(r, "/iconfont.{extension}", sqlweb.ServeFont("iconfont."), true, false)
	handleFunc(r, "/favicon.ico", htt.ServeFavicon("favicon.ico", sqlweb.MustAsset, sqlweb.AssetInfo), true, false)
	handleFunc(r, "/update", sqlweb.WrapHandlerFunc(sqlweb.ServeUpdate), false, true)
	handleFunc(r, "/exportDatabase", sqlweb.ExportDatabase, true, true)
	if sqlweb.AppConf.ImportDb {
		handleFuncNoDump(r, "/importDatabase", sqlweb.ImportDatabase, false)
	}
	if sqlweb.AppConf.MultiTenants {
		handleFunc(r, "/multipleTenantsQuery", sqlweb.MultipleTenantsQuery, true, true)
	}
	handleFunc(r, "/searchDb", sqlweb.ServeSearchDb, false, true)
	handleFunc(r, "/action", sqlweb.ServeAction, false, true)

	http.HandleFunc(sqlweb.AppConf.ContextPath+"/static/", ServeStatic(sqlweb.AppConf.ContextPath))
	http.Handle("/", r)

	fmt.Println("start to listen at ", sqlweb.AppConf.ListenPort)
	OpenExplorerWithContext(sqlweb.AppConf.ContextPath, strconv.Itoa(sqlweb.AppConf.ListenPort))

	if err := http.ListenAndServe(":"+strconv.Itoa(sqlweb.AppConf.ListenPort), nil); err != nil {
		log.Fatal(err)
	}
}

// OpenExplorerWithContext ...
func OpenExplorerWithContext(contextPath, port string) {
	go func() {
		time.Sleep(100 * time.Millisecond) // nolint gomnd

		switch runtime.GOOS {
		case "windows":
			fallthrough
		case "darwin":
			_ = open.Run("http://127.0.0.1:" + port + contextPath + "?" + ran.String(10)) // nolint gomnd
		}
	}()
}

func ServeStatic(contextPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, contextPath)[1:]
		fi, _ := sqlweb.AssetInfo(filename)
		if fi == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		buffer := bytes.NewReader(sqlweb.MustAsset(filename))
		w.Header().Set("Content-Type", DetectContentType(fi.Name()))
		w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, buffer)
	}
}

func DetectContentType(name string) (t string) {
	if t = mime.TypeByExtension(filepath.Ext(name)); t == "" {
		t = "application/octet-stream"
	}

	return
}

type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandlerFunc(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := GzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func DumpRequest(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Save a copy of this request for debugging.
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Println(err)
		}
		log.Println(string(requestDump))
		fn(w, r)
	}
}

func handleFuncNoDump(r *mux.Router, path string, f http.HandlerFunc, requiredGzip bool) {
	wrap := f

	if requiredGzip {
		wrap = GzipHandlerFunc(wrap)
	}

	r.HandleFunc(sqlweb.AppConf.ContextPath+path, wrap)
}

func handleFunc(r *mux.Router, path string, f http.HandlerFunc, requiredGzip, requiredBasicAuth bool) {
	wrap := DumpRequest(f)
	if requiredBasicAuth && len(sqlweb.AppConf.BasicAuth) > 0 {
		wrap = BasicAuth(wrap, sqlweb.AppConf.BasicAuth)
	} else {
		wrap = anonymousWrap(wrap)
	}

	if requiredGzip {
		wrap = GzipHandlerFunc(wrap)
	}

	p := filepath.Join(sqlweb.AppConf.ContextPath, path)
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	r.HandleFunc(p, wrap)
}

func anonymousWrap(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rr := r.WithContext(context.WithValue(r.Context(), "CookieValue", &htt.CookieValueImpl{
			UserID:    "anonymous",
			Name:      "anonymous",
			Avatar:    "",
			CsrfToken: "",
		}))
		f(w, rr)
	}
}

func BasicAuth(fn http.HandlerFunc, basicAuth []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		basicAuthPrefix := "Basic "

		// 获取 request header
		auth := r.Header.Get("Authorization")
		// 如果是 http basic auth
		if strings.HasPrefix(auth, basicAuthPrefix) {
			// 解码认证信息
			payload, _ := base64.StdEncoding.DecodeString(auth[len(basicAuthPrefix):])
			if yes, user := Contains(basicAuth, string(payload)); yes {
				rr := r.WithContext(context.WithValue(r.Context(), "CookieValue", &htt.CookieValueImpl{
					UserID:    user,
					Name:      user,
					Avatar:    "",
					CsrfToken: "",
				}))
				fn(w, rr) // 执行被装饰的函数
				return
			}
		}

		w.Header().Set("Content-Type", "'Content-type:text/html;charset=ISO-8859-1'")
		// 认证失败，提示 401 Unauthorized
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		// 401 状态码
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func Contains(ss []string, s string) (bool, string) {
	for _, el := range ss {
		if s == el {
			return true, el[:strings.Index(el, ":")]
		}
	}

	return false, ""
}
