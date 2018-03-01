package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type rankItem struct {
	Rank       int64  `json:"rank,string"`
	Move       int64  `json:"move,string"`
	IconURL    string `json:"icon"`
	Name       string `json:"nick"`
	Job        string `json:"job"`
	DetailJob  string `json:"detail_job"`
	Level      int64  `json:"level"`
	Exp        int64  `json:"exp"`
	Popularity int64  `json:"popular"`
	FloorStr   string `json:"floor"`
	Duration   string `json:"duration"`
	GuildID    int64  `json:"guild_worldid,string"` // ?

	Second          int   `json:"second,omitempty"`
	Minute          int   `json:"minute,omitempty"`
	World           int   `json:"world,omitempty"`
	Floor           int   `json:"rawfloor,omitempty"`
	Type            int   `json:"type,omitempty"`
	CheckedTimeUnix int64 `json:"checkedtime,omitempty"`
}

const timeFormat = "2006년 1월 2일"

var nameLE *walk.LineEdit
var searchPB *walk.PushButton
var maxFloorNE, recentFloorNE,
	maxMinuteNE, recentMinuteNE,
	maxSecondNE, recentSecondNE *walk.NumberEdit
var maxDateLE, recentDateLE *walk.LineEdit
var termLB *walk.Label
var serverCB *walk.ComboBox
var mw *walk.MainWindow
var serverList = []*struct{ Name string }{
	{"리부트"},
	{"리부트2"},
}

var serverIDList = []int{
	1,
	12,
}

const serverURL = `http://127.0.0.1:4412/getrank`

func search() {
	searchPB.SetEnabled(false)
	defer searchPB.SetEnabled(true)

	var request struct {
		World int
		Type  int
		Name  string
	}

	request.World = serverIDList[serverCB.CurrentIndex()]
	request.Type = 2
	request.Name = strings.Trim(nameLE.Text(), " \r\n")

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(request)
	resp, err := http.Post(serverURL, "application/json", &b)
	if err != nil {
		walk.MsgBox(mw, "오류", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	var response struct {
		Ok    bool
		Rank  rankItem
		MRank rankItem
		Start int64
		End   int64
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&response); err != nil {
		walk.MsgBox(mw, "오류", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	if !response.Ok {
		for _, ne := range []*walk.NumberEdit{
			maxFloorNE, recentFloorNE,
			maxMinuteNE, recentMinuteNE,
			maxSecondNE, recentSecondNE,
		} {
			ne.SetValue(-1)
		}
		recentDateLE.SetText("검색 실패")
		maxDateLE.SetText("검색 실패")
	} else {
		nameLE.SetText(response.Rank.Name)

		recentFloorNE.SetValue(float64(response.Rank.Floor))
		recentMinuteNE.SetValue(float64(response.Rank.Minute))
		recentSecondNE.SetValue(float64(response.Rank.Second))
		recentDateLE.SetText(time.Unix(response.Rank.CheckedTimeUnix, 0).Format(timeFormat))

		maxFloorNE.SetValue(float64(response.MRank.Floor))
		maxMinuteNE.SetValue(float64(response.MRank.Minute))
		maxSecondNE.SetValue(float64(response.MRank.Second))
		maxDateLE.SetText(time.Unix(response.MRank.CheckedTimeUnix, 0).Format(timeFormat))
	}

	if response.Start > 0 && response.End > 0 {
		stime, etime := time.Unix(response.Start, 0), time.Unix(response.End, 0)
		termLB.SetText("데이터 수집 기간: " + stime.Format("2006-01-02") + " ~ " + etime.Format("2006-01-02"))
	}
}

func main() {
	if runtime.GOMAXPROCS(0) < 2 {
		runtime.GOMAXPROCS(2)
	}

	mwd := MainWindow{
		AssignTo: &mw,
		Title:    "무릉전적 v2",
		Layout:   Grid{Columns: 3},
		Children: []Widget{
			Label{Text: "닉네임:"},
			LineEdit{AssignTo: &nameLE, MinSize: Size{100, 0}, OnEditingFinished: search},
			ComboBox{AssignTo: &serverCB, DisplayMember: "Name", Model: serverList, CurrentIndex: 0},
			PushButton{AssignTo: &searchPB, ColumnSpan: 3, Text: "검색", OnClicked: search},
			Composite{
				ColumnSpan: 3,
				Layout:     Grid{Columns: 2, MarginsZero: true},
				Children: []Widget{
					Composite{
						Layout: Grid{Columns: 2, MarginsZero: true},
						Children: []Widget{
							Label{ColumnSpan: 2, Text: "최고 기록"},
							NumberEdit{ColumnSpan: 2, AssignTo: &maxFloorNE, Suffix: "층", ReadOnly: true},
							NumberEdit{AssignTo: &maxMinuteNE, Suffix: "분", ReadOnly: true},
							NumberEdit{AssignTo: &maxSecondNE, Suffix: "초", ReadOnly: true},
							LineEdit{ColumnSpan: 2, AssignTo: &maxDateLE, Text: "달성 시각", Alignment: AlignCenter, ReadOnly: true},
						},
					},
					Composite{
						Layout: Grid{Columns: 2, MarginsZero: true},
						Children: []Widget{
							Label{ColumnSpan: 2, Text: "최근 기록"},
							NumberEdit{ColumnSpan: 2, AssignTo: &recentFloorNE, Suffix: "층", ReadOnly: true},
							NumberEdit{AssignTo: &recentMinuteNE, Suffix: "분", ReadOnly: true},
							NumberEdit{AssignTo: &recentSecondNE, Suffix: "초", ReadOnly: true},
							LineEdit{ColumnSpan: 2, AssignTo: &recentDateLE, Text: "달성 시각", Alignment: AlignCenter, ReadOnly: true},
						},
					},
				},
			},
			Label{AssignTo: &termLB, ColumnSpan: 3, Text: "데이터 수집 기간: 2000-00-00 ~ 2000-00-00"},
			Label{ColumnSpan: 3, Text: "정확한 검색을 보증하지 않습니다 (Beta)", TextColor: walk.RGB(255, 0, 0)},
			Label{ColumnSpan: 3, Text: "달성 시각은 최대 ±1일의 오차가 있습니다.", TextColor: walk.RGB(0, 0, 255)},
		},
	}

	if _, err := mwd.Run(); err != nil {
		fmt.Println("GUI Error:", err)
	}
}
