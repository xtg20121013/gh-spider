package main

import (
	"fmt"
	"github_spider/entity"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
)

var PROJECTS = []string{
	"/CyC2018/GFM-Converter",
	"/frank-lam/fullstack-tutorial",
	"/CyC2018/Job-Recommend",
	"/CyC2018/CS-Notes",
	"/CyC2018/Markdown-Resume",
	"/aylei/interview",
}

var saveCh = make(chan *entity.UserInfo)
var userCh = make(chan *entity.UserInfo)
var db = initDb()

//http client
var client = &http.Client{}
const cookie = "_octo=GH1.1.1278709752.1596600401; _ga=GA1.2.1804804220.1596600403; _device_id=f1c5660473fcc5bf40742ff9346d51fe; tz=Asia%2FShanghai; tz=Asia%2FShanghai; has_recent_activity=1; _gat=1; user_session=gsAu7T6MyN18IYBmbx5i-goC22VbtOCCCko7Hb7ZoYwvlaFi; __Host-user_session_same_site=gsAu7T6MyN18IYBmbx5i-goC22VbtOCCCko7Hb7ZoYwvlaFi; logged_in=yes; dotcom_user=xtg20121013; _gh_sess=PntdSr3UHsWzOM0AjJP5FSrormjkQt9Kt67Z4gLqycVUNqJW%2B0Kl6U48JuhGZEUJesscCKow43BokHkNUlgiIXNu34x3XsPfkmY%2FrGYZMJD62lS1kq0I4lP33xIGRnzje5MpIG%2BTrkFQNZPp6i4U%2F7BCI0By8fwJlwj3IAh3vAxLf%2B2dNiAr7VCOP1YkYlod3htirLgSsgPQpukjWiKmGzOSnWPcyS2mgtFsnMfPGLIIK1a7L3ukb7d%2BfeHi%2BHenyO0WoPRpuFRLWuFmfnQ%2BqPuwq5wU98k0kRgJ6sF80Gs0Kq2Fr4X6ma9EwolM%2FICeBM5rrvisSm83YrBOm%2B1shjbhyFTtet5PyCLnEQmf5W3l5k8U0Z%2Bqo8sHmynL3abnP0Il3i%2Fuy%2Bxj44gu%2B0ZXFdqGIzSrwXhm5geFCLvNbnMxh75FK%2Fo3em1bkRy87%2FIiWhZ97CGAW07YvHZdEMH%2FIHBpJ1Kj%2Bs%2BK%2FokysmDZOyXNkYbkDEAMKZuHTWoU5Ud8U6HZor%2FuM9z88%2Bw%2FFIDrdeLnysMgihzn%2F6AZPK4B%2FBKxMIWpbxBC0nEIf8y1naXY30yPmnMEP1W9OxCseuMpdkbA%2F1OwNyArem3zKD%2F5vY01OvP7BP9yebqzcrRzgoaX4n3u1bktGbZHNvzefboCkoPE1BDiywgXEAChgq56WLVUZomvXmJaVr3Bwcj6xcu59VOaoLEcquS%2BJeLajNWmbNws8ggaua6XxToaNQZZohUvLVkB6kPwr0qZQZgv6Kn%2BE9JeMowVURmMQ%2Bjh%2BzY253KLrQvszj1H3jnZeLpbbHrbJSTdsyeUQXI6SO1Ms0fL%2BCXAMT7IQjsZ8AXqomI9s1%2FYJUUvTdQPaT3bP43UdqZyk52s75aM0VWcSAksKjHoF83sfB%2FJRPZ441ITvzJ3GAZXvNBJvVI0NJXay4yVlE0YZXT9jLwOYJQsYjmKn6dpc5VHUfm5gf4Y7f8%2F%2FfDxzpd%2F6tBtxJcndXwYF0SwvGyhSrev--2AHxsgBq%2FMffNkZa--cfbDVFw9iaQW%2FiaY97vSTg%3D%3D"
const githubUrl = "https://github.com"

func main() {
	for _, path := range PROJECTS{
		go fetchWatches(path)
		go fetchStargazers(path)
		go fetchForks(path)
	}
	go queryUser()
	saveUser()
}

func reqGet(url string) (*http.Response, string) {
	req,_ := http.NewRequest("GET",url,nil)
	req.Header.Add("Cookie", cookie)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return resp, ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	doc := string(body)
	return resp, doc
}

func initDb() *gorm.DB {
	dsn := "root:1234567890@tcp(localhost:3306)/mhq?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return db
}

