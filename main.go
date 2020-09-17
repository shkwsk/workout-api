package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"google.golang.org/appengine"
)

// Response ...
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Workout モデルの宣言 (gormではDB名はstructの変数名の複数形になる)
type Workout struct {
	WorkoutID     int64     `gorm:"column:workout_id;primary_key;unique;not null;auto_increment"`
	UserID        string    `json:"user_id" gorm:"column:user_id;not null"`
	DistanceMeter float64   `json:"distance_meter" gorm:"distance_meter"`
	StartedAt     time.Time `json:"started_at" time_format:"2006-01-01T00:00:00" gorm:"started_at"`
	Seconds       int64     `json:"seconds" gorm:"seconds"`
	CreatedAt     time.Time `gorm:"created_at" sql:"DEFAULT:current_timestamp"`
}

func gormConnect() *gorm.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	USER := os.Getenv("workout_db_USER")
	PASS := os.Getenv("workout_db_PASS")
	CONNECTIONNAME := os.Getenv("workout_db_CONNECTIONNAME")
	DBNAME := os.Getenv("workout_db_DBNAME")
	localConnection := USER + ":" + PASS + "@/" + DBNAME + "?parseTime=true"
	cloudSQLConnection := USER + ":" + PASS + "@unix(/cloudsql/" + CONNECTIONNAME + ")/" + DBNAME + "?parseTime=true"
	var db *gorm.DB

	if appengine.IsAppEngine() {
		db, err = gorm.Open("mysql", cloudSQLConnection)
	} else {
		db, err = gorm.Open("mysql", localConnection)
	}
	if err != nil {
		panic(err.Error())
	}
	return db
}

// DBの初期化
func dbInit() {
	db := gormConnect()
	// コネクション解放
	defer db.Close()
	db.AutoMigrate(&Workout{}) //構造体に基づいてテーブルを作成
}

// ワークアウト登録処理
func (workout *Workout) create() error {
	db := gormConnect()
	defer db.Close()
	// Insert処理
	if err := db.Create(workout).Error; err != nil {
		return err
	}
	return nil
}

func main() {
	http.HandleFunc("/insert", insertWorkout)
	http.HandleFunc("/", handle)
	appengine.Main()
}

func insertWorkout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("This method allows only POST."))
		return
	}

	body := r.Body
	defer body.Close()

	// json parse
	buf := new(bytes.Buffer)
	io.Copy(buf, body)
	log.Println(buf)
	workout := Workout{}
	if err := json.Unmarshal(buf.Bytes(), &workout); err != nil {
		log.Fatal("Body json parse error", buf, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.WithFields(log.Fields{
		"user_id":        workout.UserID,
		"distance_meter": workout.DistanceMeter,
		"started_at":     workout.StartedAt,
		"seconds":        workout.Seconds,
	}).Info("insertWorkout")

	// db書き込み
	if err := workout.create(); err != nil {
		log.Fatal("create workout error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func handle(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Response{Status: "ok", Message: "Hello world."})
}
