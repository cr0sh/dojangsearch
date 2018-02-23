package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unsafe"
)

const timeFormat = "2006-01-02 15:04:05"

var verbLog = log.New(os.Stdout, "INFO:  ", log.Lshortfile|log.Ldate|log.Ltime)
var warnLog = log.New(os.Stderr, "WARN:  ", log.Lshortfile|log.Ldate|log.Ltime)
var errLog = log.New(os.Stderr, "ERROR: ", log.Lshortfile|log.Ldate|log.Ltime)

var lastCrawlTimeLock sync.Mutex
var lastCrawlTime int64

var db *bolt.DB

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

func (r *rankItem) fullsec() int {
	return r.Second + r.Minute*60
}

func crawlJob() {
	lastCrawlTimeLock.Lock()
	if lastCrawlTime != 0 {
		errLog.Println("Crawler: Another crawler is already running since", time.Unix(lastCrawlTime, 0).Format(timeFormat))
		lastCrawlTimeLock.Unlock()
		return
	}
	now := time.Now()
	lastCrawlTime = now.Unix()
	verbLog.Println("Crawler: Started ranking crawler at", now.Format(timeFormat))
	lastCrawlTimeLock.Unlock()
	defer func() {
		lastCrawlTimeLock.Lock()
		verbLog.Println("Crawler: Finished ranking crawler since", time.Unix(lastCrawlTime, 0).Format(timeFormat), "at", time.Now().Format(timeFormat))
		lastCrawlTime = 0
		lastCrawlTimeLock.Unlock()
	}()

	verbLog.Println("Crawler: Starting HTTP client for R1")
	r1ranks, err := crawlDojangRank(1, 2)
	if err != nil {
		errLog.Println("Crawler: crawlDojangRank failed:", err)
		return
	}

	verbLog.Printf("Crawler: Updating database for R1 (%d items)", len(r1ranks))
	if err := updateDatabase(1, 2, r1ranks, now); err != nil {
		errLog.Println("Crawler: Error while boltDB update Transaction:", err)
		return
	}

	verbLog.Println("Crawler: Starting HTTP client for R2")
	r2ranks, err := crawlDojangRank(12, 2)
	if err != nil {
		errLog.Println("Crawler: crawlDojangRank failed:", err)
		return
	}

	verbLog.Printf("Crawler: Updating database for R2 (%d items)", len(r2ranks))
	if err := updateDatabase(12, 2, r2ranks, now); err != nil {
		errLog.Println("Crawler: Error while boltDB update Transaction:", err)
		return
	}
}

func crawlDojangRank(world, typeid int) ([]rankItem, error) {
	idx := 1
	ranks := make([]rankItem, 0, 200)
	t := time.NewTicker(time.Millisecond * 200)
	defer t.Stop()

	for range t.C {
		u, _ := url.Parse("http://m.maplestory.nexon.com/MapleStory/Data/Json/Ranking/DojangThisWeekListJson.aspx")
		q := u.Query()
		q.Add("rankidx", strconv.Itoa(idx))
		q.Add("cateType", strconv.Itoa(typeid))
		q.Add("GameWorldID", strconv.Itoa(world))
		u.RawQuery = q.Encode()
		r, err := http.Get(u.String())
		if r != nil {
			defer r.Body.Close()
		}

		if err != nil {
			return nil, err
		}

		var resp struct {
			Result  string     `json:"result"`
			List    []rankItem `json:"list"`
			NextIdx int        `json:"nextidx,string"`
		}

		dec := json.NewDecoder(r.Body)
		dec.Decode(&resp)
		if len(resp.List) == 0 {
			break
		}
		ranks = append(ranks, resp.List...)
		idx = resp.NextIdx

		io.Copy(ioutil.Discard, r.Body)
	}
	return ranks, nil
}