func fetchWatches(path string) {
	for page := int32(1); ; page ++ {
		url := fmt.Sprintf("%s%s%s%d", githubUrl, path, "/watchers?page=", page)
		res, body:= reqGet(url)
		if res.StatusCode != 200{
			continue
		}
		doc := body
		innerPageReg := regexp.MustCompile(`(?s:<h2>Watchers</h2>(.*)<div class="paginate-container">)`)
		innerPage := innerPageReg.FindStringSubmatch(doc)
		userReg := regexp.MustCompile(`href="/([a-zA-Z0-9-]*)"`)
		names := userReg.FindAllStringSubmatch(innerPage[1], -1)
		if len(names) == 0{
			break
		}
		nameMap := map[string]string{}
		for _, name := range names {
			nameMap[name[1]] = name[1]
		}
		for k := range nameMap{
			u := entity.UserInfo{Index: k, From: url}
			userCh <- &u
		}
	}
}

func fetchStargazers(path string)  {
	url := fmt.Sprintf("%s%s%s", githubUrl, path, "/stargazers")
	for true{
		res, body:= reqGet(url)
		if res.StatusCode != 200{
			log.Fatal("fetchStargazers error! url="+url)
		}
		doc := body
		userReg := regexp.MustCompile(`data-octo-dimensions="link_type:self" href="/([a-zA-Z0-9-]*)"`)
		names := userReg.FindAllStringSubmatch(doc, -1)
		nameMap := map[string]string{}
		for _, name := range names {
			nameMap[name[1]] = name[1]
		}
		for k := range nameMap{
			u := entity.UserInfo{Index: k, From: url}
			userCh <- &u
		}
		//获取下一页链接
		nextReg := regexp.MustCompile(`Previous(</button>|</a>)<a rel="nofollow" class="btn btn-outline BtnGroup-item" href="(.*?)">Next</a></div>`)
		nexts := nextReg.FindStringSubmatch(doc)
		if nexts != nil{
			url = nexts[2]
		}else {
			break
		}
	}
}

func fetchForks(path string) {
	url := fmt.Sprintf("%s%s%s", githubUrl, path, "/network/members")
	res, body:= reqGet(url)
	if res.StatusCode != 200{
		log.Fatal("fetchForks error! url="+url)
	}
	doc := body
	userReg := regexp.MustCompile(`data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" href="/([a-zA-Z0-9-]*)">`)
	names := userReg.FindAllStringSubmatch(doc, -1)
	if len(names) == 0{
		return
	}
	nameMap := map[string]string{}
	for _, name := range names {
		nameMap[name[1]] = name[1]
	}
	for k := range nameMap{
		u := entity.UserInfo{Index: k, From: url}
		userCh <- &u
	}
}

func queryUser() {
	for v := range userCh{
		u := *v
		url := githubUrl + "/" + u.Index
		res, body:= reqGet(url)
		if res.StatusCode != 200{
			continue
		}
		doc := body
		// 昵称获取
		nameReg := regexp.MustCompile(`itemprop="name">(.*)</span>`)
		names := nameReg.FindStringSubmatch(doc)
		u.Name = names[1]
		// 组织获取
		companyReg := regexp.MustCompile(`aria-label="Organization: (.*?)"`)
		companys := companyReg.FindStringSubmatch(doc)
		if companys != nil{
			u.Company = companys[1]
		}
		// 地区
		locationReg := regexp.MustCompile(`aria-label="Home location: (.*?)"`)
		locations := locationReg.FindStringSubmatch(doc)
		if locations != nil{
			u.Location = locations[1]
		}
		// 邮箱
		mailReg := regexp.MustCompile(`aria-label="Email: (.*?)"`)
		mails := mailReg.FindStringSubmatch(doc)
		if mails != nil{
			u.Mail = mails[1]
		}
		// website
		websiteReg := regexp.MustCompile(`<a rel="nofollow me" class="link-gray-dark " href="(.*?)">`)
		websites := websiteReg.FindStringSubmatch(doc)
		if websites != nil{
			u.Website = websites[1]
		}
		saveCh <- &u
	}
}

func saveUser()  {
	for u := range saveCh{
		oldU := entity.UserInfo{}
		db.Where(&entity.UserInfo{Index: u.Index}).First(&oldU)
		if oldU.ID == 0{
			db.Save(u)
		}else if u.Mail != "" {
			oldU.Mail = u.Mail
			oldU.From = u.From
			db.Save(&oldU)
		}
		fmt.Println(u)
	}
}

