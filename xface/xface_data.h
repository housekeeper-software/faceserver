/**
 * Copyright (c) 2018 Horizon Robotics. All rights reserved.
 * @file      xface.h
 * @brief provides the fuction to extract face features from one image
 * @author    chuanyi.yang
 * @email     chuanyi.yang@hobot.cc
 * @version   1.0.0.0
 * @date      2018.06.15
 */
#ifndef XFACE_DATA_H
#define XFACE_DATA_H
#include <string.h>

#define HOBOT_XFACE_METRIC_LEN 256  //  加密后长度不变
#define HOBOT_XFACE_AGE_LEN 10
#define HOBOT_XFACE_LANDMARK_LEN 5
#define HOBOT_XFACE_QUALITY_LEN 13
typedef int HobotXFaceStatus;

typedef struct {
  float x1_, y1_;  // 人脸框左上角位置
  float x2_, y2_;  // 人脸框右下角位置
  float score_;    // 得分
} HobotXFaceRect;

typedef struct {
  float pitch_, yaw_, roll_;     // 人脸姿态，分别为俯仰、偏航、旋转角度
} HobotXFacePose;

typedef struct {
  /*
   * the index of the age range
   age range = [[0, 6], [7, 12], [13, 18], [19, 28],
                [29, 35], [36, 45], [46, 55], [56, 100]]
   * */
  int classification_;  // 分类结果 0-7
  float score_;         // 仅浮点年龄模型有置信度输出。
} HobotXFaceAge;

typedef struct {
  int classification_;  // 分类结果 0: female ; 1: male
  float score_;         // 仅浮点模型有置信度输出。
} HobotXFaceGender;

typedef struct {
  int classification_;  // 分类结果 0: no glass ; 1: glass ; 2 : sunglass
  float score_;         // 置信度
} HobotXFaceGlass;

typedef struct {
  int classification_;  // 分类结果 0: without hat ; 1: with hat
  float score_;         // 置信度
} HobotXFaceHat;

typedef struct {
  float x_, y_;        // 位置
  int visible_;        // 是否可见
} HobotXFaceLmk;

typedef struct {
  // quality score array, contains 13 items:
  // blur, eye_abnormal, mouth_abnormal, left_eye, right_eye,  left_brow,
  // right_brow, fore_head, left_cheek, right_cheek, nose, mouth, jaw
  float scores_[HOBOT_XFACE_QUALITY_LEN];  //  置信度 [0,1]
  // 亮度分类结果 0: Normal; 1: Bright; 2: Partly_Bright_Partly_Dark; 3: Dark
  int brightness_classification_;
} HobotXFaceQuality;

typedef struct {
  HobotXFaceRect face_rect_;                 // 人脸框位置
  float liveness_score_;                     // 活体分数
  float quality_score_;                     // 图片质量分数
  HobotXFacePose pose_;                      // 人脸姿态
  float metric_[HOBOT_XFACE_METRIC_LEN];     // 度量特征，128位或者256位
  HobotXFaceAge age_;                        // 年龄
  HobotXFaceGender gender_;                  // 性别
  HobotXFaceGlass glass_;                    // 眼镜
  HobotXFaceHat hat_;                        // 帽子
  HobotXFaceLmk landmark_[HOBOT_XFACE_LANDMARK_LEN];  // 5个关键点和遮挡分数...
  HobotXFaceQuality quality_;                // 图片质量
} HobotXFaceFeature;