func updateDatabase(world, typeid int, ranks []rankItem, updateTime time.Time) error {
	return db.Update(func(tx *bolt.Tx) error {
		br, err := tx.CreateBucketIfNotExists([]byte("recent-" + strconv.Itoa(world) + "-" + strconv.Itoa(typeid)))
		if err != nil {
			return err
		}

		bm, err := tx.CreateBucketIfNotExists([]byte("maxrecord-" + strconv.Itoa(world) + "-" + strconv.Itoa(typeid)))
		if err != nil {
			return err
		}

		bmeta, err := tx.CreateBucketIfNotExists([]byte("metadata-" + strconv.Itoa(world) + "-" + strconv.Itoa(typeid)))
		if err != nil {
			return err
		}
		for _, rank := range ranks {
			dur := []rune(rank.Duration)
			fl := []rune(rank.FloorStr)
			idx := 0
			for _, d := range dur {
				if unicode.IsNumber(d) {
					idx++
				} else {
					break
				}
			}

			idx2 := 0
			for _, f := range fl {
				if unicode.IsNumber(f) {
					idx2++
				} else {
					break
				}
			}

			var err error
			if rank.Minute, err = strconv.Atoi(string(dur[:idx])); err != nil {
				return err
			}
			if rank.Second, err = strconv.Atoi(string(dur[idx+2 : len(dur)-1])); err != nil {
				return err
			}
			if rank.Floor, err = strconv.Atoi(string(fl[:idx2])); err != nil {
				return err
			}
			rank.CheckedTimeUnix = updateTime.Unix()

			buf, err := json.Marshal(rank)
			if err != nil {
				return err
			}

			mbuf := bm.Get([]byte(rank.Name))
			if mbuf == nil {
				bm.Put([]byte(strings.ToLower(rank.Name)), buf)
				continue
			}

			var mrank rankItem
			if err := json.Unmarshal(mbuf, &mrank); err != nil {
				return err
			}

			if mrank.Floor == rank.Floor && mrank.fullsec() == rank.fullsec() {
				rank.CheckedTimeUnix = mrank.CheckedTimeUnix
				buf, err = json.Marshal(rank)
				if err != nil {
					return err
				}
			}

			br.Put([]byte(strings.ToLower(rank.Name)), buf)

			if mrank.Floor < rank.Floor || (mrank.Floor == rank.Floor && mrank.fullsec() > rank.fullsec()) {
				bm.Put([]byte(strings.ToLower(rank.Name)), buf)
			}
		}

		ntime := updateTime.Unix()
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, *(*uint64)(unsafe.Pointer(&ntime)))

		startbuf := bmeta.Get([]byte("start"))
		if startbuf == nil {
			bmeta.Put([]byte("start"), buf)
		}
		bmeta.Put([]byte("end"), buf)
		return nil
	})
}

func main() {
	update := flag.Bool("update", false, "Updates database at start if provided")
	laddr := flag.String("addr", ":4412", "Bind address for HTTP server")
	flag.Parse()
	verbLog.Println("Opening boltDB database")
	var err error
	if db, err = bolt.Open("database.db", 0600, nil); err != nil {
		errLog.Fatal("bolt.Open:", err)
	}
	verbLog.Println("Successfully opened database")

	verbLog.Println("Starting initial crawler")

	c := cron.New()
	c.AddFunc("@every 8h", crawlJob)

	verbLog.Println("Starting cronjob runner")
	c.Start()

	go func() {
		c_ := make(chan os.Signal, 1)
		signal.Notify(c_, os.Interrupt)

		s := <-c_
		verbLog.Println("Interrupt signal received:", s)
		verbLog.Println("Closing DB")
		if err := db.Close(); err != nil {
			errLog.Fatal("db.Close:", err)
		}
		os.Exit(0)
	}()

	if *update {
		verbLog.Println("Updating database at start(-update flag provided)")
		crawlJob()
	}

	http.HandleFunc("/getrank", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/json")
		var request struct {
			World int
			Type  int
			Name  string
		}
		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&request); err != nil {
			errLog.Println("HTTP: Request parse failed:", err)
			return
		}

		if err := db.View(func(tx *bolt.Tx) error {
			var response struct {
				Ok    bool
				Rank  rankItem
				MRank rankItem
				Start int64
				End   int64
			}
			enc := json.NewEncoder(w)

			br := tx.Bucket([]byte("recent-" + strconv.Itoa(request.World) + "-" + strconv.Itoa(request.Type)))
			if br == nil {
				enc.Encode(response)
				return nil
			}

			bm := tx.Bucket([]byte("maxrecord-" + strconv.Itoa(request.World) + "-" + strconv.Itoa(request.Type)))
			if bm == nil {
				enc.Encode(response)
				return nil
			}

			bmeta := tx.Bucket([]byte("metadata-" + strconv.Itoa(request.World) + "-" + strconv.Itoa(request.Type)))
			if bmeta == nil {
				enc.Encode(response)
				return nil
			}

			rank, mrank := br.Get([]byte(strings.ToLower(request.Name))), bm.Get([]byte(strings.ToLower(request.Name)))
			start, end := bmeta.Get([]byte("start")), bmeta.Get([]byte("end"))
			if start != nil && end != nil {
				ustart, uend := binary.BigEndian.Uint64(start), binary.BigEndian.Uint64(end)
				response.Start, response.End = *(*int64)(unsafe.Pointer(&ustart)), *(*int64)(unsafe.Pointer(&uend))
			}

			if rank == nil || mrank == nil {
				if err := enc.Encode(response); err != nil {
					return err
				}
				return nil
			}
			response.Ok = true

			if err := json.Unmarshal(rank, &response.Rank); err != nil {
				return err
			}
			if err := json.Unmarshal(mrank, &response.MRank); err != nil {
				return err
			}

			if err := enc.Encode(response); err != nil {
				return err
			}

			return nil
		}); err != nil {
			errLog.Println("HTTP: db.View failed:", err)
		}
	})

	verbLog.Println("Starting HTTP server on", *laddr)
	http.ListenAndServe(*laddr, nil)
}
