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

var saveCh = make(chan *entity.UserInfo, 3)
var userCh = make(chan *entity.UserInfo, 3)
var db = initDb()

//http client
var client = &http.Client{}
//todo set cookie
const cookie = ""
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
		if names != nil{
			u.Name = names[1]
		}
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
		db.Save(u)
		fmt.Println(u)
	}
}

