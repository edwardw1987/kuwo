package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getCurdir() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir
}

const Endpoint = "http://www.kuwo.cn"
const UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36"

type SearchKeyReponse struct {
	Data *DataField
}
type DataField struct {
	List  []*MusicInfo
	Total string
}
type MusicInfo struct {
	Rid             int
	Name            string
	Artist          string
	Album           string
	SongTimeMinutes string
}

type MusicUrlResponse struct {
	Code int
	Msg  string
	Url  string
}
type Client struct {
}

func ConcatUrl(endpoint string, path string) string {
	return strings.Join([]string{endpoint, path}, "")
}

func GetTimeStamp() string {
	return strconv.Itoa(int(time.Now().UnixNano()))
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (this *Client) Get(path string, reqHeaders map[string]string, args ...interface{}) ([]byte, error) {
	url := fmt.Sprintf(path, args...)
	if !strings.HasPrefix(url, "http") {
		url = ConcatUrl(Endpoint, url)
	}
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	check(err)
	if reqHeaders != nil {
		for key, value := range reqHeaders {
			req.Header.Add(key, value)
		}
	}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}
func (this *Client) GetUrlCookie(url, name string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	resp, err := client.Do(req)
	check(err)
	m := map[string]string{}
	for _, cookie := range resp.Cookies() {
		cookieStr := cookie.String()
		index := strings.Index(cookieStr, ";")
		cookieStr = cookieStr[:index]
		pairs := strings.Split(cookieStr, "=")
		key, value := pairs[0], pairs[1]
		m[key] = value
	}
	return m[name]

}
func (this *Client) SearchMusicBykeyWord(key string, pageNum, rowNum int) []*MusicInfo {
	/*GET Music List*/
	urlSearchList := "http://www.kuwo.cn/search/list"
	kw_token := this.GetUrlCookie(urlSearchList, "kw_token")

	path := "/api/www/search/searchMusicBykeyWord?key=%s&pn=%d&rn=%d"
	referer := fmt.Sprintf("%s?key=%s", urlSearchList, url.QueryEscape(key))
	reqHeaders := map[string]string{
		"Cookie":  fmt.Sprintf("kw_token=%s", kw_token),
		"csrf":    kw_token,
		"Referer": referer,
		"User-Agent": UA,
	}
	respBytes, err := this.Get(path, reqHeaders, url.QueryEscape(key), pageNum, rowNum)
	check(err)
	searchKeyRes := new(SearchKeyReponse)
	json.Unmarshal(respBytes, searchKeyRes)
	return searchKeyRes.Data.List

}

func (this *Client) DowloadMusicByInfo(musicInfo *MusicInfo, name string, dl bool) {
	path := "/url?format=mp3&rid=%d&response=url&type=convert_url3&br=128kmp3&from=web&t=%s"
	respBytes, err := this.Get(path, nil, musicInfo.Rid, GetTimeStamp()[:13])
	check(err)
	result := new(MusicUrlResponse)
	json.Unmarshal(respBytes, result)

	fmt.Printf("%s\n", result.Url)
	if dl {
		respBytes, err = this.Get(result.Url, nil)
		check(err)
		if len(name) == 0 {
			name = musicInfo.Name
		}
		outputname := fmt.Sprintf("%s-%s.mp3", musicInfo.Artist, name)
		outputpath := filepath.Join(getCurdir(), outputname)
		fmt.Printf("Save to:%s\n", outputpath)
		Save(respBytes, outputpath)
	}

}

func Save(bytes []byte, savepath string) {

	out, err := os.Create(savepath)
	check(err)
	defer out.Close()
	out.Write(bytes)
	out.Sync()
}

func main() {

	keywordPtr := flag.String("k", "", "the keyword for searching")
	pagenumPtr := flag.Int("p", 1, "number of page of music list")
	rownumPtr := flag.Int("r", 20, "number of rows of every page")
	ridPtr := flag.Int("rid", -1, "id of song")
	dlPtr := flag.Bool("dl", false, "download mp3 file?")
	namePtr := flag.String("n", "", "name of song without artist or file-extension")
	flag.Parse()
	keyword, pagenum, rownum, rid, name := *keywordPtr, *pagenumPtr, *rownumPtr, *ridPtr, *namePtr
	if len(keyword) == 0 {
		fmt.Printf("[WARNING] Argument -k Required\n")
		return
	}
	clt := new(Client)
	memo := make(map[int]*MusicInfo)
	musicInfoList := clt.SearchMusicBykeyWord(keyword, pagenum, rownum)
	lineFormat := "[%02d] %v | %s | %s | %s | %s\n"
	for i, info := range musicInfoList {
		memo[info.Rid] = info
		if rid == -1 {
			fmt.Printf(lineFormat, i+1, info.Rid, info.Artist, info.Name, info.Album, info.SongTimeMinutes)
		}
	}
	if rid > 0 {
		if info, ok := memo[rid]; ok {
			clt.DowloadMusicByInfo(info, name, *dlPtr)
		}
	}

}
