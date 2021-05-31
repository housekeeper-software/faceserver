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
	TypeFile   = 0  //表示请求方提供的是文件绝对位置
	TypeBase64 = 1  //表示请求方提供的是文件base64串

	PErrorParameters   = -1  //请求方的参数错误
	PErrorFileNotFound = -2  //请求方提供的用于人脸特征提取的文件没找到
	PErrorNOFeature    = -3  //没有获得人脸特征，第三方库返回空

	HOBOT_XFACE_METRIC_LEN   = 256
	HOBOT_XFACE_LANDMARK_LEN = 5
	HOBOT_XFACE_QUALITY_LEN  = 13
)

//Landmark 是人脸特征中的一部分
type Landmark struct {
	Visible int     `json:"visible"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

//FaceFeature 人脸特征数据结构
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

//Response 是服务器给客户端请求的应答包
type Response struct {
	ID      string        `json:"id"`  //客户端请求包中的标识
	Cmd     string        `json:"cmd"` //客户端请求的命令
	Result  int           `json:"result"` //请求处理的错误代码，0：表示成功，负数表示服务器自定义错误，其他错误由第三方库返回
	Content []FaceFeature `json:"content"` //人脸特征
}

//Request 是客户端的请求包格式，可以指定文件名或者文件的base64字符串
type Request struct {
	ConnId       uint32 //连接标识
	ReqId        int64  //请求标识
	ID           string `json:"id"`  //客户端请求标识串
	Cmd          string `json:"cmd"` //请求的命令，目前只支持 'feature'，标识提取人脸特征
	PredictMode  int    `json:"predict_mode"` //可选参数，参考第三方文档
	MaxFaceCount int    `json:"max_face_count"` //可选最大提取人脸数目，默认为1
	Type         int    `json:"type"` //指定content字段的内容，0：表示提供的是文件绝对路径，1：表示提供的是文件内容base64串
	Content      string `json:"content"` //根据 type 不同内容不同
}

//XFace 人脸特征提取对象
type XFace struct {
	reqs        map[int64]Request  //请求队列缓存，因为请求是异步的，所以需要缓存
	writeCh     chan *Request   //网络模块通过此通道向本模块写入请求
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	OnCompleted func(connId uint32, resp Response)  //请求处理完成回调接口
}

//XFaceSingleInstance 此对象是单例
var XFaceSingleInstance *XFace
var once sync.Once

//modelName 必须用相对路径，否则库不能正常初始化，这应该是第三方库的bug
var modelName = "./models_bit8/model_conf.json"

//confName 引擎初始化的一些参数，可以改变引擎的工作状态
var confName = "xface.json"

//获取XFace的单例方法
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

//初始化XFace以及第三方库引擎
// @return  可能会返回失败
func (x *XFace) Init() error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	//xface.json应该当在app 的当前目录
	conf := filepath.Join(dir, confName)

	buf := bytes.Buffer{}
	err = x.loadFile(conf, &buf)
	if err != nil {
		return err
	}

	confStr := buf.String()
	cConf := C.CString(confStr)
	cModel := C.CString(modelName)
	//改变当前工作目录
	os.Chdir(dir)
	//调用C方法初始化第三方库引擎
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

//析构XFace 引擎
func (x *XFace) UnInit() {
	x.cancel()
	x.wg.Wait()
	C.UnInitFaceLib()
}

//网络模块通过此方法向引擎申请人脸特征提取
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
			//上层通知要退出
			return
		}
	}
}

//处理客户端请求
func (x *XFace) doRequest(r *Request) {
	buf := bytes.Buffer{}
	var err error
	var result int
	if r.Type == TypeFile {
		//从本地文件中加载照片
		err = x.loadFile(r.Content, &buf)
		if err != nil {
			result = PErrorFileNotFound
		}
	} else {
		//从content 解码照片
		err = x.decode(r.Content, &buf)
		if err != nil {
			result = PErrorParameters
		}
	}
	//Content内容较大，我们不再需要，尽早释放内存
	r.Content = ""

	if err != nil {
		//加载照片失败了，我们需要通知客户端
		x.sendErrorResponse(*r, result)
		return
	}
	//调用C方法处理请求
	n := x.feature(r.ReqId, r.PredictMode, r.MaxFaceCount, buf)
	if n != 0 {
		//引擎返回失败，我们通知客户端
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
