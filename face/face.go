package face

/*
#cgo windows LDFLAGS: -L ../xface -lxface_win
#cgo linux LDFLAGS: -L ../build -L ../xface -lxface_mcil
#cgo linux LDFLAGS: -Wl,-rpath=./
#include <stdlib.h>
#include "face.h"
#include "../xface/xface_data.h"
extern void go_callback_proxy(int64_t seq, HobotXFaceImageFeatures* result);
*/
import "C"
import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

//如果C中的结构体通过typedef定义名称，在go中调用时直接使用C.xxx,否则，需要C.struct_xxx。
var (
	TypeFile   = 0
	TypeBase64 = 1

	PErrorParameters   = -1
	PErrorFileNotFound = -2
	PErrorNOFeature    = -3

	HOBOT_XFACE_METRIC_LEN   = 256
	HOBOT_XFACE_LANDMARK_LEN = 5
	HOBOT_XFACE_QUALITY_LEN  = 13
)

type Landmark struct {
	Visible int     `json:"visible"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

type FaceFeature struct {
	Rect struct {
		Score float64 `json:"score"`
		X1    float64 `json:"x1"`
		X2    float64 `json:"x2"`
		Y1    float64 `json:"y1"`
		Y2    float64 `json:"y2"`
	} `json:"rect"`
	LivenessScore float64 `json:"liveness_score"`
	QualityScore  float64 `json:"quality_score"`
	Pose          struct {
		Pitch float64 `json:"pitch"`
		Roll  float64 `json:"roll"`
		Yaw   float64 `json:"yaw"`
	} `json:"pose"`
	Metric string `json:"metric"`
	Age    struct {
		Classification int     `json:"classification"`
		Score          float64 `json:"score"`
	} `json:"age"`

	Gender struct {
		Classification int     `json:"classification"`
		Score          float64 `json:"score"`
	} `json:"gender"`
	Glass struct {
		Classification int     `json:"classification"`
		Score          float64 `json:"score"`
	} `json:"glass"`
	Hat struct {
		Classification int     `json:"classification"`
		Score          float64 `json:"score"`
	} `json:"hat"`

	Landmark []Landmark `json:"landmark"`

	Quality struct {
		Brightness int    `json:"brightness"`
		Scores     string `json:"scores"`
	} `json:"quality"`
}

type Response struct {
	ID      string        `json:"id"`
	Cmd     string        `json:"cmd"`
	Result  int           `json:"result"`
	Content []FaceFeature `json:"content"`
}

type Request struct {
	ConnId       uint32
	ReqId        int64
	ID           string `json:"id"`
	Cmd          string `json:"cmd"`
	PredictMode  int    `json:"predict_mode"`
	MaxFaceCount int    `json:"max_face_count"`
	Type         int    `json:"type"`
	Content      string `json:"content"`
}

type XFace struct {
	reqs        map[int64]Request
	writeCh     chan *Request
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	OnCompleted func(connId uint32, resp Response)
}

var XFaceSingleInstance *XFace
var once sync.Once

var modelName = "./models_bit8/model_conf.json"

var confName = "xface.json"

func GetFaceInstance() *XFace {
	once.Do(func() {
		XFaceSingleInstance = &XFace{
			reqs:    make(map[int64]Request),
			writeCh: make(chan *Request),
		}
		XFaceSingleInstance.ctx, XFaceSingleInstance.cancel = context.WithCancel(context.Background())
	})
	return XFaceSingleInstance
}

func (x *XFace) Init() error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	conf := filepath.Join(dir, confName)

	buf := bytes.Buffer{}
	err = x.loadFile(conf, &buf)
	if err != nil {
		return err
	}

	confStr := buf.String()
	cConf := C.CString(confStr)
	cModel := C.CString(modelName)
	os.Chdir(dir)
	ret := C.InitFaceLib(cConf, cModel, (C.Callback)(unsafe.Pointer(C.go_callback_proxy)))
	C.free(unsafe.Pointer(cConf))
	C.free(unsafe.Pointer(cModel))
	if ret != 0 {
		return errors.New(fmt.Sprintf("init xface failed:%d", ret))
	}
	x.wg.Add(1)
	go x.run()
	return nil
}

func (x *XFace) UnInit() {
	x.cancel()
	x.wg.Wait()
	C.UnInitFaceLib()
}

func (x *XFace) DoFeature(r *Request) {
	x.writeCh <- r
}

func (x *XFace) run() {
	defer func() {
		x.wg.Done()
	}()
	for {
		select {
		case req := <-x.writeCh:
			x.doRequest(req)
		case <-x.ctx.Done():
			return
		}
	}
}

func (x *XFace) doRequest(r *Request) {
	buf := bytes.Buffer{}
	var err error
	var result int
	if r.Type == TypeFile {
		err = x.loadFile(r.Content, &buf)
		if err != nil {
			result = PErrorFileNotFound
		}
	} else {
		err = x.decode(r.Content, &buf)
		if err != nil {
			result = PErrorParameters
		}
	}
	r.Content = ""

	if err != nil {
		x.sendErrorResponse(*r, result)
		return
	}
	n := x.feature(r.ReqId, r.PredictMode, r.MaxFaceCount, buf)
	if n != 0 {
		x.sendErrorResponse(*r, n)
		return
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	x.reqs[r.ReqId] = *r
}

func (x *XFace) sendErrorResponse(r Request, result int) {
	resp := Response{
		ID:      r.ID,
		Cmd:     r.Cmd,
		Result:  result,
		Content: nil,
	}
	x.OnCompleted(r.ConnId, resp)
}

func (x *XFace) loadFile(name string, buf *bytes.Buffer) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	tmp := make([]byte, 4096)
	for {
		n, err := f.Read(tmp)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		buf.Write(tmp[:n])
	}
	return nil
}

func (x *XFace) decode(c string, buf *bytes.Buffer) error {
	b, err := base64.StdEncoding.DecodeString(c)
	if err != nil {
		return err
	}
	buf.Write(b)
	return nil
}

func (x *XFace) feature(seq int64, predictMode int, maxFaceCount int, data bytes.Buffer) int {
	b := data.Bytes()
	var t C.int = C.DoFeature(C.int64_t(seq), C.int(predictMode), C.int(maxFaceCount), unsafe.Pointer(&b[0]), C.int(data.Len()))
	return int(t)
}

func (x *XFace) onCallback(seq int64, result int, features []FaceFeature) {
	x.mu.Lock()
	defer x.mu.Unlock()

	if r, ok := x.reqs[seq]; ok {
		resp := Response{
			ID:      r.ID,
			Cmd:     r.Cmd,
			Result:  result,
			Content: features,
		}
		x.OnCompleted(r.ConnId, resp)
	}
}

//export callbackOnCgo
func callbackOnCgo(seq C.int, result *C.HobotXFaceImageFeatures) {
	x := GetFaceInstance()
	if result == nil {
		x.onCallback(int64(seq), PErrorNOFeature, nil)
		return
	}

	if result.error_code_ != 0 {
		x.onCallback(int64(seq), int(result.error_code_), nil)
		return
	}

	features := []FaceFeature{}

	count := int(result.features_count_)

	b := strings.Builder{}

	for i := 0; i < count; i++ {
		f := FaceFeature{}
		ptr := (*C.HobotXFaceFeature)(unsafe.Pointer(uintptr(unsafe.Pointer(result.features_)) + uintptr(C.sizeof_HobotXFaceFeature*C.int(i))))

		f.Rect.X1 = float64(ptr.face_rect_.x1_)
		f.Rect.Y1 = float64(ptr.face_rect_.y1_)
		f.Rect.X2 = float64(ptr.face_rect_.x2_)
		f.Rect.Y2 = float64(ptr.face_rect_.y2_)
		f.Rect.Score = float64(ptr.face_rect_.score_)

		f.LivenessScore = float64(ptr.liveness_score_)
		f.QualityScore = float64(ptr.quality_score_)

		f.Pose.Pitch = float64(ptr.pose_.pitch_)
		f.Pose.Yaw = float64(ptr.pose_.yaw_)
		f.Pose.Roll = float64(ptr.pose_.roll_)

		{
			for k := 0; k < HOBOT_XFACE_METRIC_LEN; k++ {
				m := float64(ptr.metric_[C.int(k)])
				s := strconv.FormatFloat(m, 'g', -1, 32)
				b.WriteString(s)
				b.WriteString(",")
			}
			f.Metric = b.String()
			b.Reset()
		}

		f.Age.Classification = int(ptr.age_.classification_)
		f.Age.Score = float64(ptr.age_.score_)

		f.Gender.Classification = int(ptr.gender_.classification_)
		f.Gender.Score = float64(ptr.gender_.score_)

		f.Glass.Classification = int(ptr.glass_.classification_)
		f.Glass.Score = float64(ptr.glass_.score_)

		f.Hat.Classification = int(ptr.hat_.classification_)
		f.Hat.Score = float64(ptr.hat_.score_)

		{
			for k := 0; k < HOBOT_XFACE_LANDMARK_LEN; k++ {
				land := Landmark{}
				land.X = float64(ptr.landmark_[C.int(k)].x_)
				land.Y = float64(ptr.landmark_[C.int(k)].y_)
				land.Visible = int(ptr.landmark_[C.int(k)].visible_)
				f.Landmark = append(f.Landmark, land)
			}
		}

		{
			for k := 0; k < HOBOT_XFACE_QUALITY_LEN; k++ {
				m := float64(ptr.quality_.scores_[C.int(k)])
				s := strconv.FormatFloat(m, 'g', -1, 32)
				b.WriteString(s)
				b.WriteString(",")
			}
			f.Quality.Scores = b.String()
			b.Reset()
			f.Quality.Brightness = int(ptr.quality_.brightness_classification_)
		}
		features = append(features, f)
	}
	x.onCallback(int64(seq), 0, features)
}
