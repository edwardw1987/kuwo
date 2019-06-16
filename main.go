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

func (this *Client) Get(path string, args ...interface{}) ([]byte, error) {
	url := fmt.Sprintf(path, args...)
	if !strings.HasPrefix(url, "http") {
		url = ConcatUrl(Endpoint, url)
	}
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func (this *Client) SearchMusicBykeyWord(key string, pageNum, rowNum int) []*MusicInfo {
	/*GET Music List*/
	path := "/api/www/search/searchMusicBykeyWord?key=%s&pn=%d&rn=%d"
	respBytes, err := this.Get(path, url.QueryEscape(key), pageNum, rowNum)
	check(err)
	searchKeyRes := new(SearchKeyReponse)
	json.Unmarshal(respBytes, searchKeyRes)
	return searchKeyRes.Data.List

}

func (this *Client) DowloadMusicByInfo(musicInfo *MusicInfo) {
	path := "/url?format=mp3&rid=%d&response=url&type=convert_url3&br=128kmp3&from=web&t=%s"
	respBytes, err := this.Get(path, musicInfo.Rid, GetTimeStamp()[:13])
	check(err)
	result := new(MusicUrlResponse)
	json.Unmarshal(respBytes, result)

	fmt.Printf("Use result.Url %s to download\n", result.Url)
	respBytes, err = this.Get(result.Url)
	check(err)

	outputname := fmt.Sprintf("%s-%s.mp3", musicInfo.Artist, musicInfo.Name)
	outputpath := filepath.Join(getCurdir(), outputname)
	fmt.Printf("Save to:%s\n", outputpath)
	Save(respBytes, outputpath)

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
	flag.Parse()
	keyword, pagenum, rownum, rid := *keywordPtr, *pagenumPtr, *rownumPtr, *ridPtr
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
		clt.DowloadMusicByInfo(memo[rid])
	}

}