typedef enum {
  ErrorCode_OK = 0,                // 正确
  ErrorCode_Uninit = 1,            // 未初始化
  ErrorCode_ParamError = 2,        // 参数错误
  ErrorCode_NoImg = 3,             // 没有图片
  ErrorCode_BadRects = 4,          // 人脸框质量太差
  ErrorCode_RectTooSmall = 5,      // 人脸太小
  ErrorCode_NoRect = 6,            // 没有人脸框
  ErrorCode_NotLiveness = 7,       // 没有活体
  ErrorCode_NoMetric = 8,          // 没有提出特征
  ErrorCode_NoKeypoint = 9,        // 没有提出关键点
  ErrorCode_PoseError = 10,         // 姿态过歪
  ErrorCode_NoModel = 11,           // 模型缺失
  ErrorCode_BadQuality = 12,           // 图片质量太低
  ErrorCode_ConfigError = 13,           // 读取配置文件失败
  ErrorCode_LicenseError = 14,          // 读取授权文件失败
  ErrorCode_LengthTooShort = 15,     // 预分配字符串长度过小

  ErrorCode_Other = 100       // 其他错误
} HobotXFaceErrorCode;

typedef enum {
  PredictMode_None = 0,
  PredictMode_Rect = 1 << 0,        // 人脸框
  PredictMode_Affine = 1 << 1,      // affine转换
  PredictMode_Lmk = 1 << 2,        // 关键点
  PredictMode_Metric = (1 << 3) + PredictMode_Lmk,      // 特征提取

  PredictMode_Liveness = 1 << 4,    // 活体
  PredictMode_Pose = 1 << 5,        // 姿态
  PredictMode_Quality = (1 << 6) + PredictMode_Lmk,     // 图片质量
  PredictMode_Age = (1 << 7) + PredictMode_Lmk,         // 浮点年龄属性模型
  PredictMode_Multi = 1 << 8,       // 定点多属性模型，包括性别、年龄
  PredictMode_Gender = PredictMode_Age,      // 浮点性别属性模型
  PredictMode_Glass = 1 << 10,       // 浮点眼镜属性模型
  PredictMode_Hat = 1 << 11,        // 浮点帽子属性模型
  PredictMode_NormalizeDetect = 1 << 12,  // 只做人脸检测，并输出192*192 只有Y通道的YUV图片
  PredictMode_ImgColor = 1 << 13     // 灰度图/彩色图 检测，只适用于单人脸图片
  // 2的n次方，使用 | 来构造
} HobotXFaceMode;

#ifdef __cplusplus
inline HobotXFaceMode operator|(HobotXFaceMode a, HobotXFaceMode b) {
  return static_cast<HobotXFaceMode>(static_cast<int>(a) | static_cast<int>(b));
}
inline HobotXFaceMode operator&(HobotXFaceMode a, HobotXFaceMode b) {
  return static_cast<HobotXFaceMode>(static_cast<int>(a) & static_cast<int>(b));
}
#endif

typedef struct {
  HobotXFaceErrorCode error_code_;    // 错误码
  HobotXFaceFeature *features_;       // 特征数组
  int features_count_;                // 特征数组长度
  int img_color_;                     //  0 : 灰度图; 1: 彩色图
} HobotXFaceImageFeatures;

typedef enum {
  ImgType_None = 0,
  ImgType_RGB
} HobotXFaceImgType;

// support BMP、DIB、JPEG、JPG、JPE、JFIF、PNG、TIF、TIFF、WEBP，有其他类型图片格式请联系开发者
typedef struct {
  unsigned char *buf_;           // 必传, 图片的二进制流
  int buf_len_;                  // 必传，二进制流长度
  HobotXFaceImgType buf_type_;   // 必传，必须初始化，支持文件流(ImgType_None),和RGB（其他格式需要转换）
  int predict_mode_;             // 必传，必须初始化。模式，含义见HobotXFaceMode说明
  int max_face_count_;           // 必传，人脸数目最大值，当该值为0或负数时，返回所有检测出来的人脸。

  HobotXFaceRect face_rect_;     // 可选, 人脸位置，仅predict_mode_为PredictMode_Rect时需要传入

  int img_w_;      // 可选, 图片宽度，仅类型为ImgType_RGB时需要传入
  int img_h_;      // 可选, 图片宽度，仅类型为ImgType_RGB时需要传入
  float *affine;   // 供内部测试使用，可选, affine参数，仅predict_mode_为PredictMode_Affine时需要传入
} HobotXFaceImage;

#endif  //
